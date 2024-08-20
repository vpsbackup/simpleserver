package main

import (
	"strings"
	"log"
	"io"
	"encoding/json"
	"net/http"
)

// PeersDomainList
// just domain list, function will add https:// and /path
var PeersDomainList []string

func InitPeers() {
	PeersDomainList = strings.Split(*FlagPeers, ",")
	log.Println("peers domain list:", PeersDomainList)
	return
}

type PageStructCh struct {
	Page PageStruct
	Err error
	Domain string
}

func PeersGetVnstat() []PageStruct {
	path := "/api/vnstat"
	if len(PeersDomainList) == 0 {
		return nil
	}
	var psl []PageStruct
	ch := make(chan PageStructCh)
	for _, domain := range PeersDomainList {
		go func(c chan PageStructCh, domain string) {
			var resp PageStruct
			var ret PageStructCh
			ret.Domain = domain
			err := DoGet("https://" + domain + path, &resp)
			if err != nil {
				log.Println("get vnstat error:", domain, err)
				ret.Err = err
			} else {
				ret.Page = resp
			}
			c <- ret
			return
		}(ch, domain)
	}
	for _, _ = range PeersDomainList {
		ret := <- ch
		if ret.Err != nil {
			log.Println("peers do vnstat error:", ret.Domain, ret.Err)
			continue
		}
		psl = append(psl, ret.Page)
		
	}
	return psl
}

func DoGet(u string, ret interface{}) error {
	resp, err := http.Get(u)
	if err != nil {
		log.Println("http get error:", u, err)
		return err
	}
	defer resp.Body.Close()
	bt, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("http get read all error:", u, err)
		return err
	}
	err = json.Unmarshal(bt, ret)
	if err != nil {
		log.Println("unmarshal error:", string(bt), err)
		return err
	}
	return nil
}
