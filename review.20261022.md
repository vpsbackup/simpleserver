# simpleserver 代码审查（2026-10-22）

- 审查范围：仓库内全部 Go 代码（约 7300 行，不含 `public/` 静态资源）
- 审查方式：人工通读 + `go vet` / `staticcheck` / `golangci-lint` + `go test -cover`
- 当时基线：`df368fe`（memfile 与部署脚本改造合并后）
- 结论：单二进制、单用户的定位下整体可用，主要风险集中在**上传链路的资源与并发正确性**、
  **未鉴权端点的放大能力**、以及**日志无限增长**。本轮先修了最影响可用性的两项（#1、#5），
  另外修了一个测试暴露出来的缓存竞争 bug。
后续跟进：`50054c1` 修复 #2/#3/#4，`dcff4dd` 修复 #12/#13，2026-10-22 晚间再修复 #6 与 #8
  （详见下文各条），`f63dbff` 修复 #15，2026-10-23 修复 #10 与 #11，
  `simpleserver.gz` 残留文件已删除（#26）。

状态说明：✅ 已修复；⬜ 待处理。

---

## 一、本轮已修复

### ✅ #10 登录失败退避 + 写操作 Origin 校验（2026-10-23）

位置：`auth.go`、`cmux.go`

修复：

- 新增 `loginGuard`：按客户端 IP 统计连续失败次数，前 5 次正常提示，第 6 次起
  锁定 30s 并按次翻倍，封顶 15 分钟；锁定期内即使密码正确也返回 429 +
  `Retry-After`；登录成功清零计数；空闲 1 小时的条目自动回收；
- IP 取自 `X-Real-Ip`（`cmux` 在直连时会用 `RemoteAddr` 覆盖，无法伪造），
  回退到 `RemoteAddr`；
- 新增 `originAllowed`：非 GET/HEAD/OPTIONS 请求若带 `Origin`，其 host 必须等于
  `r.Host` 或 `Cfg.Domain`；`Sec-Fetch-Site: cross-site` 一律拒绝；无 `Origin`
  的 curl / API 调用不受影响。校验统一在 `MuxService.ServeHTTP` 入口执行，
  新增写路由自动获得保护。

### ✅ #11 `needsAuth` 改为默认拒绝（2026-10-23）

位置：`auth.go`

修复：GET/HEAD/OPTIONS 默认公开，只有 `protectedReadPath`（`/api/files/`、
`/tracer`）需要登录；其余方法默认需要鉴权，只有 `publicWritePath` 显式列出的
`/login`、`/logout` 放行；`/v1/agentproxy/` 仍按原逻辑（除 OPTIONS 外全保护）。
新增 `TestNeedsAuthDefaultsToDenyWrites`、`TestLoginFailureBackoff`、
`TestLoginBackoffExpiresAndResets`、`TestOriginAllowed`、
`TestMuxRejectsCrossOriginWrite`。

### ✅ #2/#3/#4 上传链路：配额、流式落盘与失败清理（`50054c1`）

- #2 配额按实际写入字节计，预留/提交/回滚全在 `u.Lock` 内，并发上传无法竞争或绕过总配额
  （配额不再是客户端声明的 `header.Size`）；新增 `upload_quota_test.go`。
- #3 `Handler` 改 `MultipartReader` 流式直写，不落 `/tmp`；`max_single_file_mb` 改为单文件硬上限。
- #4 任何中途失败删除文件、清 Map、回滚已入账字节。
- 注：并发配额用例的"恰好 4 个成功"断言过强（流式预留窗口重叠使成功数随调度浮动），
  已改为校验不变量（`Curr ≤ 配额`、`Curr == 磁盘用量`、无残留），带 `-race` 稳定通过。

### ✅ #6 上传错误路径返回正确状态码（2026-10-22 晚）

位置：`upload_lib.go` `Handler`

修复：非 POST 返回 405；multipart 解析失败返回 400 `bad upload request`；
配额超限与单文件超限返回 413（固定文案 `upload quota exceeded` /
`file exceeds per-file size limit`）；其他写失败返回 500 `upload failed`；
缺少 file 部件返回 400 `no file part`。内部错误字符串（含路径）、配额计数不再回显客户端。

### ✅ #8 /healthz 缓存 + 敏感字段鉴权门控（2026-10-22 晚）

位置：`healthz.go`

修复：

- 全量采集（host/cpu/mem/disk/证书/vnstat）结果缓存 30s（`healthzCacheTTL`），
  `vcsRevision()` 改为启动时读一次（`cachedVCSRevision`），`Go` 运行时信息保持每请求更新；
