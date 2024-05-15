package main

import (
	"io"
	"net/http"
)

// 100m
const MaxHTTPPayload = 100 * 1024 * 1024

var uploadService *UploaderService

func Uploader(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if IsCurl(r) {
			upCmd := "please use command `curl -X POST -H \"Content-Type: multipart/form-data\" -F \"file=@filename.fileext\" https://"
			upCmd = upCmd + *FlagDomain + "/upload`\n"
			io.WriteString(w, upCmd)
		} else {
			io.WriteString(w, GetUploadPage("上传文件", "/upload"))
		}
		return
	}

	localDir := "/root/vps/www/dl"
	if *FlagMock {
		localDir = "/Users/dilfish/go/src/simpleserver/dl"
	}

	if uploadService == nil {
		uploadService = NewUploadService(
			"https://"+*FlagDomain+"/dl/",
			localDir,
			"https://"+*FlagDomain+"/upload",
			MaxHTTPPayload,
			NeverExpire, 5)
	}

	uploadService.Handler(w, r)
}
