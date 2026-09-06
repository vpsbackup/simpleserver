package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newStaticTestDir build a static tree:
//
//	root/index.html            "root-index"
//	root/withindex/index.html  "sub-index"
//	root/empty/f.txt           "file-content"
func newStaticTestDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("index.html", "root-index")
	write("withindex/index.html", "sub-index")
	write("empty/f.txt", "file-content")
	return root
}

func TestStaticNoDirFS(t *testing.T) {
	root := newStaticTestDir(t)
	fs := http.Dir(root)
	srv := httptest.NewServer(staticNoDirFS{http.FileServer(fs), fs})
	defer srv.Close()

	cases := []struct {
		path string
		code int
		body string
	}{
		{"/", 200, "root-index"},
		{"/withindex/", 200, "sub-index"},
		{"/empty/", 404, ""},
		{"/empty/f.txt", 200, "file-content"},
		{"/nope.txt", 404, ""},
	}
	for _, c := range cases {
		resp, err := http.Get(srv.URL + c.path)
		if err != nil {
			t.Fatalf("GET %s: %v", c.path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != c.code {
			t.Errorf("GET %s = %d, want %d", c.path, resp.StatusCode, c.code)
			continue
		}
		if c.body != "" && !strings.Contains(string(body), c.body) {
			t.Errorf("GET %s body missing %q", c.path, c.body)
		}
	}

	// a directory path without the trailing slash gets redirected by the
	// FileServer to the slash form, which then 404s (no listing either way).
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get(srv.URL + "/empty")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode == 200 {
		t.Error("GET /empty should not serve a listing")
	}
}
