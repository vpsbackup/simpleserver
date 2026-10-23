package main

import (
	"crypto/sha512"
	"crypto/subtle"
	"fmt"
	"html"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	dnet "github.com/dilfish/tools/net"
)

const (
	authCookieName = "ss_auth"
	authCookieKey  = "user"
)

var authCookieMgr *dnet.CookieManager

func initAuth(cfg Config) {
	authCookieMgr = nil
	if cfg.AuthPassword == "" {
		return
	}
	exp := cfg.AuthExpireS
	if exp < 60 {
		exp = 60
	}
	authCookieMgr = dnet.NewCookieManager(cfg.CookiePass, "", exp)
}

func authEnabled() bool {
	return Cfg.AuthPassword != "" && authCookieMgr != nil
}

func calcAuthCookieValue(pass, cookieKey string, unix int64) string {
	ts := strconv.FormatInt(unix, 10)
	sum := sha512.Sum512([]byte(cookieKey + pass + ts))
	return ts + "-" + fmt.Sprintf("%x", sum)
}

func setAuthCookie(w http.ResponseWriter, r *http.Request) {
	exp := Cfg.AuthExpireS
	if exp < 60 {
		exp = 60
	}
	expireAt := time.Now().Unix() + exp
	c := &http.Cookie{
		Name:     authCookieName,
		Value:    calcAuthCookieValue(Cfg.CookiePass, authCookieKey, expireAt),
		Path:     "/",
		Expires:  time.Unix(expireAt, 0),
		HttpOnly: true,
		Secure:   cookieSecure(r),
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, c)
}

func clearAuthCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cookieSecure(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func cookieSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	proto := strings.ToLower(r.Header.Get("X-Forwarded-Proto"))
	return proto == "https"
}

func passwordMatch(got string) bool {
	want := Cfg.AuthPassword
	if want == "" {
		return false
	}
	if len(got) != len(want) {
		// still compare to keep timing closer
		subtle.ConstantTimeCompare([]byte(got), []byte(want))
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func bearerPasswordOK(r *http.Request) bool {
	// Prefer dedicated header so /v1/agentproxy can still forward Authorization to upstream.
	if passwordMatch(strings.TrimSpace(r.Header.Get("X-Site-Token"))) {
		return true
	}
	h := r.Header.Get("Authorization")
	if h == "" {
		return false
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return false
	}
	token := strings.TrimSpace(h[len(prefix):])
	// Only treat Bearer as site auth when it matches the site password.
	// Upstream API keys will not match and must rely on login cookie instead.
	return passwordMatch(token)
}

func cookieAuthorized(r *http.Request) bool {
	if authCookieMgr == nil {
		return false
	}
	return authCookieMgr.CheckCookie(r, authCookieName, authCookieKey) != ""
}

// Authorized reports whether the request may perform protected write/proxy actions.
func Authorized(r *http.Request) bool {
	if !authEnabled() {
		return true
	}
	return cookieAuthorized(r) || bearerPasswordOK(r)
}

// needsAuth defaults to deny: every non-idempotent method needs a login unless
// the path is an explicit public exception. Read-only requests stay public so
// the site, the status page and the health probe keep working anonymously.
// This replaces the old allow-list, where a new POST route was unprotected
// until someone remembered to register it (review #11).
func needsAuth(r *http.Request) bool {
	if !authEnabled() {
		return false
	}
	path := r.URL.Path

	// The proxy forwards credentials upstream, so every method is protected;
	// only the CORS preflight is public.
	if strings.HasPrefix(path, "/v1/agentproxy/") {
		return r.Method != http.MethodOptions
	}

	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return protectedReadPath(path)
	default:
		return !publicWritePath(path)
	}
}

// protectedReadPath lists the read-only endpoints that still expose private
// data (file names/sizes, future trace snapshots) and therefore need a login.
func protectedReadPath(path string) bool {
	switch {
	case path == "/api/files/list", strings.HasPrefix(path, "/api/files/"):
		return true
	case path == "/tracer":
		return true
	default:
		return false
	}
}

// publicWritePath is the explicit exception list for non-GET requests.
// /login must stay public (it is how a session is created) and /logout only
// expires a cookie, so it is harmless without a session.
func publicWritePath(path string) bool {
	switch path {
	case "/login", "/logout":
		return true
	default:
		return false
	}
}

func rejectUnauthorized(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	secFetch := r.Header.Get("Sec-Fetch-Mode")
	if r.Method == http.MethodGet || strings.Contains(accept, "text/html") || secFetch == "navigate" {
		http.Redirect(w, r, "/dilfish.html", http.StatusFound)
		return
	}
	http.Error(w, "unauthorized: login at /dilfish.html first", http.StatusUnauthorized)
}

// --- login failure backoff (review #10) ---

const (
	// loginFailFree is how many consecutive failures are answered normally
	// before locking starts, so a user who mistypes a couple of times is fine.
	loginFailFree = 5
	// loginLockBase doubles per extra failure up to loginLockMax.
	loginLockBase       = 30 * time.Second
	loginLockMax        = 15 * time.Minute
	loginAttemptIdleTTL = time.Hour
)

type loginAttempt struct {
	fails       int
	lockedUntil time.Time
	lastSeen    time.Time
}

type loginGuard struct {
	mu      sync.Mutex
	entries map[string]*loginAttempt
	now     func() time.Time
}

func newLoginGuard() *loginGuard {
	return &loginGuard{entries: make(map[string]*loginAttempt), now: time.Now}
}

var loginLimiter = newLoginGuard()

// retryAfter reports how long the caller must wait before another attempt.
func (g *loginGuard) retryAfter(key string) time.Duration {
	g.mu.Lock()
	defer g.mu.Unlock()
	a := g.entries[key]
	if a == nil {
		return 0
	}
	if d := a.lockedUntil.Sub(g.now()); d > 0 {
		return d
	}
	return 0
}

// fail records one failed attempt and returns the lock duration (0 when the
// caller may retry immediately) plus the consecutive failure count.
func (g *loginGuard) fail(key string) (time.Duration, int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.now()
	g.gcLocked(now)
	a := g.entries[key]
	if a == nil {
		a = &loginAttempt{}
		g.entries[key] = a
	}
	a.fails++
	a.lastSeen = now
	if a.fails <= loginFailFree {
		return 0, a.fails
	}
	lock := loginLockBase
	for n := a.fails - loginFailFree; n > 1 && lock < loginLockMax; n-- {
		lock *= 2
	}
	if lock > loginLockMax {
		lock = loginLockMax
	}
	a.lockedUntil = now.Add(lock)
	return lock, a.fails
}

// succeed drops the failure history so a valid password resets the backoff.
func (g *loginGuard) succeed(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.entries, key)
}