- 匿名请求不再返回 `host`/`disks`/`cert`（hostname、内核、磁盘用量、证书路径与错误），
  登录 cookie 或 `X-Site-Token` 鉴权后返回完整快照；探针关心的 `ok`/`uptime`/`vcs_revision` 匿名可见，
  `deploy_all.sh verify` 读取的 `vcs_revision` 不受影响；
- 新增 `TestHealthzCacheAndRedaction` 覆盖匿名裁剪、鉴权可见与 TTL 缓存行为。

### ✅ #12/#13 /dl/ 下载语义与 agentproxy 收紧（`dcff4dd`）

- #12 `/dl/` 强制 `Content-Disposition: attachment` + `X-Content-Type-Options: nosniff`，
  上传的 `.html`/`.svg` 不再在本站源可执行；
- #13 agentproxy：共享 Transport 带拨号/TLS/响应头超时，上游端口仅放行 443（回环豁免），
  CORS 收紧为同站域，`ErrorHandler` 不再回显内部错误细节。

### ✅ #26 `simpleserver.gz` 已删除

### ✅ #1 TCP echo 一次只能服务一个连接（P0）

位置：`tcp.go`（旧实现 `handleTcp(conn)` 在 `Accept` 循环里同步调用）

问题：任意一个客户端连上后不发数据，`handleTcp` 里的 `conn.Read` 永久阻塞，
整个 `Accept` 循环停摆，10082 端口后续连接全部排队。默认配置 `tcp_port = 10082`
是开启的，所以线上一直存在。

修复：

- 每个连接 `go handleTcp(conn)`，accept 循环不再被单个连接占用；
- 用带缓冲 channel 做信号量，限制并发连接 512，超出直接 close；
- 每条连接设置 60s 读/写空闲超时，读超时属于正常回收，不记日志；
- `Accept` 遇到 `net.ErrClosed` 正常返回，读 buffer 移到循环外复用；
- 抽出 `ServeTCP(ls *net.TCPListener)` 便于测试。

新增 `tcp_test.go`：echo 正确性、空闲连接不阻塞其他连接（回归用例）、
空闲超时回收、连接数打满时丢弃。

### ✅ #5 HTTP 超时、优雅退出与退出码（P0）

位置：`main.go`、`quic.go`

问题：

- `http.ListenAndServe` 没有任何超时，而 HTTP/3 直接监听 udp/443 对公网；
- 没有信号处理，重启时在途请求被硬断；
- `InitMux` 失败时 `return`，进程退出码为 0，巡检/systemd 认为启动成功。

修复：

- `http.Server` 增加 `ReadHeaderTimeout = 10s`、`IdleTimeout = 90s`（大于 nginx
  默认 `keepalive_timeout` 75s）。**刻意不设 `ReadTimeout`/`WriteTimeout`**：
  100MB 上传和 SSE 流式代理会被直接掐断；
- `signal.NotifyContext(SIGINT, SIGTERM)` + `srv.Shutdown(10s)`，超时再 `Close()`。
  部署脚本里的 `pkill` 默认就是 SIGTERM，现在会先排空在途请求；
- `InitMux` 失败、任一听者异常退出（端口占用、证书损坏）都以退出码 1 结束；
- `quic.go` 从 `http3.ListenAndServeQUIC`（出错 panic 在 goroutine 里）改为
  `NewHTTP3Server()` 预加载证书并返回 `*http3.Server`，由 main 统一启动与关闭；
- 配置校验补充：`quic_only` 必须配合 `use_quic`，否则两个 listener 都不启动、进程空转。

新增 `quic_test.go`：证书缺失时报错、正常证书下 `Addr`/`TLSConfig`/`NextProtos` 正确。
进程级验证见文末。

### ✅ 附加：md5 缓存预热与新上传竞争

位置：`upload_lib.go`（`warmMD5Cache`）

现象：`TestWriteFileWarmsMD5CacheAndDeleteClears` 偶发失败（约在 20 次全量测试中出现一次）。

原因：启动时或上传后触发的后台预热可能在文件写到一半时取到旧的 size/mtime，
算完哈希后把结果写回缓存，覆盖掉上传路径刚写入的正确条目。

修复：哈希完成后重新 `Stat`，大小或 mtime 与开始时不一致则丢弃该结果，
交给上传路径自己写缓存。`go test -count=5` 与 `-race` 均稳定通过。

---

## 二、待处理

（#2/#3/#4/#6/#8/#12/#13/#26 已修复，移入第一节。）

### P0 —— 建议下一轮就做

（原 P0 四项已全部修复，无待处理项。）

### P1 —— 安全面

#### ⬜ #7 `/api/dns/query` 的 `resolver` 参数可指向任意 IP

位置：`dns.go` `DnsQueryHandler`

只做了 `net.ParseIP` 校验，匿名用户可让服务器向任意地址的 53 端口发查询，
可被用于内网探测或当作开放 DNS 转发器。`/api/dns/trace` 单请求最长 12s，也没有限速。

