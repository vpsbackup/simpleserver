package main

import (
	"log"
	"net/http"
)

func ApiHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("api handler request:", r.RequestURI)
	uri := r.RequestURI
	if len(uri) < len("/api/") {
		log.Println("bad api request uri:", uri)
		w.Write([]byte("bad api request"))
		return
	}
	task := r.URL.Path[len("/api/"):]
	switch task {
	case "t", "t/list":
		if r.Method == http.MethodGet {
			ApiMsgList(w, r)
			return
		}
		if r.Method == http.MethodPost {
			ApiMsgCreate(w, r)
			return
		}
	case "t/delete":
		if r.Method == http.MethodPost {
			ApiMsgDelete(w, r)
			return
		}
	case "files/list":
		if r.Method == http.MethodGet {
			ApiFilesList(w, r)
			return
		}
	// Delegated for every method so the handler itself owns the 405 answer;
	// a method check here would make that branch unreachable and turn a bad
	// method into "bad api request name" with a 200.
	case "files/upload":
		ApiFilesUpload(w, r)
		return
	case "files/delete":
		if r.Method == http.MethodPost {
			ApiFilesDelete(w, r)
			return
		}
	case "dns/query":
		if r.Method == http.MethodGet {
			DnsQueryHandler(w, r)
			return
		}
	case "dns/trace":
		if r.Method == http.MethodGet {
			DnsTraceHandler(w, r)
			return
		}
	}
	log.Println("unknown api:", uri)
	w.Write([]byte("bad api request name"))
}