// gcLocked drops idle entries so a stream of unique source addresses cannot
// grow the map forever.
func (g *loginGuard) gcLocked(now time.Time) {
	for k, a := range g.entries {
		if now.Sub(a.lastSeen) > loginAttemptIdleTTL && !now.Before(a.lockedUntil) {
			delete(g.entries, k)
		}
	}
}

// clientIP prefers the address set by the front proxy in ServeHTTP. Behind
// nginx X-Real-Ip is trusted; when the server faces the internet directly,
// cmux overwrites X-Real-Ip with RemoteAddr, so it cannot be spoofed.
func clientIP(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-Real-Ip")); v != "" {
		return v
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// originAllowed rejects state-changing requests that a browser marks as
// coming from another site (review #10: SameSite=Lax alone is not enough).
// Non-browser clients (curl, API integrations) send no Origin and keep working.
func originAllowed(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	if strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site") {
		return false
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	originHost := hostWithoutPort(u.Host)
	for _, candidate := range []string{r.Host, r.Header.Get("X-Forwarded-Host"), Cfg.Domain} {
		if candidate != "" && originHost == hostWithoutPort(candidate) {
			return true
		}
	}
	// A reverse proxy that does not preserve Host leaves a loopback/private
	// address here, so comparing hosts would reject every legitimate login.
	// Fall back to the browser signal we already checked above.
	if isInternalHost(r.Host) {
		return true
	}
	return false
}

func isInternalHost(h string) bool {
	host := hostWithoutPort(h)
	if host == "" || host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified())
}

func hostWithoutPort(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	if host, _, err := net.SplitHostPort(h); err == nil {
		return strings.Trim(host, "[]")
	}
	return strings.Trim(h, "[]")
}

