#!/bin/zsh
# simpleserver 多机部署脚本（本地运行，不进服务器）
#
# 用法:
#   ./deploy_all.sh arm | ddeb | ats | hka | lsa | lse | lss | lsu   部署单台
#   ./deploy_all.sh all          依序部署 arm → ddeb → ats → hka → lsa → lse → lss → lsu
#   ./deploy_all.sh verify       只校验所有机器 /healthz revision 是否等于本地 HEAD，不部署
#
# 说明:
#   - 跳板: 本机只直连 arm 和 ddeb；ats/hka/lsa/lse/lss/lsu 都经 ddeb 二跳
#   - 代码同步: 所有机器都在机上 git pull --ff-only（需要各机对 GitHub 有拉取权限；
#     首次部署没有 checkout 时会自动 clone）
#   - 编译: 各机 /usr/local/go/bin/go build -o /root/vps/www/simpleserver .
#   - 静态文件: 只同步 public/*.html → /root/vps/www/
#     （index.html 是各机定制品牌页，故意不覆盖）
#   - 重启: arm 用服务器上的 /root/vps/www/deploy.sh；其余 pkill + nohup
#   - 校验: 部署后 curl /healthz，比对 vcs_revision 是否等于本地 HEAD（3 次重试规避重启抖动）

set -e
cd "$(dirname "$0")"

GO=/usr/local/go/bin/go
REPO=git@github.com:vpsbackup/simpleserver.git
CHECKOUT=/root/go/src/simpleserver
RUNTIME=/root/vps/www
LOCAL_HEAD=$(git rev-parse HEAD)
# 这些机器本机不可达，统一经 ddeb 二跳
DDEB_JUMP_HOSTS=(ats hka lsa lse lss lsu)

# host_domain <host> — 各机 /healthz 域名；只有 ddeb 的域名前缀不同（deb）
host_domain() {
  case "$1" in
  ddeb) print deb.871116.xyz ;;
  *) print "$1.871116.xyz" ;;
  esac
}

log()  { print -P " %F{blue}==>%f $1"; }
good() { print -P " %F{green}✓%f $1"; }
bad()  { print -P " %F{red}✗%f $1"; }

# verify_host <domain> [expect_rev] [via] — via=ddeb 时从 ddeb 上 curl（跳板机本机可能不可达），3 次重试
verify_host() {
  local domain=$1 expect=${2:-$LOCAL_HEAD} via=${3:-local} rev
  for i in 1 2 3; do
    if [[ "$via" == "ddeb" ]]; then
      rev=$(ssh -o ConnectTimeout=10 ddeb "curl -s --max-time 8 https://$domain/healthz" 2>/dev/null |
        python3 -c 'import json,sys; print(json.load(sys.stdin).get("vcs_revision",""))' 2>/dev/null || true)
    else
      rev=$(curl -s --max-time 8 "https://$domain/healthz" 2>/dev/null |
        python3 -c 'import json,sys; print(json.load(sys.stdin).get("vcs_revision",""))' 2>/dev/null || true)
    fi
    if [[ "$rev" == "$expect"* ]]; then
      good "$domain @ ${rev:0:8}"
      return 0
    fi
    sleep 2
  done
  bad "$domain @ ${rev:-无响应} (期望 ${expect:0:8})"
  return 1
}

# 代码同步（三处部署共用）：没有 checkout 就先 clone，然后 cd 进去
checkout_part=$(cat <<EOF
if [ ! -d $CHECKOUT ]; then
  echo "首次部署，clone $REPO → $CHECKOUT"
  git clone $REPO $CHECKOUT
fi
cd $CHECKOUT
EOF
)

# 编译 + 同步静态页 + 重启（ddeb 及各跳板机通用，需先 cd 到 checkout 并完成代码同步）
# 同步 public/*.html 但跳过 index.html（各机定制品牌页）
restart_part=$(cat <<EOF
$GO build -o $RUNTIME/simpleserver .
for f in public/*.html; do
  b=\$(basename \$f)
  [ "\$b" = "index.html" ] && continue
  cp "\$f" "$RUNTIME/\$b"
done
cd $RUNTIME
pkill simpleserver 2>/dev/null || true
sleep 1
nohup ./simpleserver -c ./config.json >>1.txt 2>>2.txt &
sleep 2
pgrep -a simpleserver | head -1
EOF
)

deploy_arm() {
  log "部署 arm (git pull + build + deploy.sh)"
  ssh -o ConnectTimeout=10 arm bash -s <<EOF
set -e
$checkout_part
git pull --ff-only 2>&1 | tail -1
$GO build -o $RUNTIME/simpleserver .
for f in public/*.html; do
  b=\$(basename \$f)
  [ "\$b" = "index.html" ] && continue
  cp "\$f" "$RUNTIME/\$b"
done
$RUNTIME/deploy.sh
EOF
}

deploy_ddeb() {
  log "部署 ddeb (git pull + build + 重启)"
  ssh -o ConnectTimeout=10 ddeb bash -s <<EOF
set -e
$checkout_part
git pull --ff-only 2>&1 | tail -1
$restart_part
EOF
}

# deploy_via_ddeb <host> — 经 ddeb 跳板：机上 git pull + build + 重启（ats/hka/lsa/lse/lss/lsu 通用）
deploy_via_ddeb() {
  local host=$1
  log "部署 $host (经 ddeb 跳板, git pull + build + 重启)"
  ssh -o ConnectTimeout=10 ddeb "ssh -o ConnectTimeout=10 $host bash -s" <<EOF
set -e
$checkout_part
git pull --ff-only 2>&1 | tail -1
$restart_part
EOF
}

case "$1" in
arm)  deploy_arm;  verify_host "$(host_domain arm)" ;;
ddeb) deploy_ddeb; verify_host "$(host_domain ddeb)" ;;
ats | hka | lsa | lse | lss | lsu)
  deploy_via_ddeb "$1"
  verify_host "$(host_domain "$1")" "" ddeb
  ;;
all)
  deploy_arm;  verify_host "$(host_domain arm)"
  deploy_ddeb; verify_host "$(host_domain ddeb)"
  for h in "${DDEB_JUMP_HOSTS[@]}"; do
    deploy_via_ddeb "$h"
    verify_host "$(host_domain "$h")" "" ddeb
  done
  ;;
verify)
  verify_host "$(host_domain arm)" || true
  verify_host "$(host_domain ddeb)" || true
  for h in "${DDEB_JUMP_HOSTS[@]}"; do
    verify_host "$(host_domain "$h")" "" ddeb || true
  done
  print -P " %F{blue}本地 HEAD:%f ${LOCAL_HEAD:0:8}"
  ;;
*)
  print "用法: $0 arm|ddeb|ats|hka|lsa|lse|lss|lsu|all|verify"
  exit 1
  ;;
esac
