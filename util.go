package main

import (
	"net/http"
	"strings"
)

func IsCurl(req *http.Request) bool {
	v := req.Header.Get("User-Agent")
	return strings.Contains(v, "curl/")
}
