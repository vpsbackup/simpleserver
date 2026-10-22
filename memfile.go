package main

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// defaultMemFileSize 是不带大小请求时返回的字节数，与原实现保持一致。
	defaultMemFileSize int64 = 100
	// maxMemFileSize 限制单次响应大小，避免误传大数打满带宽。
	maxMemFileSize int64 = 999_999_999
)

// MemFileHandler 返回指定大小的内容，用于测速。
//
// 大小取自 /memfile/{size}，查询串不参与解析（/memfile/1024?x=1 仍然是 1024）。
// 内容按需生成，不会为请求的大小分配内存。
// Range 请求（单段与多段）由 http.ServeContent 处理，因此 206/416/HEAD 都是标准行为。
func MemFileHandler(w http.ResponseWriter, r *http.Request) {
	size := memFileSize(r.URL.Path)
	w.Header().Set("Content-Type", "text/plain")
	// modtime 传零值：不输出 Last-Modified，也不参与 If-Modified-Since。
	// 没有校验器时 If-Range 会被忽略（退回 200 全量），符合 RFC 9110。
	http.ServeContent(w, r, "memfile.txt", time.Time{}, newMemFileContent(size))
}

// memFileSize 解析 /memfile/{size}。任何异常（非数字、非正数、超上限）都静默
// 退回 defaultMemFileSize，保持与旧实现一致的容错行为。
func memFileSize(path string) int64 {
	if !strings.HasPrefix(path, "/memfile/") {
		return defaultMemFileSize
	}
	sizeStr := strings.TrimPrefix(path, "/memfile/")
	if sizeStr == "" {
		return defaultMemFileSize
	}
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil || size <= 0 || size > maxMemFileSize {
		return defaultMemFileSize
	}
	return size
}

// memFileContent 是按需生成内容的 io.ReadSeeker。
//
// 内容布局与原实现一致：每 10 字节为一段，段内容是 9 位补零的段结束偏移加换行；
// 末尾不足 10 字节的部分为 1,2,3... 递增的数字。
type memFileContent struct {
	size int64 // 总字节数
	off  int64 // 当前读写位置
	full int64 // 完整 10 字节段覆盖的字节数
}

var _ io.ReadSeeker = (*memFileContent)(nil)

func newMemFileContent(size int64) *memFileContent {
	return &memFileContent{size: size, full: size / 10 * 10}
}

func (c *memFileContent) Read(p []byte) (int, error) {
	if c.off >= c.size {
		return 0, io.EOF
	}
	limit := int64(len(p))
	if remain := c.size - c.off; limit > remain {
		limit = remain
	}

	var (
		block [10]byte
		tail  [9]byte
	)
	for i, tailLen := int64(0), c.size-c.full; i < tailLen; i++ {
		tail[i] = byte('0' + (i+1)%10)
	}

	n := int64(0)
	for n < limit {
		pos := c.off + n
		if pos < c.full {
			idx := pos / 10
			fillMemFileBlock(block[:], (idx+1)*10)
			n += int64(copy(p[n:limit], block[pos%10:]))
			continue
		}
		n += int64(copy(p[n:limit], tail[pos-c.full:]))
	}
	c.off += n
	return int(n), nil
}

func (c *memFileContent) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = c.off + offset
	case io.SeekEnd:
		abs = c.size + offset
	default:
		return 0, errors.New("memfile: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("memfile: negative position")
	}
	c.off = abs
	return abs, nil
}

// fillMemFileBlock 把 num 写成 9 位补零数字加换行，共 10 字节。
func fillMemFileBlock(b []byte, num int64) {
	for i := 8; i >= 0; i-- {
		b[i] = byte('0' + num%10)
		num /= 10
	}
	b[9] = '\n'
}
