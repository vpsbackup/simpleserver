package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
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

type PageInfo struct {
	Key   string
	Value string
}

type PageTrafficInfo struct {
	Time string
	Rx   string
	Tx   string
}

type PageTraffic struct {
	Name    string
	Traffic []PageTrafficInfo
}

type PageStruct struct {
	InfoList    []PageInfo
	TrafficList []PageTraffic
}

func PrintVnstat(v *Vnstat) PageStruct {
	var ps PageStruct
	// ps.InfoList = append(ps.InfoList, PageInfo{Key: "vnstat version", Value: v.VnstatVersion})
	// ps.InfoList = append(ps.InfoList, PageInfo{Key: "vnstat.json version", Value: v.JsonVersion})
	for _, intf := range v.Interfaces {
		name := intf.Name
		if intf.Alias != "" {
			name = name + "(" + intf.Alias + ")"
		}
		// ps.InfoList = append(ps.InfoList, PageInfo{Key: name + " created", Value: DateString(&intf.Created.Date)})
		ps.InfoList = append(ps.InfoList, PageInfo{Key: name + " updated", Value: DateString(&intf.Updated.Date) + " " + TimeString(&intf.Updated.Time)})

		var traffic PageTraffic

		traffic.Traffic = nil
		traffic.Name = "traffic of five minute of " + name
		// 最近12个5分钟
		if len(intf.Traffic.FiveMinute) > 12 {
			intf.Traffic.FiveMinute = intf.Traffic.FiveMinute[len(intf.Traffic.FiveMinute)-12:]
		}
		for i := len(intf.Traffic.FiveMinute) - 1; i >= 0; i-- {
			traffic.Traffic = append(traffic.Traffic, PageTrafficInfo{
				Time: DateString(&intf.Traffic.FiveMinute[i].Date) + " " + TimeString(&intf.Traffic.FiveMinute[i].Time),
				Rx:   TrafficString(intf.Traffic.FiveMinute[i].Rx),
				Tx:   TrafficString(intf.Traffic.FiveMinute[i].Tx),
			})
		}
		ps.TrafficList = append(ps.TrafficList, traffic)

		traffic.Traffic = nil
		traffic.Name = "traffic of day of " + name
		// 最近7天
		if len(intf.Traffic.Day) > 7 {
			intf.Traffic.Day = intf.Traffic.Day[len(intf.Traffic.Day)-7:]
		}
		for i := len(intf.Traffic.Day) - 1; i >= 0; i-- {
			traffic.Traffic = append(traffic.Traffic, PageTrafficInfo{
				Time: DateString(&intf.Traffic.Day[i].Date),
				Rx:   TrafficString(intf.Traffic.Day[i].Rx),
				Tx:   TrafficString(intf.Traffic.Day[i].Tx),
			})
		}
		ps.TrafficList = append(ps.TrafficList, traffic)

		traffic.Traffic = nil
		traffic.Name = "traffic of hour of " + name
		// 最近12小时
		if len(intf.Traffic.Hour) > 12 {
			intf.Traffic.Hour = intf.Traffic.Hour[len(intf.Traffic.Hour)-12:]
		}
		for i := len(intf.Traffic.Hour) - 1; i >= 0; i-- {
			traffic.Traffic = append(traffic.Traffic, PageTrafficInfo{
				Time: DateString(&intf.Traffic.Hour[i].Date) + " " + TimeString(&intf.Traffic.Hour[i].Time),
				Rx:   TrafficString(intf.Traffic.Hour[i].Rx),
				Tx:   TrafficString(intf.Traffic.Hour[i].Tx),
			})
		}
		ps.TrafficList = append(ps.TrafficList, traffic)

		// 最近3个月
		if len(intf.Traffic.Month) > 3 {
			intf.Traffic.Month = intf.Traffic.Month[len(intf.Traffic.Month)-3:]
		}
		traffic.Traffic = nil
		traffic.Name = "traffic of month of " + name
		for i := len(intf.Traffic.Month) - 1; i >= 0; i-- {
			traffic.Traffic = append(traffic.Traffic, PageTrafficInfo{
				Time: DateString(&intf.Traffic.Month[i].Date),
				Rx:   TrafficString(intf.Traffic.Month[i].Rx),
				Tx:   TrafficString(intf.Traffic.Month[i].Tx),
			})
		}
		ps.TrafficList = append(ps.TrafficList, traffic)

		traffic.Traffic = nil
		traffic.Name = "traffic of year of " + name
		for i := len(intf.Traffic.Year) - 1; i >= 0; i-- {
			traffic.Traffic = append(traffic.Traffic, PageTrafficInfo{
				Time: DateString(&intf.Traffic.Year[i].Date),
				Rx:   TrafficString(intf.Traffic.Year[i].Rx),
				Tx:   TrafficString(intf.Traffic.Year[i].Tx),
			})
		}
		ps.TrafficList = append(ps.TrafficList, traffic)

		traffic.Traffic = nil
		traffic.Name = "traffic of top of " + name
		// 最近3个 top
		if len(intf.Traffic.Top) > 3 {
			intf.Traffic.Top = intf.Traffic.Top[:3]
		}
		for _, top := range intf.Traffic.Top {
			traffic.Traffic = append(traffic.Traffic, PageTrafficInfo{
				Time: DateString(&top.Date),
				Rx:   TrafficString(top.Rx),
				Tx:   TrafficString(top.Tx),
			})
		}
		ps.TrafficList = append(ps.TrafficList, traffic)

		traffic.Traffic = nil
		traffic.Name = "total traffic of " + name
		traffic.Traffic = append(traffic.Traffic, PageTrafficInfo{
			Time: "all",
			Rx:   TrafficString(intf.Traffic.Total.Rx),
			Tx:   TrafficString(intf.Traffic.Total.Tx),
		})
		ps.TrafficList = append(ps.TrafficList, traffic)
	}
	return ps
}

func VnstatHandler(w http.ResponseWriter, r *http.Request) {
	file, err := os.Open("vnstat.log")
	if err != nil {
		log.Println("open error:", err)
		w.Write([]byte("no vnstat.log file"))
		return
	}
	defer file.Close()
	bt, _ := io.ReadAll(file)
	var vs Vnstat
	json.Unmarshal(bt, &vs)
	data := PrintVnstat(&vs)
	t, err := template.ParseFiles("vnstat.html")
	if err != nil {
		log.Println("parse file error", err)
		w.Write([]byte("no vnstat.html file"))
		return
	}
	err = t.Execute(w, data)
	if err != nil {
		log.Println("execute error:", err)
	}
}
