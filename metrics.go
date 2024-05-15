package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

func RunTcping() {
	metric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tcping_latency",
			Help: "tcping latency",
		}, []string{"isp", "region", "version", "district"})
	prometheus.MustRegister(metric)
	for addr, info := range IPMap {
		if info.Region != "上海" {
			continue
		}
		t := NewTcpingService(addr, info.ISP, info.Region, info.Version, info.District, 60, metric)
		go t.Start()
	}
	if *FlagTcping6 {
		for addr, info := range IPMap6 {
			if info.Region != "上海" {
				continue
			}
			t := NewTcpingService(addr, info.ISP, info.Region, info.Version, info.District, 60, metric)
			go t.Start()
		}
	}
}
