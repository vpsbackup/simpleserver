package main

import (
	"errors"
	"log"
	"net"
	"net/http"
)

var GlobalViewService *ViewService

type ViewService struct {
	v4 *RangerMap
	v6 *IPv6Manager
}

func InitView(fn4, fn6 string) error {
	GlobalViewService = &ViewService{}
	rm := NewRangerMap()
	err := rm.ReadFile(fn4)
	if err != nil {
		log.Println("read ipv4 file error:", fn4, err)
		return err
	}
	rm.Manage()
	GlobalViewService.v4 = rm
	v6 := &IPv6Manager{}
	err = v6.ReadV6File(fn6)
	if err != nil {
		log.Println("read ipv6 file error:", fn6, err)
		return err
	}
	v6.SortV6()
	GlobalViewService.v6 = v6
	return nil
}

func (vs *ViewService) Handle(w http.ResponseWriter, r *http.Request) {
	ipStr := r.RequestURI
	log.Println("view request:", ipStr)
	if len(ipStr) < len("/ip/") {
		w.Write([]byte("bad ip"))
		return
	}
	ipStr = ipStr[len("/ip/"):]
	view, err := vs.Find(ipStr)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write([]byte(view))
	return
}

func (vs *ViewService) Find(ipStr string) (string, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		log.Println("bad ip:", ipStr)
		return "", errors.New("bad ip:" + ipStr)
	}
	v4 := ip.To4()
	view := "默认-默认-默认-默认-默认-默认"
	if v4 != nil {
		view = vs.v4.Find(IpToU32(v4))
	} else {
		view = vs.v6.FindV6(ip.String())
	}
	return view, nil
}
