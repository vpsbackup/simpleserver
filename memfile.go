package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func MemFileHandler(w http.ResponseWriter, r *http.Request) {
	u := r.RequestURI
	sizeStr := "100"
	if len(u) > len("/memfile/") {
		sizeStr = u[len("/memfile/"):]
	}
	memFile(w, r, sizeStr, false)
}

func memFile(w http.ResponseWriter, r *http.Request, sizeStr string, ifSleep bool) {
	size := uint64(100)
	size, _ = strconv.ParseUint(sizeStr, 10, 32)
	if size <= 0 {
		size = 100
	}
	// 999m
	if size > 999_999_999 {
		size = 100
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", strconv.FormatInt(int64(size), 10))

	str := ""
	wSize := 0
	for size >= 10 {
		wSize = wSize + 10
		size = size - 10
		// 10 bytes
		str = str + fmt.Sprintf("%09d\n", wSize)
		if len(str) > 50 {
			w.Write([]byte(str))
			str = ""
		}
		// 100byte 100ms
		// 1k/s
		if ifSleep && (wSize%100 == 0) {
			time.Sleep(time.Millisecond * 100)
		}
	}
	if size > 0 {
		for i := 1; i < int(size)+1; i++ {
			str = str + string('0'+i)
		}
	}
	if str != "" {
		w.Write([]byte(str))
	}
	str = ""
}