建议：resolver 改为配置白名单（或删除该参数）；两个 DNS 接口加按 IP 的速率限制与结果缓存。

#### ✅ #8 `/healthz` 匿名且每次全量采集（已修复，见第一节）

位置：`healthz.go` `HealthzHandler`

每次请求都会执行 gopsutil 的 host/cpu/mem/disk、读取证书文件、调用一次
`debug.ReadBuildInfo`。公开端点 + 高频探针 = 持续开销；同时暴露 hostname、内核版本、
磁盘用量、证书路径与错误信息。

建议：结果缓存 15–30s（vnstat 已有 1 分钟缓存，可参照 `vnstatCacheTTL`），
`vcsRevision()` 启动时算一次；敏感字段只在鉴权后返回。

#### ⬜ #9 `/ip` 把请求头原样回显，包括 Cookie

位置：`cfip.go` `CFIPHandler`

遍历 `r.Header` 全量输出，`Cookie`、`Authorization` 也会出现在页面/JSON 里
（自己的凭证，分享截图即泄漏）。`auth.go` 已有 `redactHeaders`，这里没用上。
另外 `RenderHTML` 每个请求都 `template.ParseFiles`。

建议：用 `redactHeaders` 过滤敏感头；模板启动时解析一次（或 `//go:embed`）。

#### ✅ #10 登录没有失败限速（已修复，见第一节）

位置：`auth.go` `LoginHandler`

`passwordMatch` 是常数时间比较，但没有失败计数/退避，可无限在线爆破。
POST 全靠 `SameSite=Lax` 挡 CSRF。

建议：按 IP 做失败退避/锁定；写操作统一校验 `Origin`。

#### ✅ #11 `needsAuth` 是白名单，新增路由默认无保护（已修复，见第一节）

位置：`auth.go` `needsAuth`

目前是“列出来的才鉴权”，以后新增一个 `POST /api/xxx` 忘记登记就直接裸奔。

建议：改为“非 GET 默认需要鉴权，显式列公开例外”。

#### ✅ #12 `/dl/` 上传文件同源伺服，可造成存储型 XSS（已修复，见第一节）

上传 `.html` / `.svg` 后访问 `/dl/<name>` 即在本站域下执行脚本。上传接口虽需登录，
但与 cookie 鉴权同源，仍是可被利用的提权面（当前是否 inline 取决于 nginx 配置，需现场确认）。

建议：对 `/dl/` 强制 `Content-Disposition: attachment` + `X-Content-Type-Options: nosniff`，
或把上传域名与主站分离。

#### ✅ #13 agentproxy 上游调用无超时（已修复，见第一节）

位置：`agentproxy.go` `AgentProxy`

使用默认 `http.Transport`，没有 `ResponseHeaderTimeout`/`DialContext` 超时；
每个请求新建一次 `ReverseProxy`；`ErrorHandler` 把内部错误文本直接回给客户端；
CORS 为 `*`。

建议：复用带超时的 Transport，限制上游端口为 443，错误信息结构化且不回显细节。

### P2 —— 性能与可运维性

#### ⬜ #14 留言列表无分页，`MsgShow` 用 RequestURI 取 id

- `msg.go` `ApiMsgList`：`FindSort` 没有 limit/skip，Mongo 侧也没有 `createAt` 索引，
  数据量上去后是全表扫描 + 全量内存；
- `msg.go` `MsgShow` 用 `r.RequestURI[len("/t/list/"):]`，`/t/list/<id>?x=1` 会查不到；
- `view.go` `Handle` 同样用 `RequestURI`，`/ip/1.2.3.4?x=1` 解析失败。

建议：加 limit/skip 与总数、建索引；路径解析统一用 `r.URL.Path`（与 memfile 修复一致）。

#### ✅ #15 Mongo 查询没有超时（已修复）

位置：`mongo.go`

`Find` / `FindSort` / `Del` 都用 `context.Background()`，Mongo 一慢就挂死 handler。
`mongo.NewClient` + `client.Connect` 在驱动 1.17 已废弃。

已修复（`f63dbff`）：连接改为 `mongo.Connect` + `Ping`（5s），每个操作
`context.WithTimeout`（`mongoOpTimeout` = 3s），慢查询不再无限占用 handler。

#### ⬜ #16 日志量是当前最大的运维风险

- 每请求 2 行 + 整个 header map（`cmux.go`）
- UDP echo 每个包 2 行（`udp.go`）、TCP 每次读 1 行（旧 `tcp.go`，本轮已去掉）
- 归属地未命中每次一条（`range_tree.go`、`bsearch.go`）——IPv6 访客几乎每次请求都命中
- 部署脚本 `>>1.txt 2>>2.txt`，没有轮转

