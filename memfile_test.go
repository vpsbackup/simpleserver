package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// memFileRef 是原实现的直译，作为内容生成的对照基准。
func memFileRef(size int64) string {
	var b strings.Builder
	for w := int64(0); size >= 10; size, w = size-10, w+10 {
		b.WriteString(fmt.Sprintf("%09d\n", w+10))
	}
	for i := int64(1); i <= size; i++ {
		b.WriteString(strconv.Itoa(int(i % 10)))
	}
	return b.String()
}

func TestMemFileContentMatchesReference(t *testing.T) {
	for _, size := range []int64{1, 5, 9, 10, 11, 19, 20, 21, 99, 100, 101, 999, 1000, 100_003} {
		got, err := io.ReadAll(newMemFileContent(size))
		if err != nil {
			t.Fatalf("size %d: ReadAll: %v", size, err)
		}
		if want := memFileRef(size); string(got) != want {
			t.Fatalf("size %d: content mismatch\ngot  %q\nwant %q", size, got, want)
		}
	}
}

func TestMemFileContentSeek(t *testing.T) {
	c := newMemFileContent(24)
	ref := memFileRef(24)

	cases := []struct {
		whence int
		offset int64
		pos    int64
	}{
		{io.SeekStart, 10, 10},
		{io.SeekCurrent, 4, 14},
		{io.SeekEnd, -4, 20},
		{io.SeekStart, 0, 0},
	}
	for _, tc := range cases {
		pos, err := c.Seek(tc.offset, tc.whence)
		if err != nil {
			t.Fatalf("Seek(%d, %d): %v", tc.offset, tc.whence, err)
		}
		if pos != tc.pos {
			t.Fatalf("Seek(%d, %d) = %d, want %d", tc.offset, tc.whence, pos, tc.pos)
		}
	}

	buf := make([]byte, 4)
	n, err := c.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(buf[:n]) != ref[:4] {
		t.Fatalf("Read after rewind = %q, want %q", buf[:n], ref[:4])
	}

	if _, err := c.Seek(-1, io.SeekStart); err == nil {
		t.Fatal("Seek to negative position should fail")
	}
}

func newMemFileServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/memfile/", MemFileHandler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// memFileDo 发起请求并读完整 body，返回 {status, header, body}。
// （返回值不是 *http.Response：那会带着已关闭的 Body 侥幸存活，bodyclose 会报警 #22）
type memFileResp struct {
	Status int
	Header http.Header
	Body   string
}

func memFileDo(t *testing.T, srv *httptest.Server, method, target, rangeHdr string) memFileResp {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL+target, nil)
	if err != nil {
		t.Fatalf("%s %s: %v", method, target, err)
	}
	if rangeHdr != "" {
		req.Header.Set("Range", rangeHdr)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, target, err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Header.Del("Date") // Date varies per request, irrelevant to tests
	if err := errors.Join(resp.Body.Close(), err); err != nil {
		t.Fatalf("%s %s: read body: %v", method, target, err)
	}
	return memFileResp{Status: resp.StatusCode, Header: resp.Header, Body: string(body)}
}

