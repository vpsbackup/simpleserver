package main

import (
	"fmt"
	dio "github.com/dilfish/tools/io"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const NeverExpire = time.Duration(time.Hour * 24 * 365)

type UploaderService struct {
	MaxSize     int64
	MaxMem      int64
	Curr        int64
	BasePath    string
	BaseURL     string
	JumpBackURL string
	NameLen     int
	Expire      time.Duration
	Lock        sync.Mutex
	Map         map[string]time.Time
}

// WriteFile write reader into file
func (u *UploaderService) WriteFile(name string, rc io.Reader) (int64, string, error) {
	ext := filepath.Ext(name)
	if ext == "" {
		ext = ".noext"
	}
	name = dio.RandStr(u.NameLen) + ext
	fn := u.BasePath + "/" + name
	file, err := os.Create(fn)
	if err != nil {
		log.Println("create file name", err, "name", name)
		return 0, "", err
	}
	defer file.Close()
	u.Lock.Lock()
	defer u.Lock.Unlock()
	u.Map[fn] = time.Now().Add(u.Expire)
	log.Println("upload file", "file name", fn, "path", u.Map[fn])
	n, err := io.Copy(file, rc)
	return n, name, err
}

// Handler return page if get
// and write file into disk if post
func (u *UploaderService) Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		io.WriteString(w, "Not Supported")
		return
	}
	err := r.ParseMultipartForm(u.MaxMem)
	if err != nil {
		io.WriteString(w, "Read File Error:"+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Println("parse file error:", err)
		io.WriteString(w, "Read File error:"+err.Error())
		return
	}
	if u.Curr+header.Size > u.MaxSize {
		log.Println("too many write", "curr", u.Curr, "size", header.Size, "max size", u.MaxSize)
		msg := fmt.Sprintf("curr: %d, max: %d", u.Curr, u.MaxSize)
		io.WriteString(w, "Too many write: "+msg)
		return
	}
	defer file.Close()
	n, name, err := u.WriteFile(header.Filename, file)
	if err != nil {
		log.Println("write file", err)
		io.WriteString(w, "write file error"+err.Error())
		return
	}
	u.Curr = u.Curr + n
	show := `<!doctype html>
<html lang="zh-cmn-Hans">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
    <title>上传成功</title>
    <style>
      body { margin: 0; min-height: 100dvh; background: #fafaf9; color: #1c1917;
        font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
        display: grid; place-items: center; }
      .card { background: #fff; border: 1px solid #e7e5e4; border-radius: 16px; padding: 24px;
        max-width: 560px; width: calc(100% - 32px); box-shadow: 0 1px 2px rgba(0,0,0,0.04); }
      h1 { margin: 0 0 12px; font-size: 20px; }
      p { margin: 8px 0; font-size: 14px; line-height: 1.7; word-break: break-all; }
      a { color: #2563eb; }
    </style>
  </head>
  <body>
    <div class="card">
      <h1>上传成功 ✔</h1>
      <p>文件访问地址：<a href="` + u.BaseURL + name + `">` + u.BaseURL + name + `</a></p>`
	if u.JumpBackURL != "" {
		show = show + `      <p>或者返回<a href="` + u.JumpBackURL + `">上传页面</a>继续。</p>`
	}
	show = show + `    </div>
  </body>
</html>`
	if IsCurl(r) {
		io.WriteString(w, u.BaseURL+name+"\n")
	} else {
		io.WriteString(w, show)
	}
	return
}

func NewUploadService(baseURL, basePath, jump string, maxSize, maxTotal int64, expire time.Duration, nameLen int) *UploaderService {
	var u UploaderService
	u.MaxSize = maxTotal
	u.MaxMem = maxSize
	u.BasePath = basePath
	u.BaseURL = baseURL
	u.JumpBackURL = jump
	u.NameLen = nameLen
	if expire < time.Minute {
		expire = time.Minute
	}
	u.Expire = expire
	if expire != NeverExpire {
		log.Println("u.Expire", "expire", expire)
	}
	if u.NameLen < 1 {
		u.NameLen = 10
	}
	u.Map = make(map[string]time.Time)
	if expire != NeverExpire {
		go u.Patrol()
	}
	return &u
}

func (u *UploaderService) Patrol() {
	for {
		time.Sleep(time.Minute)
		tbd := []string{}
		u.Lock.Lock()
		for k, v := range u.Map {
			if v.Before(time.Now()) {
				tbd = append(tbd, k)
			}
		}
		u.Lock.Unlock()
		for _, tb := range tbd {
			os.Remove(tb)
			log.Println("uploader service remove", "file name", tb)
			delete(u.Map, tb)
		}
	}
}

func GetUploadPage(title, path string) string {
	var page = `<!doctype html>
<html lang="zh-cmn-Hans">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
    <meta name="theme-color" content="#fafaf9">
    <title>` + title + `</title>
    <style>
      :root {
        --bg: #fafaf9;
        --card: #ffffff;
        --line: #e7e5e4;
        --text: #1c1917;
        --muted: #78716c;
        --accent: #2563eb;
        --accent-text: #ffffff;
        --btn-bg: #f5f5f4;
      }
      * { box-sizing: border-box; }
      body {
        margin: 0;
        min-height: 100dvh;
        background: var(--bg);
        color: var(--text);
        font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
      }
      .page { max-width: 640px; margin: 0 auto; padding: max(28px, env(safe-area-inset-top)) 16px 32px; }
      .card {
        background: var(--card);
        border: 1px solid var(--line);
        border-radius: 16px;
        padding: 22px;
        box-shadow: 0 1px 2px rgba(0,0,0,0.04);
        margin-bottom: 16px;
      }
      h1 { margin: 0 0 16px; font-size: 20px; }
      .file-line { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
      input[type="file"] { font-size: 14px; color: var(--muted); flex: 1; min-width: 200px; }
      input[type="submit"] {
        appearance: none;
        border: 0;
        border-radius: 10px;
        background: var(--accent);
        color: var(--accent-text);
        font-size: 14px;
        font-weight: 600;
        padding: 10px 22px;
        cursor: pointer;
        min-height: 40px;
      }
      input[type="submit"]:hover { background: #1d4fd8; }
      .hint { color: var(--muted); font-size: 13px; line-height: 1.8; margin: 0; word-break: break-all; }
      .hint code { color: var(--accent); }
    </style>
  </head>
  <body>
    <div class="page">
      <div class="card">
        <h1>` + title + `</h1>
        <form action="` + path + `" method="post" enctype="multipart/form-data">
          <div class="file-line">
            <input type="file" name="file">
            <input type="submit" value="上传">
          </div>
        </form>
      </div>
      <div class="card">
        <p class="hint">累积上传最多 1G，单次最大 10M。<br>
        curl 也行：curl -X POST -H "Content-Type: multipart/form-data" -F "file=@filename.fileext" https://` + Cfg.Domain + `/upload</p>
      </div>
    </div>
  </body>
</html>
`
	return page
}