// DilfishHandler is the obscure login page at /dilfish.html.
func DilfishHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(`<!doctype html>
<html lang="zh-cmn-Hans">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
  <meta name="robots" content="noindex,nofollow">
  <meta name="theme-color" content="#fafaf9">
  <title>dilfish</title>
  <link href="/302/bootstrap.css" rel="stylesheet">
  <style>
    :root {
      --bg: #fafaf9;
      --card: #ffffff;
      --line: #e7e5e4;
      --text: #1c1917;
      --muted: #78716c;
      --accent: #2563eb;
      --accent-text: #ffffff;
      --ok: #16a34a;
      --danger: #b91c1c;
      --btn-bg: #f5f5f4;
    }
    * { box-sizing: border-box; }
    html, body { height: 100%; }
    body {
      margin: 0;
      min-height: 100dvh;
      color: var(--text);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
      background: var(--bg);
    }
    .page {
      min-height: 100dvh;
      display: flex;
      flex-direction: column;
      padding: max(20px, env(safe-area-inset-top)) 20px max(20px, env(safe-area-inset-bottom));
    }
    .topbar {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      margin-bottom: 24px;
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 10px;
      text-decoration: none;
      color: var(--text);
      font-weight: 650;
      letter-spacing: 0.02em;
    }
    .brand-mark {
      width: 34px;
      height: 34px;
      border-radius: 10px;
      display: grid;
      place-items: center;
      background: var(--accent);
      color: var(--accent-text);
      font-size: 15px;
    }
    .brand-sub {
      display: block;
      font-size: 12px;
      color: var(--muted);
      font-weight: 500;
      margin-top: 2px;
    }
    .home-link {
      color: var(--muted);
      text-decoration: none;
      font-size: 14px;
      padding: 8px 12px;
      border-radius: 999px;
      border: 1px solid var(--line);
      background: var(--btn-bg);
    }
    .home-link:hover { color: var(--text); border-color: #d6d3d1; }
    .stage {
      flex: 1;
      display: grid;
      place-items: center;
    }
    .card {
      width: min(440px, 100%);
      background: var(--card);
      border: 1px solid var(--line);
      border-radius: 20px;
      box-shadow: 0 1px 2px rgba(0,0,0,0.04);
      padding: 28px 24px 24px;
    }
    .card h1 {
      margin: 0 0 8px;
      font-size: clamp(1.5rem, 4vw, 1.85rem);
      font-weight: 700;
    }
    .card .lead {
      margin: 0 0 22px;
      color: var(--muted);
      font-size: 14px;
      line-height: 1.6;
    }
    .auth-error {
      margin: 0 0 14px;
      padding: 10px 12px;
      border-radius: 12px;
      background: #fdecec;
      border: 1px solid rgba(185,28,28,0.35);
      color: var(--danger);
      font-size: 13px;
    }
    .auth-ok {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 18px;
      padding: 8px 12px;
      border-radius: 999px;
      background: #e9f7ef;
      border: 1px solid rgba(22,163,74,0.35);
      color: var(--ok);
      font-size: 13px;
      font-weight: 650;
    }
    .auth-ok::before {
      content: "";
      width: 8px;
      height: 8px;
      border-radius: 50%;
      background: var(--ok);
    }
    label {
      display: block;
      margin-bottom: 8px;
      color: var(--muted);
      font-size: 13px;
    }
    .form-control {
      width: 100%;
      border-radius: 12px;
      border: 1px solid var(--line);
      background: var(--bg);
      color: var(--text);
      padding: 12px 14px;
      font-size: 16px;
      outline: none;
    }
    .form-control:focus {
      border-color: rgba(37,99,235,0.6);
      box-shadow: 0 0 0 3px rgba(37,99,235,0.15);
    }
    .form-control::placeholder { color: #a8a29e; }
    .actions {
      display: flex;
      gap: 10px;
      margin-top: 16px;
    }
    .btn-main, .btn-ghost {
      appearance: none;
      border: 0;
      border-radius: 12px;
      padding: 12px 16px;
      font-size: 15px;
      font-weight: 650;
      cursor: pointer;
      text-decoration: none;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      min-height: 46px;
    }
    .btn-main {
      flex: 1;
      color: var(--accent-text);
      background: var(--accent);
    }
    .btn-main:hover { background: #1d4fd8; }
    .btn-ghost {
      color: var(--text);
      background: var(--btn-bg);
      border: 1px solid var(--line);
    }
    .btn-ghost:hover { background: #ebebe9; }
    .links {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 10px;
      margin-top: 18px;
    }
    .links a {
      text-decoration: none;
      color: var(--text);
      border: 1px solid var(--line);
      background: var(--card);
      border-radius: 14px;
      padding: 14px 12px;
      text-align: center;
      font-size: 14px;
      font-weight: 600;
    }
    .links a:hover {
      border-color: rgba(37,99,235,0.45);
      background: #f0f6ff;
    }
    .links a span {
      display: block;
      margin-top: 4px;
      color: var(--muted);
      font-size: 12px;
      font-weight: 500;
    }
    .disabled-box {
      padding: 14px;
      border-radius: 14px;
      border: 1px dashed var(--line);
      color: var(--muted);
      font-size: 14px;
      line-height: 1.6;
    }
    .foot {
      margin-top: 24px;
      text-align: center;
      color: var(--muted);
      font-size: 12px;
    }
    @media (max-width: 480px) {
      .page { padding-left: 16px; padding-right: 16px; }
      .card { padding: 24px 18px 18px; border-radius: 18px; }
      .links { grid-template-columns: 1fr; }
      .actions { flex-direction: column; }
      .btn-ghost { width: 100%; }
    }
  </style>
</head>
<body>
  <div class="page">
    <div class="topbar">
      <a class="brand" href="/dilfish.html">
        <div class="brand-mark">D</div>
        <div>
          dilfish
          <span class="brand-sub">private gate</span>
        </div>
      </a>
      <a class="home-link" href="/">首页</a>
    </div>
    <div class="stage">
      <div class="card">
`)
	if !authEnabled() {
		b.WriteString(`        <h1>鉴权未启用</h1>
        <p class="lead">当前配置里 <code>auth_password</code> 为空，站点登录保护处于关闭状态。</p>
        <div class="disabled-box">设置 auth_password 与 cookie_pass 后重新部署，即可启用此页面登录。</div>
        <div class="actions">
          <a class="btn-ghost" href="/">返回首页</a>
        </div>`)
	} else if Authorized(r) {
		b.WriteString(`        <div class="auth-ok">已登录</div>
        <h1>欢迎回来</h1>
        <p class="lead">登录状态有效。你可以进入工具页，或在这里登出。</p>
        <div class="links">
          <a href="/agent.html">AI 助手<span>对话与模型代理</span></a>
          <a href="/upload">上传 · 文件<span>文件上传与管理</span></a>
          <a href="/t">留言<span>临时记事板</span></a>
          <a href="/dns.html">DNS 工具<span>解析与追踪</span></a>
          <a href="/status.html">状态<span>机器与流量监控</span></a>
          <a href="/">首页<span>公开介绍页</span></a>
        </div>
        <form action="/logout" method="post" class="actions">
          <button class="btn-ghost" type="submit">登出</button>
        </form>`)
	} else {
		msg := r.URL.Query().Get("err")
		b.WriteString(`        <h1>登录</h1>
        <p class="lead">这是站点私有入口。验证通过后可使用上传、留言和 AI 代理。</p>
`)
		if msg != "" {
			b.WriteString(`        <div class="auth-error">` + html.EscapeString(msg) + `</div>
`)
		}
		b.WriteString(`        <form action="/login" method="post">
          <label for="password">密码</label>
          <input class="form-control" type="password" id="password" name="password" placeholder="输入访问密码" required autofocus autocomplete="current-password">
          <div class="actions">
            <button class="btn-main" type="submit">进入</button>
          </div>
        </form>`)
	}
	b.WriteString(`
      </div>
    </div>
    <div class="foot">ARM.871116.XYZ · personal access only</div>
  </div>
</body>
</html>
`)
	w.Write([]byte(b.String()))
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/dilfish.html", http.StatusFound)
		return
	}
	if !authEnabled() {
		http.Redirect(w, r, "/dilfish.html", http.StatusFound)
		return
	}
	ip := clientIP(r)
	if wait := loginLimiter.retryAfter(ip); wait > 0 {
		log.Println("login blocked from", ip, "retry after", wait)
		retryAfterHeader(w, wait)
		http.Error(w, "too many login attempts, try again later", http.StatusTooManyRequests)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/dilfish.html?err="+htmlQuery("bad form"), http.StatusFound)
		return
	}
	if !passwordMatch(r.FormValue("password")) {
		lock, fails := loginLimiter.fail(ip)
		log.Println("login failed from", ip, "attempt", fails)
		if lock > 0 {
			retryAfterHeader(w, lock)
			http.Error(w, "too many login attempts, try again later", http.StatusTooManyRequests)
			return
		}
		http.Redirect(w, r, "/dilfish.html?err="+htmlQuery("password incorrect"), http.StatusFound)
		return
	}
	loginLimiter.succeed(ip)
	setAuthCookie(w, r)
	next := r.FormValue("next")
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
		next = "/dilfish.html"
	}
	http.Redirect(w, r, next, http.StatusFound)
}

func retryAfterHeader(w http.ResponseWriter, d time.Duration) {
	secs := int64((d + time.Second - 1) / time.Second)
	if secs < 1 {
		secs = 1
	}
	w.Header().Set("Retry-After", strconv.FormatInt(secs, 10))
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	clearAuthCookie(w, r)
	http.Redirect(w, r, "/dilfish.html", http.StatusFound)
}

func htmlQuery(s string) string {
	return url.QueryEscape(s)
}

func redactHeaders(h http.Header) http.Header {
	out := h.Clone()
	if out.Get("Authorization") != "" {
		out.Set("Authorization", "[redacted]")
	}
	if out.Get("Cookie") != "" {
		out.Set("Cookie", "[redacted]")
	}
	if out.Get("X-Site-Token") != "" {
		out.Set("X-Site-Token", "[redacted]")
	}
	return out
}
