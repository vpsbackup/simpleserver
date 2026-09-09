package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	dio "github.com/dilfish/tools/io"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// UploadedFileInfo is one entry of the upload dir for the manager API.
type UploadedFileInfo struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modtime"`
	URL     string    `json:"url"`
	MD5     string    `json:"md5"`
}

// ListFiles read the upload dir from disk, newest first.
func (u *UploaderService) ListFiles() ([]UploadedFileInfo, int64, error) {
	entries, err := os.ReadDir(u.BasePath)
	if err != nil {
		return nil, 0, err
	}
	out := make([]UploadedFileInfo, 0, len(entries))
	var total int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			log.Println("file info error:", e.Name(), err)
			continue
		}
		file, err := os.Open(filepath.Join(u.BasePath, e.Name()))
		if err != nil {
			log.Println("open file for md5 error:", e.Name(), err)
			continue
		}
		hash := md5.New()
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil {
			if copyErr != nil {
				log.Println("calculate file md5 error:", e.Name(), copyErr)
			} else {
				log.Println("close file after md5 error:", e.Name(), closeErr)
			}
			continue
		}
		total += info.Size()
		out = append(out, UploadedFileInfo{
			Name:    e.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			URL:     u.BaseURL + e.Name(),
			MD5:     fmt.Sprintf("%x", hash.Sum(nil)),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ModTime.After(out[j].ModTime) })
	return out, total, nil
}

