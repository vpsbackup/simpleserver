package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type uploadJSON struct {
	OK   bool   `json:"ok"`
	Name string `json:"name"`
	URL  string `json:"url"`
	Size int64  `json:"size"`
}

// useTestUploader points the package-level uploader at a temp dir, so
// ApiFilesUpload exercises the same getUploader() path as production.
func useTestUploader(t *testing.T, maxSingle, maxTotal int64) (*UploaderService, string) {
	t.Helper()
	dir := t.TempDir()
	old := uploadService
	u := NewUploadService("https://example.test/dl/", dir, "", maxSingle, maxTotal, NeverExpire, 5)
	uploadService = u
	t.Cleanup(func() { uploadService = old })
	return u, dir
}

// apiUploadRequest builds POST /api/files/upload with one file part.
func apiUploadRequest(t *testing.T, filename, body string) *http.Request {
	t.Helper()
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		defer pw.Close()
		fw, err := mw.CreateFormFile("file", filename)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := io.WriteString(fw, body); err != nil {
			pw.CloseWithError(err)
			return
		}
		if err := mw.Close(); err != nil {
			pw.CloseWithError(err)
		}
	}()
	req := httptest.NewRequest(http.MethodPost, "/api/files/upload", pr)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func decodeUploadJSON(t *testing.T, rr *httptest.ResponseRecorder) uploadJSON {
	t.Helper()
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("content-type = %q want json", ct)
	}
	var got uploadJSON
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not json: %v %q", err, rr.Body.String())
	}
	return got
}

func TestApiFilesUploadReturnsRealName(t *testing.T) {
	_, dir := useTestUploader(t, 1<<20, 1<<20)
	const body = "encrypted-store-ciphertext"

	rr := httptest.NewRecorder()
	ApiFilesUpload(rr, apiUploadRequest(t, "store.20261111.2030.totp", body))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %q", rr.Code, rr.Body.String())
	}
	got := decodeUploadJSON(t, rr)
	if !got.OK {
		t.Fatalf("ok = false: %q", rr.Body.String())
	}
	if !strings.HasSuffix(got.Name, ".totp") {
		t.Fatalf("name %q should keep the .totp extension", got.Name)
	}
	if len(got.Name) != 5+len(".totp") {
		t.Fatalf("name %q should be RandStr(5)+ext", got.Name)
	}
	if got.URL != "https://example.test/dl/"+got.Name {
		t.Fatalf("url %q != base+name", got.URL)
	}
	if got.Size != int64(len(body)) {
		t.Fatalf("size = %d want %d", got.Size, len(body))
	}
	// The name must be the file actually on disk, not the client's.
	onDisk, err := os.ReadFile(filepath.Join(dir, got.Name))
	if err != nil {
		t.Fatalf("returned name is not on disk: %v", err)
	}
	if string(onDisk) != body {
		t.Fatalf("disk content = %q want %q", onDisk, body)
	}
	if _, err := os.Stat(filepath.Join(dir, "store.20261111.2030.totp")); err == nil {
		t.Fatal("client file name must not be used verbatim")
	}
	invariants(t, getUploader(), 1)
}

// The method check lives in the handler, so /api/files/upload must reach it
// for every method instead of falling through to "bad api request name".
func TestApiFilesUploadBadMethodThroughApiHandler(t *testing.T) {
	useTestUploader(t, 1<<20, 1<<20)
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		rr := httptest.NewRecorder()
		ApiHandler(rr, httptest.NewRequest(method, "/api/files/upload", nil))
		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s status = %d body = %q want 405", method, rr.Code, rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), "bad method for file upload") {
			t.Fatalf("%s body = %q", method, rr.Body.String())
		}
	}
}

func TestApiFilesUploadBadRequest(t *testing.T) {
	u, _ := useTestUploader(t, 1<<20, 1<<20)

	// Not multipart at all.
	req := httptest.NewRequest(http.MethodPost, "/api/files/upload", strings.NewReader("plain"))
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	ApiFilesUpload(rr, req)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "bad upload request") {
		t.Fatalf("non-multipart: %d %q", rr.Code, rr.Body.String())
	}

	// Multipart but without a file part.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("note", "no file here"); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/files/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr = httptest.NewRecorder()
	ApiFilesUpload(rr, req)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "no file part") {
		t.Fatalf("no file part: %d %q", rr.Code, rr.Body.String())
	}
	invariants(t, u, 0)
}

