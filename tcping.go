package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"log"
	"net"
	"time"
)

type TcpingService struct {
	addr     string
	isp      string
	region   string
	interval int
	version  string
	district string
	metric   *prometheus.GaugeVec
}

func NewTcpingService(addr, isp, region, version, district string, interval int, metric *prometheus.GaugeVec) *TcpingService {
	return &TcpingService{
		addr:     addr,
		isp:      isp,
		region:   region,
		version:  version,
		interval: interval,
		district: district,
		metric:   metric,
	}
}

// Start forever
func (s *TcpingService) Start() {
	var count int
	for {
		time.Sleep(time.Second * time.Duration(s.interval))
		elp := tcping(s.addr)
		if elp != 0 {
			s.metric.With(prometheus.Labels{"version": s.version,
				"isp":      s.isp,
				"region":   s.region,
				"district": s.district}).Set(float64(elp))
		}
		count = count + 1
		if count > 100 {
			count = 0
			log.Println("tcping result:", s.addr, s.isp, s.region, elp)
		}
	}
}

func tcping(addr string) int64 {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, time.Duration(1500)*time.Millisecond)
	if err != nil {
		log.Println("dial addr error:", addr, err)
		return 0
	}
	defer conn.Close()
	elapsed := time.Since(start).Milliseconds()
	return elapsed
}
