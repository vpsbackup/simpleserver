package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

// 100m
var MaxHTTPPayload = int64(100 * 1024 * 1024)

// 10g
var MaxTotalFileSize = int64(10 * 1024 * 1024 * 1024)

var uploadService *UploaderService

// getUploader lazily build the upload service for both upload and manager APIs.
func getUploader() *UploaderService {
	if uploadService == nil {
		uploadService = NewUploadService(
			"https://"+Cfg.Domain+"/dl/",
			Cfg.UploadDir,
			"https://"+Cfg.Domain+"/upload",
			MaxHTTPPayload,
			MaxTotalFileSize,
			NeverExpire, 5)
	}
	return uploadService
}

// ApiFilesList list files in upload_dir as json
// get only
func ApiFilesList(w http.ResponseWriter, r *http.Request) {
	log.Println("request is:", r.Method, r.RequestURI)
	if r.Method != http.MethodGet {
		http.Error(w, "bad method for file list", http.StatusMethodNotAllowed)
		return
	}
	files, total, err := getUploader().ListFiles()
	if err != nil {
		log.Println("list files error:", err)
		http.Error(w, "list files error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if files == nil {
		files = []UploadedFileInfo{}
	}
	writeJson(w, map[string]interface{}{"list": files, "count": len(files), "total_size": total})
}

// ApiFilesDelete delete one uploaded file by name
// post only
func ApiFilesDelete(w http.ResponseWriter, r *http.Request) {
	log.Println("request is:", r.Method, r.RequestURI)
	if r.Method != http.MethodPost {
		http.Error(w, "bad method for file delete", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, "bad json body", http.StatusBadRequest)
		return
	}
	if err := getUploader().DeleteFile(body.Name); err != nil {
		log.Println("delete file error:", body.Name, err)
		http.Error(w, "delete file error: "+err.Error(), http.StatusBadRequest)
		return
	}
	writeJson(w, map[string]interface{}{"ok": true})
}

func Uploader(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if IsCurl(r) {
			upCmd := "please use command `curl -X POST -H \"Content-Type: multipart/form-data\" -F \"file=@filename.fileext\" https://"
			upCmd = upCmd + Cfg.Domain + "/upload`\n"
			io.WriteString(w, upCmd)
		} else {
			io.WriteString(w, GetUploadPage("上传 · 文件", "/upload"))
		}
		return
	}

	getUploader().Handler(w, r)
}
