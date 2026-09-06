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
// Note: index files are served with http.ServeContent rather than the
// wrapped FileServer, because the FileServer redirects /index.html back to
// "./" which would loop with the rewrite below.
type staticNoDirFS struct {
	handler http.Handler // the wrapped http.FileServer
	fs      http.FileSystem
}

func (s staticNoDirFS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "/") {
		s.handler.ServeHTTP(w, r)
		return
	}
	// directory request: only serve it when an index.html exists, otherwise
	// answer 404 instead of letting the FileServer render a listing.
	idx := path.Join(path.Clean(r.URL.Path), "index.html")
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
