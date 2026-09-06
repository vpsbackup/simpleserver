#!/bin/zsh
# simpleserver 多机部署脚本（本地运行，不进服务器）
#
# 用法:
#   ./deploy_all.sh arm | ddeb | ats | hka   部署单台
#   ./deploy_all.sh all                      依序部署 arm → ddeb → ats → hka
#   ./deploy_all.sh verify                   只校验四台 /healthz revision 是否等于本地 HEAD，不部署
#
# 说明:
#   - 代码同步: arm/ddeb/ats 在机上 git pull --ff-only; hka 没有 GitHub 拉取权限,
#     从 ddeb 的 checkout 打 tar 同步（所以 all 模式下 ddeb 必须先于 hka）
#   - 编译: 各机 /usr/local/go/bin/go build -o /root/vps/www/simpleserver .
#   - 静态文件: 只同步 public/agent.html → /root/vps/www/agent.html
#     （index.html 是各机定制品牌页，故意不覆盖）
#   - 重启: arm 用服务器上的 /root/vps/www/deploy.sh；其余 pkill + nohup
#   - 校验: 部署后 curl /healthz，比对 vcs_revision 是否等于本地 HEAD（3 次重试规避重启抖动）

set -e
cd "$(dirname "$0")"

GO=/usr/local/go/bin/go
CHECKOUT=/root/go/src/simpleserver
RUNTIME=/root/vps/www
LOCAL_HEAD=$(git rev-parse HEAD)

log()  { print -P " %F{blue}==>%f $1"; }
good() { print -P " %F{green}✓%f $1"; }
bad()  { print -P " %F{red}✗%f $1"; }

# verify_host <domain> [expect_rev]
verify_host() {
  local domain=$1 expect=${2:-$LOCAL_HEAD} rev
  for i in 1 2 3; do
    rev=$(curl -s --max-time 8 "https://$domain/healthz" 2>/dev/null |
      python3 -c 'import json,sys; print(json.load(sys.stdin).get("vcs_revision",""))' 2>/dev/null || true)
    if [[ "$rev" == "$expect"* ]]; then
      good "$domain @ ${rev:0:8}"
      return 0
    fi
    sleep 2
  done
  bad "$domain @ ${rev:-无响应} (期望 ${expect:0:8})"
  return 1
}

# 编译 + 同步 agent.html + 重启（ddeb/ats/hka 通用，需先 cd 到 checkout 并完成代码同步）
restart_part=$(cat <<EOF
$GO build -o $RUNTIME/simpleserver .
cp public/agent.html $RUNTIME/agent.html
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
cd $CHECKOUT
git pull --ff-only 2>&1 | tail -1
$GO build -o $RUNTIME/simpleserver .
cp public/agent.html $RUNTIME/agent.html
$RUNTIME/deploy.sh
EOF
}

deploy_ddeb() {
  log "部署 ddeb (git pull + build + 重启)"
  ssh -o ConnectTimeout=10 ddeb bash -s <<EOF
set -e
cd $CHECKOUT
git pull --ff-only 2>&1 | tail -1
$restart_part
EOF
}

deploy_ats() {
  log "部署 ats (经 ddeb 跳板, git pull + build + 重启)"
  ssh -o ConnectTimeout=10 ddeb "ssh -o ConnectTimeout=10 ats bash -s" <<EOF
set -e
cd $CHECKOUT
git pull --ff-only 2>&1 | tail -1
$restart_part
EOF
}

deploy_hka() {
  log "部署 hka (经 ddeb 跳板, tar 同步代码 + build + 重启)"
  log "先确保 ddeb checkout 是最新的"
  ssh -o ConnectTimeout=10 ddeb "cd $CHECKOUT && git pull --ff-only 2>&1 | tail -1"
  log "tar 同步 ddeb → hka"
  ssh -o ConnectTimeout=10 ddeb "cd /root/go/src && tar czf - simpleserver" |
    ssh -o ConnectTimeout=10 ddeb "ssh -o ConnectTimeout=10 hka 'tar xzf - -C /root/go/src/'"
  log "编译并重启 hka"
  ssh -o ConnectTimeout=10 ddeb "ssh -o ConnectTimeout=10 hka bash -s" <<EOF
set -e
cd $CHECKOUT
$restart_part
EOF
}

case "$1" in
arm)  deploy_arm;  verify_host arm.871116.xyz ;;
ddeb) deploy_ddeb; verify_host deb.871116.xyz ;;
ats)  deploy_ats;  verify_host ats.871116.xyz ;;
hka)  deploy_hka;  verify_host hka.871116.xyz ;;
all)
  deploy_arm;  verify_host arm.871116.xyz
  deploy_ddeb; verify_host deb.871116.xyz
  deploy_ats;  verify_host ats.871116.xyz
  deploy_hka;  verify_host hka.871116.xyz
  ;;
verify)
  for d in arm.871116.xyz deb.871116.xyz ats.871116.xyz hka.871116.xyz; do
    verify_host "$d" || true
  done
  print -P " %F{blue}本地 HEAD:%f ${LOCAL_HEAD:0:8}"
  ;;
*)
  print "用法: $0 arm|ddeb|ats|hka|all|verify"
  exit 1
  ;;
esac
