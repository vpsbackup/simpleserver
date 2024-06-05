package main

import (
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

type IPInfo struct {
	Key   string
	Value string
}

// CFIPHandler handles cloudflare HTTP headers info
func CFIPHandler(w http.ResponseWriter, r *http.Request) {
	var headerList []string
	var kvList []IPInfo
	for k, _ := range r.Header {
		headerList = append(headerList, k)
	}
	sort.Strings(headerList)
	for _, hdr := range headerList {
		var ii IPInfo
		vv := strings.Join(r.Header[hdr], ",")
		ii.Key = hdr
		ii.Value = vv
		kvList = append(kvList, ii)
	}
	var ii IPInfo
	ii.Key = "X-Server"
	ii.Value = "iPhone 8 Plus"
	kvList = append([]IPInfo{ii}, kvList...)

	ii.Key = "X-Remote-IP"
	ipStr := r.RemoteAddr
	if *FlagBehindNginx {
		ipStr = r.Header.Get("X-Real-Ip")
	}
	ii.Value = ipStr
	kvList = append([]IPInfo{ii}, kvList...)

	view, _ := GlobalViewService.Find(ipStr)
	ii.Key = "X-IP-View"
	ii.Value = view
	kvList = append([]IPInfo{ii}, kvList...)

	ii.Key = "X-time"
	ii.Value = time.Now().Format("2006-01-02T15:04:05 -07:00:00")
	kvList = append([]IPInfo{ii}, kvList...)

	if IsCurl(r) {
		for _, kv := range kvList {
			w.Write([]byte(kv.Key + ": " + kv.Value + "\n"))
		}
		return
	}

	if r.Method != "GET" {
		bt, _ := json.Marshal(kvList)
		w.Write(bt)
		return
	}
	RenderHTML("ip.index.html", w, kvList)
	return
}

func RenderHTML(fn string, wr io.Writer, data interface{}) error {
	t, err := template.ParseFiles(fn)
	if err != nil {
		log.Println("parse file", "error", err, "file_name", fn)
		return err
	}
	return t.Execute(wr, data)
}
