package main

import (
	"crypto/sha512"
	"crypto/subtle"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

func needsAuth(r *http.Request) bool {
	if !authEnabled() {
		return false
	}
	path := r.URL.Path
	switch {
	case r.Method == http.MethodPost && path == "/upload":
		return true
	case r.Method == http.MethodPost && path == "/t":
		return true
	case strings.HasPrefix(path, "/v1/agentproxy/"):
		return r.Method != http.MethodOptions
	default:
		return false
	}
}

func rejectUnauthorized(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	secFetch := r.Header.Get("Sec-Fetch-Mode")
	if r.Method == http.MethodGet || strings.Contains(accept, "text/html") || secFetch == "navigate" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	http.Error(w, "unauthorized: login at / first", http.StatusUnauthorized)
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
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
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
    <meta name="description" content="about">
    <meta name="author" content="sean">
    <meta name="theme-color" content="#DC3545"/>
    <title>About this site & me</title>
    <link href="/302/bootstrap.css" rel="stylesheet">
    <style>
      .auth-corner {
        position: fixed;
        top: 16px;
        right: 16px;
        z-index: 1030;
        width: min(280px, calc(100vw - 32px));
        background: #fff;
        border: 1px solid rgba(0,0,0,.08);
        border-radius: 12px;
        box-shadow: 0 8px 24px rgba(0,0,0,.08);
        padding: 12px;
      }
      .auth-corner .auth-title {
        font-size: 12px;
        color: #6c757d;
        margin-bottom: 8px;
      }
      .auth-corner .auth-row {
        display: flex;
        gap: 8px;
        align-items: center;
      }
      .auth-corner .form-control {
        min-width: 0;
      }
      .auth-corner .auth-links {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
        margin-top: 8px;
        font-size: 13px;
      }
      .auth-corner .auth-error {
        color: #dc3545;
        font-size: 12px;
        margin-bottom: 8px;
      }
      .auth-corner .auth-ok {
        color: #198754;
        font-size: 13px;
        font-weight: 600;
      }
      body { padding-top: 12px; }
      @media (max-width: 576px) {
        main.container { padding-top: 88px; }
      }
    </style>
  </head>
  <body>
`)
	b.WriteString(`    <div class="auth-corner">`)
	if !authEnabled() {
		b.WriteString(`<div class="auth-title">Auth</div><div class="text-muted" style="font-size:13px">disabled</div>`)
	} else if Authorized(r) {
		b.WriteString(`<div class="auth-row" style="justify-content:space-between">
  <div class="auth-ok">已登录</div>
  <form action="/logout" method="post" style="margin:0">
    <button class="btn btn-sm btn-outline-danger" type="submit">登出</button>
  </form>
</div>
<div class="auth-links">
  <a href="/agent.html">AI 助手</a>
  <a href="/upload">上传</a>
  <a href="/t">留言</a>
</div>`)
	} else {
		msg := r.URL.Query().Get("err")
		if msg != "" {
			b.WriteString(`<div class="auth-error">` + html.EscapeString(msg) + `</div>`)
		}
		b.WriteString(`<div class="auth-title">登录</div>
<form action="/login" method="post" class="auth-row">
  <input class="form-control form-control-sm" type="password" id="password" name="password" placeholder="密码" required autofocus>
  <button class="btn btn-sm btn-danger" type="submit">登录</button>
</form>`)
	}
	b.WriteString(`</div>
    <main role="main" class="container">
      <h1 class="mt-5">🪛 About this site</h1>
      <p class="lead">
     This is a personal website, you need authorative information to visit it.
</p>
      <h1 class="mt-5">About me</h1>
      <p class="lead">
veteran programmer, grshccji atsign duck.com
</p>
</main>
    <footer class="footer">
      <div class="container">
  <span class="text-muted">
        ARM.871116.XYZ &copy;
        All rights reserved，
        2020-2026
  </span>
  </div>
</footer>
<footer class="footer">
<div class="container">
  <span class="text-muted">
 This website is powered by honourable IPv6,
 and obsoleting the despicable IPv4.
  </span>
</div>
</footer>
  </body>
</html>
`)
	w.Write([]byte(b.String()))
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if !authEnabled() {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/?err="+htmlQuery("bad form"), http.StatusFound)
		return
	}
	if !passwordMatch(r.FormValue("password")) {
		log.Println("login failed from", r.RemoteAddr)
		http.Redirect(w, r, "/?err="+htmlQuery("password incorrect"), http.StatusFound)
		return
	}
	setAuthCookie(w, r)
	next := r.FormValue("next")
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
		next = "/"
	}
	http.Redirect(w, r, next, http.StatusFound)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	clearAuthCookie(w, r)
	http.Redirect(w, r, "/", http.StatusFound)
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
