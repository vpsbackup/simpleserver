package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

type VnstatDate struct {
	Day   int `json:"day"`
	Month int `json:"month"`
	Year  int `json:"year"`
}

type VnstatInterfaceCreated struct {
	Date VnstatDate `json:"date"`
}

type VnstatInterfaceTrafficDay struct {
	Date VnstatDate `json:"date"`
	Id   int        `json:"id"`
	Rx   int64      `json:"rx"`
	Tx   int64      `json:"tx"`
}

type VnstatTime struct {
	Hour   int `json:"hour"`
	Minute int `json:"minute"`
}

type VnstatInterfaceTrafficFiveMinute struct {
	Date VnstatDate `json:"date"`
	Id   int        `json:"id"`
	Rx   int64      `json:"rx"`
	Tx   int64      `json:"tx"`
	Time VnstatTime `json:"time"`
}

type VnstatTotal struct {
	Rx int64 `json:"rx"`
	Tx int64 `json:"tx"`
}

type VnstatInterfaceTraffic struct {
	Day        []VnstatInterfaceTrafficDay        `json:"day"`
	FiveMinute []VnstatInterfaceTrafficFiveMinute `json:"fiveminute"`
	// Hour is same struct as FiveMinute
	Hour []VnstatInterfaceTrafficFiveMinute `json:"hour"`
	// Month is same struct as Day
	Month []VnstatInterfaceTrafficDay `json:"month"`
	// Top is same struct as Day
	Top   []VnstatInterfaceTrafficDay `json:"top"`
	Total VnstatTotal                 `json:"total"`
	// Year is same struct as Day
	Year []VnstatInterfaceTrafficDay `json:"year"`
}

type VnstatUpdated struct {
	Date VnstatDate `json:"date"`
	Time VnstatTime `json:"time"`
}

type VnstatInterface struct {
	Alias   string                 `json:"alias"`
	Created VnstatInterfaceCreated `json:"created"`
	Name    string                 `json:"name"`
	Traffic VnstatInterfaceTraffic `json:"traffic"`
	Updated VnstatUpdated          `json:"updated"`
}

type Vnstat struct {
	JsonVersion   string            `json:"jsonversion"`
	VnstatVersion string            `json:"vnstatversion"`
	Interfaces    []VnstatInterface `json:"interfaces"`
}

func DateString(d *VnstatDate) string {
	return fmt.Sprintf("%4d-%02d-%02d", d.Year, d.Month, d.Day)
}

func TimeString(t *VnstatTime) string {
	return fmt.Sprintf("%02d:%02d:00", t.Hour, t.Minute)
}

func TrafficString(rxtx int64) string {
	if rxtx > 9223372036854775807 || rxtx < 0 {
		return "bad.traffic"
	}
	units := []string{"B", "K-", "M-", "G-", "T-", "P-"}
	str := ""
	uIdx := 0
	for {
		if rxtx >= 1024 {
			resid := rxtx % 1024
			uIdx = uIdx + 1
			if uIdx == len(units) {
				// 最后一次
				str = strconv.FormatInt(int64(rxtx), 10) + units[uIdx-1] + str
				return str
			} else {
				// 只加 resid
				rxtx = rxtx / 1024
				str = strconv.FormatInt(int64(resid), 10) + units[uIdx-1] + str
			}
		} else {
			break
		}
	}
	if rxtx > 0 {
		str = strconv.FormatInt(int64(rxtx), 10) + units[uIdx] + str
	}
	return str
}

// ParseVnstatJson parses the output of `vnstat --json`.
func ParseVnstatJson(bt []byte) (*Vnstat, error) {
	var vs Vnstat
	if err := json.Unmarshal(bt, &vs); err != nil {
		return nil, err
	}
	if len(vs.Interfaces) == 0 {
		return nil, fmt.Errorf("vnstat json has no interfaces")
	}
	return &vs, nil
}

// GetVnstat runs `vnstat --json` with a short timeout and caches the
// parsed result for vnstatCacheTTL to avoid forking on every probe.
func GetVnstat() (*Vnstat, error) {
	vnstatCacheMu.Lock()
	defer vnstatCacheMu.Unlock()
	if vnstatCache != nil && time.Since(vnstatCacheAt) < vnstatCacheTTL {
		return vnstatCache, vnstatCacheErr
	}
	ctx, cancel := context.WithTimeout(context.Background(), vnstatCmdTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "vnstat", "--json").Output()
	if err != nil {
		log.Println("run vnstat error:", err)
		vnstatCache = nil
		vnstatCacheErr = err
		vnstatCacheAt = time.Now()
		return nil, err
	}
	vs, err := ParseVnstatJson(out)
	if err != nil {
		log.Println("parse vnstat json error:", err)
		vnstatCache = nil
		vnstatCacheErr = err
		vnstatCacheAt = time.Now()
		return nil, err
	}
	vnstatCache = vs
	vnstatCacheErr = nil
	vnstatCacheAt = time.Now()
	return vs, nil
}

const (
	vnstatCacheTTL   = time.Minute
	vnstatCmdTimeout = 2 * time.Second
)

var (
	vnstatCacheMu  sync.Mutex
	vnstatCache    *Vnstat
	vnstatCacheAt  time.Time
	vnstatCacheErr error
)