func TestApiFilesUploadLimits(t *testing.T) {
	// Per-file cap.
	u, _ := useTestUploader(t, 8, 1<<20)
	rr := httptest.NewRecorder()
	ApiFilesUpload(rr, apiUploadRequest(t, "big.totp", "0123456789"))
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("per-file status = %d body = %q", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "file exceeds per-file size limit") {
		t.Fatalf("per-file body = %q", rr.Body.String())
	}
	invariants(t, u, 0)

	// Total quota.
	u, _ = useTestUploader(t, 1<<20, 4)
	rr = httptest.NewRecorder()
	ApiFilesUpload(rr, apiUploadRequest(t, "quota.totp", "0123456789"))
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("quota status = %d body = %q", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "upload quota exceeded") {
		t.Fatalf("quota body = %q", rr.Body.String())
	}
	invariants(t, u, 0)
}

func TestApiFilesUploadNeedsAuth(t *testing.T) {
	enableTestAuth(t)
	dir := t.TempDir()
	oldSvc := uploadService
	uploadService = nil
	Cfg.UploadDir = dir
	Cfg.Domain = "arm.test"
	Cfg.Msg = false
	Cfg.Tcping = false
	Cfg.EnableView = false
	Cfg.Tracer = false
	Cfg.StaticDir = "./public"
	t.Cleanup(func() { uploadService = oldSvc })
	mux, err := InitMux()
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, apiUploadRequest(t, "store.20261111.2030.totp", "ciphertext"))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("no token status = %d body = %q want 401", rr.Code, rr.Body.String())
	}

	req := apiUploadRequest(t, "store.20261111.2030.totp", "ciphertext")
	req.Header.Set("X-Site-Token", "site-pass-123")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("token status = %d body = %q", rr.Code, rr.Body.String())
	}
	got := decodeUploadJSON(t, rr)
	if got.URL != "https://arm.test/dl/"+got.Name {
		t.Fatalf("url %q should use the configured domain", got.URL)
	}
	if _, err := os.Stat(filepath.Join(dir, got.Name)); err != nil {
		t.Fatalf("file not written into upload_dir: %v", err)
	}
}

// The /upload refactor moved the multipart walk into writeMultipartFile; its
// responses must stay byte-for-byte what they were.
func TestUploadHandlerLegacyResponsesUnchanged(t *testing.T) {
	u, dir := useTestUploader(t, 1<<20, 1<<20)

	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("plain"))
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	u.Handler(rr, req)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "bad upload request") {
		t.Fatalf("non-multipart: %d %q", rr.Code, rr.Body.String())
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("note", "x"); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodPost, "/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr = httptest.NewRecorder()
	u.Handler(rr, req)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "no file part") {
		t.Fatalf("no file part: %d %q", rr.Code, rr.Body.String())
	}

	// A stored file plus a truncated trailing part stays a success, as before.
	var trunc bytes.Buffer
	mw = multipart.NewWriter(&trunc)
	fw, err := mw.CreateFormFile("file", "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(fw, "hello"); err != nil {
		t.Fatal(err)
	}
	// Terminate the part, then start a malformed second part by hand; the
	// writer is left open on purpose so no closing boundary is emitted.
	trunc.WriteString("\r\n--" + mw.Boundary() + "\r\nthis-is-not-a-header\r\n\r\n")
	req = httptest.NewRequest(http.MethodPost, "/upload", &trunc)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr = httptest.NewRecorder()
	u.Handler(rr, req)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "上传成功") {
		t.Fatalf("truncated tail after a stored file: %d %q", rr.Code, rr.Body.String())
	}
	invariants(t, u, 1)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one stored file, got %d", len(entries))
	}
	bt, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	if string(bt) != "hello" {
		t.Fatalf("stored content = %q want hello", bt)
	}
}
