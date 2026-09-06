# simpleserver

一个单二进制的个人多合一站点服务：留言板、文件上传与管理、AI 代理、DNS 工具、
机器状态页、IP 归属地、TCP/UDP echo 等，数据存 MongoDB，可选站点密码登录。
**部署形态**：nginx 终结 TLS 并反代到本服务的 10080 端口；QUIC（HTTP/3）由本服务
直接监听 udp/443；`/dl/` 下的上传文件由静态文件服务器直接伺服。

## 页面

| 路径 | 说明 | 访问 |
|---|---|---|
| `/` | 主页（各机器页脚品牌不同，`index.html` 不随部署脚本同步） | 公开 |
| `/dilfish.html` | 站点私有入口：登录 / 工具导航 | 公开（写操作需登录） |
| `/t` | 留言 / 临时记事板（卡片列表、新增、删除、拷贝） | 读公开，写需登录 |
| `/upload` | 上传 + 文件管理合一页（进度条、文件列表、删除、拷贝链接） | 页面公开，操作需登录 |
| `/dns.html` | DNS 工具：递归解析 + `dig +trace` 式迭代追踪 | 公开 |
| `/status.html` | 机器状态：主机/CPU/内存/磁盘/流量/TLS 证书到期（30s 自动刷新） | 公开 |
| `/agent.html` | AI 助手（对话与模型代理，依赖 `/v1/agentproxy/`） | 公开，代理需登录 |

## API

均走 `/api/` 前缀（见 `api.go`），除注明外 GET 返回 JSON、写操作为 POST + JSON：

| 端点 | 说明 | 鉴权 |
|---|---|---|
| `GET /api/t/list`、`POST /api/t`、`POST /api/t/delete` | 留言板增删查 | 读公开 / 写需登录 |
| `GET /api/files/list`、`POST /api/files/delete` | 上传目录文件列表 / 删除 | 需登录 |
| `GET /api/dns/query?name=&type=&resolver=` | 单次 DNS 递归查询 | 公开 |
| `GET /api/dns/trace?name=&type=` | DNS 迭代追踪（根 → TLD → 权威） | 公开 |
| `GET /healthz` | 存活 + 版本 + 主机/内存/磁盘/流量/证书到期（探针友好，匿名） | 公开 |

需要鉴权的接口支持两种方式：登录 cookie（`/dilfish.html` 登录获得），或请求头
`X-Site-Token: <auth_password>`（curl 友好）。未登录的浏览器写请求会 302 到
`/dilfish.html`。

## 静态文件与目录列表策略

`static_dir` 指向的目录由 `staticNoDirFS`（见 `staticfs.go`）伺服：

- **永远不会生成目录列表**：访问没有 `index.html` 的目录返回 404（如 `/dl/`、
  `/pass/`），防止上传目录被匿名浏览；
- 有 `index.html` 的目录（含站点根 `/`）正常返回该页；
- 普通文件请求（如 `/dl/<随机名>.txt`、`/agent.html`）不受影响。

因此往 `static_dir` 下放任何新目录都是安全的；要做一个可浏览的目录页，放一个
`index.html` 进去即可。上传文件靠随机文件名 + 目录不可枚举 + 管理器鉴权三层配合。

## 配置

启动：`simpleserver -c config.json`，全部字段见 `sample.config.json` 和
`config.go` 里的字段注释。常用：

- `port` HTTP/1-2 监听端口（默认 10080，nginx 反代到这里）
- `domain` 对外域名（出现在页面文案里，并用于 HTTP/3 SNI 校验）
- `msg` + `mongo_uri/mongo_db/mongo_coll` 留言板（MongoDB）
- `upload_dir` 上传存储目录（下载路径固定为 `/dl/<随机名>`）
- `auth_password` / `cookie_pass` 站点密码与 cookie 签名密钥（两者都填才启用鉴权）
- `use_quic` / `quic_cert_path` HTTP/3（证书同时是 status 页证书到期检查的来源）
- `static_dir` 静态根（**不要指向仓库根**；服务器上为 `./`，即运行目录）

## 开发与测试

```sh
go build ./...
go test -count=1 ./...   # 全部离线，不打真实网络
```

## 部署

四台机器（arm / ddeb / ats / hka），本地一键：

```sh
./deploy_all.sh all      # 依序 arm → ddeb → ats → hka
./deploy_all.sh arm      # 单台
./deploy_all.sh verify   # 只读校验四台 revision 是否等于本地 HEAD
```

注意：arm/ddeb/ats 在机上 `git pull`；hka 无 GitHub 拉取权限，从 ddeb tar 同步
（所以 `all` 模式下 ddeb 先于 hka）。`public/*.html` 会同步到服务器运行目录，
但 `index.html` 例外——各机器页脚品牌不同，是按机器定制的。
