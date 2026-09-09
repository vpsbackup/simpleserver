package main

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestListFilesMD5CacheAndKeepRowOnError(t *testing.T) {
	dir := t.TempDir()
	name := "hello.txt"
	content := []byte("hello-md5-cache")
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("%x", md5.Sum(content))

	u := NewUploadService("https://example.test/dl/", dir, "", 1024, 1024*1024, NeverExpire, 5)

	list1, total1, err := u.ListFiles()
	if err != nil {
		t.Fatal(err)
	}
	if total1 != info.Size() || len(list1) != 1 {
		t.Fatalf("first list: len=%d total=%d want 1/%d", len(list1), total1, info.Size())
	}
	if list1[0].MD5 != want {
		t.Fatalf("first md5=%q want %q", list1[0].MD5, want)
	}
	if _, ok := u.cachedMD5(name, info.Size(), info.ModTime()); !ok {
		t.Fatal("expected md5 cache filled after first list")
	}

	// Force miss by wrong size so ListFiles recomputes from disk.
	u.putCachedMD5(name, info.Size()+1, info.ModTime(), "deadbeef")
	list2, _, err := u.ListFiles()
	if err != nil {
		t.Fatal(err)
	}
	if list2[0].MD5 != want {
		t.Fatalf("recomputed md5=%q want %q", list2[0].MD5, want)
	}

	// Hash failure must still keep the row with empty md5.
	gone := filepath.Join(dir, "missing-open.txt")
	if err := os.WriteFile(gone, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(gone, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(gone, 0o644) })

	list3, total3, err := u.ListFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(list3) != 2 {
		t.Fatalf("expected 2 rows after unreadable file, got %d", len(list3))
	}
	var foundMissing bool
	for _, f := range list3 {
		if f.Name == "missing-open.txt" {
			foundMissing = true
			if f.MD5 != "" {
				t.Fatalf("unreadable file md5 should be empty, got %q", f.MD5)
			}
		}
	}
	if !foundMissing {
		t.Fatal("unreadable file missing from list")
	}
	if total3 <= info.Size() {
		t.Fatalf("total_size should still include unreadable file, got %d", total3)
	}
}

func TestWriteFileWarmsMD5CacheAndDeleteClears(t *testing.T) {
	dir := t.TempDir()
	u := NewUploadService("https://example.test/dl/", dir, "", 1024, 1024*1024, NeverExpire, 5)
	content := "warm-cache-body"
	n, name, err := u.WriteFile("a.txt", strings.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(content)) {
		t.Fatalf("wrote %d want %d", n, len(content))
	}
	info, err := os.Stat(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	sum, ok := u.cachedMD5(name, info.Size(), info.ModTime())
	if !ok {
		t.Fatal("upload should warm md5 cache")
	}
	want := fmt.Sprintf("%x", md5.Sum([]byte(content)))
	if sum != want {
		t.Fatalf("cached md5=%q want %q", sum, want)
	}

	// Second list should hit cache (same value).
	list, _, err := u.ListFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].MD5 != want {
		t.Fatalf("list after upload: %+v", list)
	}

	if err := u.DeleteFile(name); err != nil {
		t.Fatal(err)
	}
	if _, ok := u.cachedMD5(name, info.Size(), info.ModTime()); ok {
		t.Fatal("delete should clear md5 cache")
	}
	// Ensure cache map entry gone even with stale key lookup.
	u.md5Mu.Lock()
	_, still := u.md5Cache[name]
	u.md5Mu.Unlock()
	if still {
		t.Fatal("md5Cache entry still present after delete")
	}
}

func TestCachedMD5MismatchInvalidates(t *testing.T) {
	u := NewUploadService("https://example.test/dl/", t.TempDir(), "", 1, 1, NeverExpire, 5)
	now := time.Now()
	u.putCachedMD5("f.bin", 10, now, "abc")
	if _, ok := u.cachedMD5("f.bin", 11, now); ok {
		t.Fatal("size mismatch should miss")
	}
	if _, ok := u.cachedMD5("f.bin", 10, now.Add(time.Second)); ok {
		t.Fatal("mtime mismatch should miss")
	}
	if sum, ok := u.cachedMD5("f.bin", 10, now); !ok || sum != "abc" {
		t.Fatalf("exact match failed: %q %v", sum, ok)
	}
}
