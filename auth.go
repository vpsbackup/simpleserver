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
  </head>
  <body>
    <main role="main" class="container">
      <h1 class="mt-5">🪛 About this site</h1>
      <p class="lead">
     This is a personal website, you need authorative information to visit it.
</p>
      <h1 class="mt-5">About me</h1>
      <p class="lead">
veteran programmer, grshccji atsign duck.com
</p>
`)
	if !authEnabled() {
		b.WriteString(`<div class="alert alert-secondary mt-4">Auth is disabled (auth_password is empty).</div>`)
	} else if Authorized(r) {
		b.WriteString(`<div class="alert alert-success mt-4">已登录</div>
<form action="/logout" method="post" class="mt-3">
  <button class="btn btn-outline-danger" type="submit">登出</button>
</form>
<p class="mt-4"><a href="/agent.html">打开 AI 助手</a> · <a href="/upload">上传</a> · <a href="/t">留言</a></p>
`)
	} else {
		msg := r.URL.Query().Get("err")
		if msg != "" {
			b.WriteString(`<div class="alert alert-danger mt-4">` + html.EscapeString(msg) + `</div>`)
		}
		b.WriteString(`<h1 class="mt-5">Login</h1>
<form action="/login" method="post" class="mt-3" style="max-width:420px">
  <div class="mb-3">
    <label class="form-label" for="password">Password</label>
    <input class="form-control" type="password" id="password" name="password" required autofocus>
  </div>
  <button class="btn btn-danger" type="submit">登录</button>
</form>
`)
	}
	b.WriteString(`
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