建议：默认只记错误、访问日志合并为单行 JSON 或采样、加 `logrotate`/lumberjack。

#### ⬜ #17 三个页面的大段 HTML 内联在 Go 字符串里

`auth.go`（登录页）、`page_msg.go`、`upload_lib.go`（上传页）。

带来两个具体问题：

1. 无法格式化/lint，改样式要动 Go 字符串；
2. 文案与配置漂移：上传页写死“单次最大 10M · 累积 10G”，配置默认是 100MB / 10GB；
   登录页页脚写死 `ARM.871116.XYZ`，其余 7 台机器的品牌是错的。

建议：`//go:embed public/*.html` + `html/template`，文案从 `Cfg` 注入；
页脚品牌跟随 `Cfg.Domain` 或每机 `index.html` 的约定。

#### ⬜ #18 `MaxSingleFileMB` 一个字段承担两种语义

位置：`config.go` → `upload_lib.go` 的 `ParseMultipartForm`

既被当作单文件上限（实际未生效），又被当作 multipart 内存上限（100MB × 并发
会打爆 1GB 内存的机器）。建议拆成 `max_single_file_mb` 与 `multipart_memory_mb`（默认 8–16MB）。

### P3 —— 代码质量与工程化

| 编号 | 状态 | 内容 |
|---|---|---|
| #19 | ⬜ | `staticcheck` 9 条小问题：`mongo.NewClient`/`Connect` 已废弃、`vnstat.go` 中 `rxtx > math.MaxInt64` 恒假、多处冗余 `return`、`302` 应写 `http.StatusFound` |
| #20 | ✅ | `agentproxy.go`、`udp.go` 已 `gofmt -w`（`gofmt -l .` 全库干净） |
| #21 | ✅（部分） | 新增 `udp_test.go`（echo 往返 + Close 退出）、`TestAuthCookieExpiry`（过期/篡改/垃圾/缺失 cookie）；上传配额并发与 healthz 用例已随 #2/#8 修复补过；覆盖率 45.4% → 50.5% |
| #22 | ✅ | `memFileDo` 改为返回 `{Status, Header, Body}`，不再返回 Body 已关闭的 `*http.Response`（bodyclose 不再报警） |
| #23 | ⬜ | `tracer.go` 是空壳：`FlightRecorder` 启动了但没人读，`/tracer` 只回 `ok`。要么实现快照下载，要么删除 |
| #24 | ⬜ | `data.go` + `data6.go` 共 1137 行生成数据，建议挪到 JSON + `//go:embed` 或加 `go:generate` |
| #25 | ⬜ | `deploy_all.sh`：`verify_host` 用前缀匹配且依赖本地 HEAD（本地提交未 push 时会全部误报，本轮部署已实际踩到）；`all` 遇错即停会留下半套部署；1GB 机器建议 `GOFLAGS=-p=2`；重启判定用 `sleep 2 + pgrep`，不如轮询 `/healthz` |
| #26 | ✅ | 仓库遗留未跟踪文件 `simpleserver.gz`，已删除 |
| #27 | ⬜ | 没有 CI（无 `.github/`），`go vet + go test -count=1 ./...` 一个 workflow 即可覆盖 |
| #28 | ⬜ | 没有 systemd unit，靠 `nohup`；机器重启后服务不会自动拉起 |

---

## 三、本轮验证记录

```
go build ./...                 BUILD_OK
go vet ./...                   VET_OK
go test -count=1 ./...         ok   simpleserver   0.906s
go test -count=5 ./...         ok（确认 md5 用例不再偶发失败）
go test -count=1 -race ./...   ok（顺带发现并修掉了 tcp 测试里的全局变量竞争）
gofmt -l                       改动文件全部干净
GOOS=linux GOARCH=amd64 build  OK
```

进程级验证（临时配置，未触碰线上）：

| 场景 | 结果 |
| --- | --- |
| `GET /healthz`、`GET /agent.html` | 200 |
| `kill -TERM` | 退出码 0，日志 `shutdown signal received, draining connections` |
| 配置非法（`port = 70000`） | 退出码 1 |
| `static_dir` 不存在 | 退出码 1 |
| 端口被占用 | 退出码 1，日志 `serve error: ... address already in use` |

## 四、建议的后续顺序

1. ~~#2 + #3 + #4 + #6：上传链路~~（✅ 已完成）
2. ~~#11 + #10：鉴权默认拒绝、登录失败退避与 Origin 校验~~（✅ 已完成），剩 #7 DNS resolver 白名单；
3. #16：日志收敛；
4. #17 + #18：模板外置 + 配置项拆分，为后续改 UI 铺路；
5. #19–#28：随日常维护逐步清理，优先补 CI 与 systemd。