func TestMemFileHandler(t *testing.T) {
	srv := newMemFileServer(t)

	t.Run("full", func(t *testing.T) {
		r := memFileDo(t, srv, "GET", "/memfile/32", "")
		body := r.Body
		if r.Status != http.StatusOK {
			t.Fatalf("status = %d, want 200", r.Status)
		}
		if body != memFileRef(32) {
			t.Fatalf("body = %q, want %q", body, memFileRef(32))
		}
		if got := r.Header.Get("Content-Length"); got != "32" {
			t.Fatalf("Content-Length = %q, want 32", got)
		}
		if got := r.Header.Get("Accept-Ranges"); got != "bytes" {
			t.Fatalf("Accept-Ranges = %q, want bytes", got)
		}
	})

	t.Run("default size", func(t *testing.T) {
		r := memFileDo(t, srv, "GET", "/memfile/", "")
		if r.Status != http.StatusOK {
			t.Fatalf("status = %d, want 200", r.Status)
		}
		if len(r.Body) != int(defaultMemFileSize) {
			t.Fatalf("len(body) = %d, want %d", len(r.Body), defaultMemFileSize)
		}
	})

	// 回归：带查询串时大小不能被忽略（旧实现用 RequestURI 解析会退化成默认值）。
	t.Run("query string", func(t *testing.T) {
		body := memFileDo(t, srv, "GET", "/memfile/32?ts=123", "").Body
		if len(body) != 32 {
			t.Fatalf("len(body) = %d, want 32", len(body))
		}
		if body != memFileRef(32) {
			t.Fatalf("body = %q, want %q", body, memFileRef(32))
		}
	})

	rangeCases := []struct {
		name  string
		hdr   string
		start int64
		end   int64 // 含
	}{
		{"closed", "bytes=10-19", 10, 19},
		{"open ended", "bytes=30-", 30, 31},
		{"suffix", "bytes=-10", 22, 31},
		{"single byte", "bytes=7-7", 7, 7},
	}
	for _, tc := range rangeCases {
		t.Run("range "+tc.name, func(t *testing.T) {
			ref := memFileRef(32)
			r := memFileDo(t, srv, "GET", "/memfile/32", tc.hdr)
			body := r.Body
			if r.Status != http.StatusPartialContent {
				t.Fatalf("status = %d, want 206", r.Status)
			}
			want := ref[tc.start : tc.end+1]
			if body != want {
				t.Fatalf("body = %q, want %q", body, want)
			}
			wantCR := fmt.Sprintf("bytes %d-%d/32", tc.start, tc.end)
			if got := r.Header.Get("Content-Range"); got != wantCR {
				t.Fatalf("Content-Range = %q, want %q", got, wantCR)
			}
			if got := r.Header.Get("Content-Length"); got != strconv.FormatInt(tc.end-tc.start+1, 10) {
				t.Fatalf("Content-Length = %q, want %d", got, tc.end-tc.start+1)
			}
		})
	}

	t.Run("multi range", func(t *testing.T) {
		ref := memFileRef(32)
		r := memFileDo(t, srv, "GET", "/memfile/32", "bytes=0-4,20-24")
		body := r.Body
		if r.Status != http.StatusPartialContent {
			t.Fatalf("status = %d, want 206", r.Status)
		}
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "multipart/byteranges") {
			t.Fatalf("Content-Type = %q, want multipart/byteranges", ct)
		}
		for _, want := range []string{ref[0:5], ref[20:25]} {
			if !strings.Contains(body, want) {
				t.Fatalf("body missing part %q: %q", want, body)
			}
		}
	})

	t.Run("range not satisfiable", func(t *testing.T) {
		r := memFileDo(t, srv, "GET", "/memfile/32", "bytes=40-50")
		if r.Status != http.StatusRequestedRangeNotSatisfiable {
			t.Fatalf("status = %d, want 416", r.Status)
		}
		if got := r.Header.Get("Content-Range"); got != "bytes */32" {
			t.Fatalf("Content-Range = %q, want bytes */32", got)
		}
	})

	t.Run("head", func(t *testing.T) {
		r := memFileDo(t, srv, "HEAD", "/memfile/32", "")
		if r.Status != http.StatusOK {
			t.Fatalf("status = %d, want 200", r.Status)
		}
		if r.Body != "" {
			t.Fatalf("HEAD body = %q, want empty", r.Body)
		}
		if got := r.Header.Get("Content-Length"); got != "32" {
			t.Fatalf("Content-Length = %q, want 32", got)
		}
	})

	badCases := []struct {
		name   string
		target string
	}{
		{"not a number", "/memfile/abc"},
		{"zero", "/memfile/0"},
		{"negative", "/memfile/-5"},
		{"too large", "/memfile/1000000000"},
	}
	for _, tc := range badCases {
		t.Run("fallback "+tc.name, func(t *testing.T) {
			r := memFileDo(t, srv, "GET", tc.target, "")
			if r.Status != http.StatusOK {
				t.Fatalf("status = %d, want 200", r.Status)
			}
			if len(r.Body) != int(defaultMemFileSize) {
				t.Fatalf("len(body) = %d, want %d", len(r.Body), defaultMemFileSize)
			}
		})
	}
}

func TestMemFileSize(t *testing.T) {
	cases := []struct {
		path string
		size int64
	}{
		{"/memfile/", defaultMemFileSize},
		{"/memfile/1", 1},
		{"/memfile/1024", 1024},
		{"/memfile/999999999", 999_999_999},
		// 任何异常都退回默认大小。
		{"/memfile/0", defaultMemFileSize},
		{"/memfile/-1", defaultMemFileSize},
		{"/memfile/1000000000", defaultMemFileSize},
		{"/memfile/1k", defaultMemFileSize},
		{"/memfile/1/", defaultMemFileSize},
		{"/other/1", defaultMemFileSize},
	}
	for _, c := range cases {
		if size := memFileSize(c.path); size != c.size {
			t.Errorf("memFileSize(%q) = %d, want %d", c.path, size, c.size)
		}
	}
}
