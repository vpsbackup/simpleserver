package main

import (
	"net/http"
	"path"
	"strings"
)

// staticNoDirFS serves static files from a http.FileSystem but never lists
// directories. The stock http.FileServer generates a browsable listing for
// any directory without an index.html, which would expose everything under
// the static root (e.g. the upload dir /dl/) to anonymous visitors.
//
// Behaviour:
//   - "/" and any directory containing an index.html are served normally;
//   - a directory request without an index.html returns 404;
//   - regular file requests (e.g. /dl/<random>.txt) pass through untouched.
//
// The wrapped handler is normally mounted with http.StripPrefix, so the request
// path still carries the mount prefix while fs paths do not: prefix names that
// mount ("/dl/", empty for a root mount) and the index lookup happens below it.
//
// Note: index files are served with http.ServeContent rather than the
// wrapped FileServer, because the FileServer redirects /index.html back to
// "./" which would loop with the rewrite below.
type staticNoDirFS struct {
	handler http.Handler // the wrapped http.FileServer
	fs      http.FileSystem
	prefix  string // URL prefix of the mount, e.g. "/dl/"; "" when mounted at /
}

func (s staticNoDirFS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "/") {
		s.handler.ServeHTTP(w, r)
		return
	}
	// directory request: only serve it when an index.html exists, otherwise
	// answer 404 instead of letting the FileServer render a listing.
	rel := strings.TrimPrefix(r.URL.Path, s.prefix)
	if !strings.HasPrefix(rel, "/") {
		rel = "/" + rel
	}
	idx := path.Join(path.Clean(rel), "index.html")
	f, err := s.fs.Open(idx)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, "index.html", st.ModTime(), f)
}

// dlAttachmentHandler 把 /dl/ 的响应强制成"下载"语义（review #12）：
// 上传的 .html/.svg 只会被另存，不会在本站源里执行脚本——
// 配合 nosniff 挡掉内容嗅探，存储型 XSS 的执行面就没了。
type dlAttachmentHandler struct {
	inner http.Handler
}

func (h dlAttachmentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if !strings.HasSuffix(r.URL.Path, "/") {
		name := strings.Map(func(c rune) rune {
			if c < 0x20 || c == '"' {
				return '_'
			}
			return c
		}, path.Base(r.URL.Path))
		w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
	}
	h.inner.ServeHTTP(w, r)
}
