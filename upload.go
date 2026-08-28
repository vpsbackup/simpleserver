package main

import (
	"io"
	"net/http"
)

// 100m
var MaxHTTPPayload = int64(100 * 1024 * 1024)

// 10g
var MaxTotalFileSize = int64(10 * 1024 * 1024 * 1024)

var uploadService *UploaderService

func Uploader(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if IsCurl(r) {
			upCmd := "please use command `curl -X POST -H \"Content-Type: multipart/form-data\" -F \"file=@filename.fileext\" https://"
			upCmd = upCmd + Cfg.Domain + "/upload`\n"
			io.WriteString(w, upCmd)
		} else {
			io.WriteString(w, GetUploadPage("上传文件", "/upload"))
		}
		return
	}

	if uploadService == nil {
		uploadService = NewUploadService(
			"https://"+Cfg.Domain+"/dl/",
			Cfg.UploadDir,
			"https://"+Cfg.Domain+"/upload",
			MaxHTTPPayload,
			MaxTotalFileSize,
			NeverExpire, 5)
	}

	uploadService.Handler(w, r)
}
