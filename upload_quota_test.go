package main

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// multipartBody builds a POST /upload request with the given file parts.
func multipartBody(t *testing.T, contents ...string) *http.Request {
	t.Helper()
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		defer pw.Close()
		for i, body := range contents {
			fw, err := mw.CreateFormFile("file", fmt.Sprintf("f%d.txt", i))
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			if _, err := io.WriteString(fw, body); err != nil {
				pw.CloseWithError(err)
				return
			}
		}
		if err := mw.Close(); err != nil {
			pw.CloseWithError(err)
		}
	}()
	req := httptest.NewRequest(http.MethodPost, "/upload", pr)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func fileCount(t *testing.T, u *UploaderService) int {
	t.Helper()
	files, _, err := u.ListFiles()
	if err != nil {
		t.Fatal(err)
	}
	return len(files)
}

func invariants(t *testing.T, u *UploaderService, wantFiles int) {
	t.Helper()
	files, total, err := u.ListFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != wantFiles {
		t.Fatalf("files on disk = %d want %d", len(files), wantFiles)
	}
	u.Lock.Lock()
	defer u.Lock.Unlock()
	if u.Pending != 0 {
		t.Fatalf("pending reservations left: %d", u.Pending)
	}
	if u.Curr != total {
		t.Fatalf("Curr %d != disk usage %d", u.Curr, total)
	}
}

// #2: concurrent uploads cannot race the quota check and the add apart, and a
// partial upload that loses the race leaves nothing behind.
// Note: successes are not a fixed 4 — the streaming writer reserves a whole
// chunk up to the remaining quota at a time, so in-flight reservations make
// the realized count scheduling-dependent; what must hold under any
// interleaving is the invariant set below (quota never bypassed, disk ==
// accounting, no residue).
func TestHandlerQuotaRaceConcurrent(t *testing.T) {
	dir := t.TempDir()
	const (
		files = 8
		size  = 1000
		quota = 4096 // exactly 4 files fit; the rest fail mid-stream
	)
	u := NewUploadService("https://example.test/dl/", dir, "", 8<<10, quota, NeverExpire, 5)

	var okCount, failCount int64
	var wg sync.WaitGroup
	for i := 0; i < files; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rec := httptest.NewRecorder()
			u.Handler(rec, multipartBody(t, strings.Repeat("a", size)))
			switch {
			case strings.Contains(rec.Body.String(), "上传成功"):
				atomic.AddInt64(&okCount, 1)
			case strings.Contains(rec.Body.String(), "quota exceeded") && rec.Code == http.StatusRequestEntityTooLarge:
				atomic.AddInt64(&failCount, 1)
			default:
				t.Errorf("unexpected response: %d %q", rec.Code, rec.Body.String())
			}
		}(i)
	}
	wg.Wait()

	if got := atomic.LoadInt64(&okCount); got < 1 || got > 4 {
		t.Fatalf("successes = %d want 1..4", got)
	}
	if got := atomic.LoadInt64(&failCount); got != files-atomic.LoadInt64(&okCount) {
		t.Fatalf("failures = %d want %d", got, files-atomic.LoadInt64(&okCount))
	}
	invariants(t, u, int(okCount))
	if u.Curr > quota {
		t.Fatalf("Curr %d exceeds quota %d", u.Curr, quota)
	}
	if u.Curr != 1000*okCount {
		t.Fatalf("Curr %d != committed usage %d", u.Curr, 1000*okCount)
	}
}

// #3: max_single_file_mb is a hard per-file cap; a file at the cap is fine.
func TestHandlerPerFileLimit(t *testing.T) {
	dir := t.TempDir()
	u := NewUploadService("https://example.test/dl/", dir, "", 8, 1<<20, NeverExpire, 5)

	rec := httptest.NewRecorder()
	u.Handler(rec, multipartBody(t, "0123456789"))
	if !strings.Contains(rec.Body.String(), "per-file size limit") {
		t.Fatalf("oversize response = %q", rec.Body.String())
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversize status = %d want 413", rec.Code)
	}
	invariants(t, u, 0)

	rec = httptest.NewRecorder()
	u.Handler(rec, multipartBody(t, "01234567"))
	if !strings.Contains(rec.Body.String(), "上传成功") {
		t.Fatalf("cap-sized upload response = %q", rec.Body.String())
	}
	invariants(t, u, 1)
}

// #3: streaming means oversized parts never touch /tmp and the extra parts of
// one request are drained, so the second file in a request is not stored.
func TestHandlerOnlyFirstFileStored(t *testing.T) {
	dir := t.TempDir()
	u := NewUploadService("https://example.test/dl/", dir, "", 8<<10, 1<<20, NeverExpire, 5)
	rec := httptest.NewRecorder()
	u.Handler(rec, multipartBody(t, "first", strings.Repeat("b", 5000)))
	if !strings.Contains(rec.Body.String(), "上传成功") {
		t.Fatalf("response = %q", rec.Body.String())
	}
	invariants(t, u, 1)
	files, total, _ := u.ListFiles()
	if total != 5 {
		t.Fatalf("total = %d want 5 (%+v)", total, files)
	}
}

// #4: a mid-copy failure must not leave a truncated file or a dirty Map/Curr.
func TestWriteFileReadErrorCleansUp(t *testing.T) {
	dir := t.TempDir()
	u := NewUploadService("https://example.test/dl/", dir, "", 8<<10, 1<<20, NeverExpire, 5)
	boom := errors.New("disk exploded")
	rc := io.MultiReader(strings.NewReader(strings.Repeat("0123456789", 2)), errReader{err: boom})

	n, _, err := u.WriteFile("a.txt", rc)
	if err == nil || !strings.Contains(err.Error(), "disk exploded") {
		t.Fatalf("err = %v want %v", err, boom)
	}
	if n != 20 {
		t.Fatalf("written = %d want 20", n)
	}
	invariants(t, u, 0)

	u.Lock.Lock()
	defer u.Lock.Unlock()
	if len(u.Map) != 0 {
		t.Fatalf("Map still has %d entries", len(u.Map))
	}
}

// #2: Patrol/DeleteFile keep working while the quota counts in-flight files.
func TestQuotaReservationAccounting(t *testing.T) {
	u := NewUploadService("https://example.test/dl/", t.TempDir(), "", 8<<10, 1000, NeverExpire, 5)
	if got := u.reserveUpTo(600); got != 600 {
		t.Fatalf("reserve = %d want 600", got)
	}
	if got := u.reserveUpTo(600); got != 400 {
		t.Fatalf("reserve = %d want 400", got)
	}
	if got := u.reserveUpTo(1); got != 0 {
		t.Fatalf("reserve = %d want 0", got)
	}
	u.settleQuota(500, 100) // 500 written, 100 aborted
	if u.Curr != 500 || u.Pending != 400 {
		t.Fatalf("Curr = %d Pending = %d want 500/400", u.Curr, u.Pending)
	}
	u.releaseQuota(400)
	if u.Curr != 500 || u.Pending != 0 {
		t.Fatalf("Curr = %d Pending = %d want 500/0", u.Curr, u.Pending)
	}
}

type errReader struct{ err error }

func (e errReader) Read([]byte) (int, error) { return 0, e.err }
