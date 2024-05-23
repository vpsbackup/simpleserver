package main

import (
	"log"
	"net"
	"net/http"
)

type ViewService struct {
	v4 *RangerMap
	v6 *IPv6Manager
}

func InitView(fn4, fn6 string) *ViewService {
	var vs ViewService
	rm := NewRangerMap()
	err := rm.ReadFile(fn4)
	if err != nil {
		log.Println("read ipv4 file error:", fn4, err)
		return nil
	}
	rm.Manage()
	vs.v4 = rm
	v6 := &IPv6Manager{}
	err = v6.ReadV6File(fn6)
	if err != nil {
		log.Println("read ipv6 file error:", fn6, err)
	}
	v6.SortV6()
	vs.v6 = v6
	return &vs
}

func (vs *ViewService) Handle(w http.ResponseWriter, r *http.Request) {
	ipStr := r.RequestURI
	log.Println("view request:", ipStr)
	if len(ipStr) < len("/ip/") {
		w.Write([]byte("bad ip"))
		return
	}
	ipStr = ipStr[len("/ip/"):]
	ip := net.ParseIP(ipStr)
	if ip == nil {
		w.Write([]byte("bad ip"))
		return
	}
	v4 := ip.To4()
	view := "默认-默认-默认-默认-默认-默认"
	if v4 != nil {
		view = vs.v4.Find(IpToU32(v4))
	} else {
		view = vs.v6.FindV6(ip.String())
	}
	w.Write([]byte(view))
	return
}
