package main

import (
	"log"
	"net/http"
	"encoding/json"
)

func ApiHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("api handler request:", r.RequestURI)
	uri := r.RequestURI
	if len(uri) < len("/api/") {
		log.Println("bad api request uri:", uri)
		w.Write([]byte("bad api request"))
		return
	}
	task := uri[len("/api/"):]
	switch task {
		case "vnstat":
			data, err := GetVnstat()
			if err != nil {
				log.Println("get api vnstat error:", err)
				w.Write([]byte(err.Error()))
				return
			}
			bt, _ := json.Marshal(data)
			w.Write(bt)
			return
	}
	log.Println("unknown api:", uri)
	w.Write([]byte("bad api request name"))
}
