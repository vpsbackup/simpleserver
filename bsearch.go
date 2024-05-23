package main

import (
	"errors"
	"log"
	"net"
	"sort"
	"strings"

	dio "github.com/dilfish/tools/io"
	dnet "github.com/dilfish/tools/net"
)

// / ipv6 可用地址集中在 2xxx:xxxx:xxxx:xxxx:xxxx:xxxx:xxxx:xxxx

// IPV6Range 将 IPv6 分为2个 uint64 表示
// start - end 表示一个 IP 段
type IPV6Range struct {
	StartPrefix  uint64
	StartPostfix uint64
	EndPrefix    uint64
	EndPostfix   uint64
	View         string
}

type IPRangeArray []IPV6Range

// Len 实现 Sort 接口
func (r IPRangeArray) Len() int {
	return len(r)
}

// Less 实现 Sort 接口
func (r IPRangeArray) Less(i, j int) bool {
	if r[i].StartPrefix < r[j].StartPrefix {
		return true
	}
	if r[i].StartPrefix > r[j].StartPrefix {
		return false
	}
	if r[i].StartPostfix < r[j].StartPostfix {
		return true
	}
	return false
}

// Swap 实现 Sort 接口
func (r IPRangeArray) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

type IPv6Manager struct {
	IPRangeList []IPV6Range
}

// var IPRangeList []IPV6Range

func (ip6m *IPv6Manager) cb6(line string) error {
	array := strings.Split(line, " ")
	if len(array) != 3 {
		return errors.New("bad format:" + line)
	}
	start := net.ParseIP(array[0])
	if start == nil {
		return errors.New("bad ip:" + array[0])
	}
	if start.To4() != nil {
		return errors.New("bad start ip:" + start.String())
	}
	end := net.ParseIP(array[1])
	if end == nil {
		return errors.New("bad end ip:" + array[1])
	}
	if end.To4() != nil {
		return errors.New("bad end ip:" + end.String())
	}
	var ir IPV6Range
	ir.View = array[2]
	ir.StartPrefix, ir.StartPostfix = IPv6ToUint64(start.String())
	ir.EndPrefix, ir.EndPostfix = IPv6ToUint64(end.String())
	ip6m.IPRangeList = append(ip6m.IPRangeList, ir)
	return nil
}

func (ip6m *IPv6Manager) ReadV6File(fn string) error {
	return dio.ReadLine(fn, ip6m.cb6)
}

func (ip6m *IPv6Manager) SortV6() {
	sort.Sort(IPRangeArray(ip6m.IPRangeList))
	log.Println("v6 ip range has:", len(ip6m.IPRangeList))
}

// Judge 如果 ir 大于等于 prefix+postfix，返回 true
func Judge(prefix, postfix uint64, ir IPV6Range) bool {
	if ir.EndPrefix > prefix {
		return true
	}
	if ir.EndPrefix == prefix && ir.EndPostfix >= postfix {
		return true
	}
	return false
}

func (ip6m *IPv6Manager) FindV6(ip string) string {
	prefix, postfix := IPv6ToUint64(ip)
	// 二分法查找
	idx := sort.Search(len(ip6m.IPRangeList), func(i int) bool {
		ir := ip6m.IPRangeList[i]
		ok := Judge(prefix, postfix, ir)
		// log.Printf("search 0x%x-0x%x in 0x%x-0x%x to 0x%x-0x%x with %t\n", prefix, postfix, ir.StartPrefix, ir.StartPostfix, ir.EndPrefix, ir.EndPostfix, ok)
		return ok
	})
	if len(ip6m.IPRangeList) <= idx {
		log.Println("bad search, this should not happen:", ip, idx)
		return "默认-默认-默认-默认-默认-默认"
	}
	return ip6m.IPRangeList[idx].View
}

func IPv6ToUint64(ipstr string) (uint64, uint64) {
	return dnet.IPv62Num(ipstr)
}

func IpToU32(ip net.IP) uint32 {
	return dnet.IP2Num(ip.String())
}