// DeleteFile remove one uploaded file from disk and quota tracking.
func (u *UploaderService) DeleteFile(name string) error {
	if name == "" || name == "." || name == ".." ||
		filepath.Base(name) != name || strings.ContainsAny(name, "/\\") {
		return errors.New("bad file name")
	}
	fn := u.BasePath + "/" + name
	info, err := os.Stat(fn)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("not a regular file")
	}
	if err := os.Remove(fn); err != nil {
		return err
	}
	u.Lock.Lock()
	delete(u.Map, fn)
	u.Curr -= info.Size()
	if u.Curr < 0 {
		u.Curr = 0
	}
	u.Lock.Unlock()
	return nil
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
			if info, err := os.Stat(tb); err == nil {
				u.Lock.Lock()
				u.Curr -= info.Size()
				if u.Curr < 0 {
					u.Curr = 0
				}
				u.Lock.Unlock()
			}
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
      --danger: #b91c1c;
      --btn-bg: #f5f5f4;
      --btn-line: #e7e5e4;
      --input-bg: #fafaf9;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100dvh;
      background: var(--bg);
      color: var(--text);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
    }
    .page { max-width: 760px; margin: 0 auto; padding: max(20px, env(safe-area-inset-top)) 16px max(24px, env(safe-area-inset-bottom)); }
    .topbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 20px; }
    .brand { display: flex; align-items: center; gap: 10px; font-weight: 650; letter-spacing: 0.02em; }
    .brand-mark {
      width: 34px; height: 34px; border-radius: 10px; display: grid; place-items: center;
      background: var(--accent); color: var(--accent-text); font-size: 15px;
    }
    .brand-sub { display: block; font-size: 12px; color: var(--muted); font-weight: 400; }
    .btn {
      appearance: none; border: 1px solid var(--btn-line); border-radius: 10px;
      background: var(--btn-bg); color: var(--text);
      padding: 8px 14px; font-size: 14px; cursor: pointer; text-decoration: none;
      display: inline-flex; align-items: center; gap: 6px; min-height: 38px;
      transition: background .15s, border-color .15s;
    }
    .btn:hover { background: #ebebe9; }
    .btn-primary { background: var(--accent); color: var(--accent-text); border-color: var(--accent); font-weight: 600; }
    .btn-primary:hover { background: #1d4fd8; }
    .btn-primary:disabled { opacity: .5; cursor: not-allowed; }
    .btn-danger { color: var(--danger); }
    .btn-danger:hover { background: #fdecec; }
    .card {
      background: var(--card); border: 1px solid var(--line); border-radius: 16px;
      padding: 18px; box-shadow: 0 1px 2px rgba(0,0,0,0.04);
    }
    .composer { margin-bottom: 22px; }
    .file-line { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; margin-bottom: 12px; }
    .file-label {
      border: 1px dashed var(--line); border-radius: 10px; background: var(--input-bg);
      color: var(--muted); padding: 10px 16px; cursor: pointer; font-size: 14px;
      display: inline-flex; align-items: center; gap: 8px; min-height: 40px;
    }
    .file-label:hover { border-color: var(--accent); color: var(--text); }
    .file-label input { display: none; }
    .file-name { color: var(--muted); font-size: 13px; word-break: break-all; flex: 1; min-width: 140px; }
    .progress { height: 6px; border-radius: 999px; background: var(--input-bg); overflow: hidden; margin-bottom: 12px; display: none; }
    .progress .bar { height: 100%; width: 0; background: var(--accent); transition: width .2s; }
    .composer-foot { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
    .hint { font-size: 12px; color: var(--muted); word-break: break-all; }
    .hint code { color: var(--accent); }
    .list-head {
      display: flex; align-items: center; justify-content: space-between;
      margin: 0 2px 12px; color: var(--muted); font-size: 13px;
    }
    .msg-list { display: flex; flex-direction: column; gap: 12px; }
    .file-card { padding: 14px 16px; }
    .file-top { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 6px; font-size: 12px; color: var(--muted); flex-wrap: wrap; }
    .file-title { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 14px; word-break: break-all; margin-bottom: 12px; }
    .file-title a { color: var(--text); text-decoration: none; border-bottom: 1px dotted var(--line); }
    .file-title a:hover { color: var(--accent); }
    .msg-actions { display: flex; gap: 8px; flex-wrap: wrap; }
    .msg-actions .btn { padding: 6px 12px; font-size: 13px; min-height: 34px; }
    .empty { text-align: center; color: var(--muted); padding: 32px 0; font-size: 14px; }
    .empty a { color: var(--accent); }
    .toast {
      position: fixed; left: 50%; bottom: 28px; transform: translateX(-50%) translateY(20px);
      background: var(--card); border: 1px solid var(--line); color: var(--text);
      padding: 10px 18px; border-radius: 12px; font-size: 14px; opacity: 0; pointer-events: none;
      transition: opacity .2s, transform .2s; z-index: 99; max-width: 86vw;
      box-shadow: 0 8px 24px rgba(0,0,0,0.12);
    }
    .toast.show { opacity: 1; transform: translateX(-50%) translateY(0); }
    .toast.ok { border-color: rgba(37,99,235,0.45); }
    .toast.err { border-color: rgba(185,28,28,0.5); }
    @media (max-width: 480px) {
      .page { padding-left: 12px; padding-right: 12px; }
      .card { padding: 14px; border-radius: 13px; }
      .msg-actions .btn { flex: 1; justify-content: center; }
      .composer-foot { flex-direction: column; align-items: stretch; }
      .composer-foot .btn-primary { justify-content: center; }
      .file-line { flex-direction: column; align-items: stretch; }
      .file-label { justify-content: center; }
    }
  </style>
</head>
<body>
<div class="page">
  <div class="topbar">
    <div class="brand">
      <span class="brand-mark">⇪</span>
      <span>` + title + `<span class="brand-sub">单次最大 10M · 累积 10G</span></span>
    </div>
    <button class="btn" id="refresh-btn" type="button">刷新</button>
  </div>

  <form class="card composer" id="upload-form" action="` + path + `" method="post" enctype="multipart/form-data">
    <div class="file-line">
      <label class="file-label">📎 选择文件
        <input type="file" name="file" id="file-input" required>
      </label>
      <span class="file-name" id="file-name">未选择文件</span>
    </div>
    <div class="progress" id="progress"><div class="bar" id="progress-bar"></div></div>
    <div class="composer-foot">
      <span class="hint">curl 也行：curl -X POST -H "Content-Type: multipart/form-data" -F "file=@f.ext" -H "X-Site-Token: 密码" https://` + Cfg.Domain + `/upload</span>
      <button class="btn btn-primary" type="submit" id="upload-btn">上传</button>
    </div>
  </form>

  <div class="list-head">
    <span id="list-count"></span>
    <span>按时间倒序</span>
  </div>
  <div class="msg-list" id="file-list"><div class="empty">加载中…</div></div>
</div>

<div class="toast" id="toast"></div>

<script>
(function () {
  'use strict';

  var listEl = document.getElementById('file-list');
  var countEl = document.getElementById('list-count');
  var form = document.getElementById('upload-form');
  var input = document.getElementById('file-input');
  var fileNameEl = document.getElementById('file-name');
  var uploadBtn = document.getElementById('upload-btn');
  var progressEl = document.getElementById('progress');
  var barEl = document.getElementById('progress-bar');
  var toastEl = document.getElementById('toast');
  var toastTimer = null;

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;').replace(/'/g, '&#39;');
  }

  function toast(msg, ok) {
    toastEl.textContent = msg;
    toastEl.className = 'toast show ' + (ok ? 'ok' : 'err');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(function () { toastEl.className = 'toast'; }, 2200);
  }

  function fmtTime(iso) {
    var d = new Date(iso);
    if (isNaN(d.getTime())) return iso;
    return d.toLocaleString('zh-CN', {
      timeZone: 'Asia/Shanghai', hour12: false,
      year: 'numeric', month: '2-digit', day: '2-digit',
      hour: '2-digit', minute: '2-digit', second: '2-digit'
    });
  }

  function humanSize(n) {
    if (n < 1024) return n + ' B';
    var units = ['KB', 'MB', 'GB', 'TB'];
    var i = -1;
    do { n /= 1024; i++; } while (n >= 1024 && i < units.length - 1);
    return n.toFixed(n >= 100 ? 0 : 1) + ' ' + units[i];
  }

  function errMsg(res) {
    if (res.status === 401) return '请先在 /dilfish.html 登录';
    return '请求失败 (HTTP ' + res.status + ')';
  }

  function copyText(text) {
    if (navigator.clipboard && window.isSecureContext) {
      return navigator.clipboard.writeText(text);
    }
    return new Promise(function (resolve, reject) {
      var ta = document.createElement('textarea');
      ta.value = text;
      ta.style.position = 'fixed';
      ta.style.opacity = '0';
      document.body.appendChild(ta);
      ta.select();
      try {
        document.execCommand('copy') ? resolve() : reject(new Error('copy failed'));
      } catch (e) {
        reject(e);
      } finally {
        document.body.removeChild(ta);
      }
    });
  }

  function render(files, total) {
    countEl.textContent = '共 ' + files.length + ' 个文件 · ' + humanSize(total);
    if (!files.length) {
      listEl.innerHTML = '<div class="empty">还没有文件，传一个吧</div>';
      return;
    }
    var html = '';
    for (var i = 0; i < files.length; i++) {
      var f = files[i];
      var name = escapeHtml(f.name);
      html += '<article class="card file-card" data-name="' + name + '">'
        + '<div class="file-top"><span>' + escapeHtml(fmtTime(f.modtime)) + '</span><span>' + humanSize(f.size) + '</span></div>'
        + '<div class="file-title"><a href="' + escapeHtml(f.url) + '" target="_blank" rel="noopener">' + name + '</a></div>'
        + '<div class="file-top"><span>MD5</span><span class="file-md5">' + escapeHtml(f.md5 || '未知') + '</span></div>'
        + '<div class="msg-actions">'
        + '<button class="btn" type="button" data-act="copy">拷贝链接</button>'
        + '<a class="btn" href="' + escapeHtml(f.url) + '" target="_blank" rel="noopener">打开</a>'
        + '<button class="btn btn-danger" type="button" data-act="del">删除</button>'
        + '</div></article>';
    }
    listEl.innerHTML = html;
  }

  function loadList() {
    fetch('/api/files/list', { credentials: 'same-origin' })
      .then(function (res) {
        if (!res.ok) throw new Error(errMsg(res));
        var ct = res.headers.get('content-type') || '';
        if (ct.indexOf('json') === -1) throw new Error('请先在 /dilfish.html 登录');
        return res.json();
      })
      .then(function (data) { render(data.list || [], data.total_size || 0); })
      .catch(function (e) {
        countEl.textContent = '';
        listEl.innerHTML = '<div class="empty">' + escapeHtml(e.message)
          + '。<a href="/dilfish.html">去登录</a></div>';
      });
  }

  input.addEventListener('change', function () {
    fileNameEl.textContent = input.files.length ? input.files[0].name : '未选择文件';
  });

  form.addEventListener('submit', function (ev) {
    ev.preventDefault();
    if (!input.files.length) {
      toast('先选择一个文件', false);
      return;
    }
    var fd = new FormData();
    fd.append('file', input.files[0]);
    var xhr = new XMLHttpRequest();
    uploadBtn.disabled = true;
    progressEl.style.display = 'block';
    barEl.style.width = '0';
    xhr.upload.addEventListener('progress', function (ev2) {
      if (ev2.lengthComputable) {
        barEl.style.width = Math.round(ev2.loaded / ev2.total * 100) + '%';
      }
    });
    xhr.addEventListener('load', function () {
      uploadBtn.disabled = false;
      setTimeout(function () { progressEl.style.display = 'none'; }, 400);
      if (xhr.status === 200) {
        input.value = '';
        fileNameEl.textContent = '未选择文件';
        toast('上传成功', true);
        loadList();
      } else {
        toast('上传失败：' + errMsg(xhr), false);
      }
    });
    xhr.addEventListener('error', function () {
      uploadBtn.disabled = false;
      progressEl.style.display = 'none';
      toast('上传失败：网络错误', false);
    });
    xhr.open('POST', form.getAttribute('action'));
    xhr.withCredentials = true;
    xhr.send(fd);
  });

  listEl.addEventListener('click', function (ev) {
    var btn = ev.target.closest('button[data-act]');
    if (!btn) return;
    var card = btn.closest('.file-card');
    var name = card.getAttribute('data-name');

    if (btn.getAttribute('data-act') === 'copy') {
      var link = card.querySelector('.file-title a').href;
      copyText(link)
        .then(function () { toast('链接已拷贝', true); })
        .catch(function (e) { toast('拷贝失败：' + e.message, false); });
      return;
    }

    if (btn.getAttribute('data-act') === 'del') {
      if (!confirm('确定删除 ' + name + ' ？删除后下载链接立即失效。')) return;
      fetch('/api/files/delete', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: name })
      })
        .then(function (res) {
          if (!res.ok) throw new Error(errMsg(res));
          toast('已删除', true);
          loadList();
        })
        .catch(function (e) { toast('删除失败：' + e.message, false); });
    }
  });

  document.getElementById('refresh-btn').addEventListener('click', loadList);
  loadList();
})();
</script>
</body>
</html>
`
	return page
}
