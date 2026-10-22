package main

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
)

var startTime = time.Now()

type healthzInfo struct {
	OK          bool        `json:"ok"`
	UptimeS     int64       `json:"uptime_s"`
	VCSRevision string      `json:"vcs_revision"`
	Host        *hostInfo   `json:"host,omitempty"`
	CPU         *cpuInfo    `json:"cpu,omitempty"`
	Mem         *memInfo    `json:"mem,omitempty"`
	Swap        *swapInfo   `json:"swap,omitempty"`
	Disks       []diskInfo  `json:"disks,omitempty"`
	Go          *goInfo     `json:"go,omitempty"`
	Cert        *certInfo   `json:"cert,omitempty"`
	Vnstat      *vnstatInfo `json:"vnstat,omitempty"`
}

type hostInfo struct {
	Hostname      string `json:"hostname"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	Platform      string `json:"platform"`
	KernelVersion string `json:"kernel_version"`
	BootTime      int64  `json:"boot_time"`
	UptimeS       uint64 `json:"uptime_s"`
	Procs         uint64 `json:"procs"`
}

type cpuInfo struct {
	Num     int     `json:"num"`
	Percent float64 `json:"percent"`
	Load1   float64 `json:"load1"`
	Load5   float64 `json:"load5"`
	Load15  float64 `json:"load15"`
}

type memInfo struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Available   uint64  `json:"available"`
	UsedPercent float64 `json:"used_percent"`
}

type swapInfo struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
}

type diskInfo struct {
	Path        string  `json:"path"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

type goInfo struct {
	Version    string `json:"version"`
	Goroutines int    `json:"goroutines"`
}

type certInfo struct {
	Path     string    `json:"path"`
	Subject  string    `json:"subject,omitempty"`
	NotAfter time.Time `json:"not_after,omitempty"`
	DaysLeft int       `json:"days_left,omitempty"`
	Error    string    `json:"error,omitempty"`
}

type vnstatInfo struct {
	Available     bool          `json:"available"`
	Error         string        `json:"error,omitempty"`
	VnstatVersion string        `json:"vnstat_version,omitempty"`
	Interfaces    []vnstatIface `json:"interfaces,omitempty"`
}

type vnstatIface struct {
	Name    string      `json:"name"`
	Alias   string      `json:"alias,omitempty"`
	Updated string      `json:"updated,omitempty"`
	TodayRx int64       `json:"today_rx"`
	TodayTx int64       `json:"today_tx"`
	MonthRx int64       `json:"month_rx"`
	MonthTx int64       `json:"month_tx"`
	TotalRx int64       `json:"total_rx"`
	TotalTx int64       `json:"total_tx"`
	Days    []vnstatDay `json:"days,omitempty"`
}

type vnstatDay struct {
	Date string `json:"date"`
	Rx   int64  `json:"rx"`
	Tx   int64  `json:"tx"`
}

func vcsRevision() string {
	info := collectVersionInfo()
	if !info.OK {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			return s.Value
		}
	}
	return ""
}

// cachedVCSRevision reads build info once at startup instead of per request (#8).
var cachedVCSRevision = vcsRevision()

// #8: full host/cpu/mem/disk collection on every /healthz hit is expensive for
// an anonymous endpoint; cache the snapshot and refresh at most once per TTL.
var (
	healthzCacheMu  sync.Mutex
	healthzCacheAt  time.Time
	healthzCacheTTL = 30 * time.Second
	healthzCache    *healthzSnapshot
)

type healthzSnapshot struct {
	Host   *hostInfo
	CPU    *cpuInfo
	Mem    *memInfo
	Swap   *swapInfo
	Disks  []diskInfo
	Cert   *certInfo
	Vnstat *vnstatInfo
}

// resetHealthzCache drops the cached snapshot (test helper).
func resetHealthzCache() {
	healthzCacheMu.Lock()
	healthzCache = nil
	healthzCacheMu.Unlock()
}

func collectHost() *hostInfo {
	h, err := host.Info()
	if err != nil {
		log.Println("healthz host info error:", err)
		return nil
	}
	return &hostInfo{
		Hostname:      h.Hostname,
		OS:            h.OS,
		Arch:          runtime.GOARCH,
		Platform:      h.Platform + " " + h.PlatformVersion,
		KernelVersion: h.KernelVersion,
		BootTime:      int64(h.BootTime),
		UptimeS:       h.Uptime,
		Procs:         h.Procs,
	}
}

func collectCPU() *cpuInfo {
	n, err := cpu.Counts(true)
	if err != nil {
		log.Println("healthz cpu counts error:", err)
		return nil
	}
	c := &cpuInfo{Num: n}
	if pct, err := cpu.Percent(0, false); err == nil && len(pct) > 0 {
		c.Percent = pct[0]
	} else if err != nil {
		log.Println("healthz cpu percent error:", err)
	}
	if l, err := load.Avg(); err == nil {
		c.Load1, c.Load5, c.Load15 = l.Load1, l.Load5, l.Load15
	} else {
		log.Println("healthz load avg error:", err)
	}
	return c
}

func collectMem() (*memInfo, *swapInfo) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		log.Println("healthz mem error:", err)
		return nil, nil
	}
	mi := &memInfo{
		Total:       vm.Total,
		Used:        vm.Used,
		Available:   vm.Available,
		UsedPercent: vm.UsedPercent,
	}
	sw, err := mem.SwapMemory()
	if err != nil {
		log.Println("healthz swap error:", err)
		return mi, nil
	}
	return mi, &swapInfo{Total: sw.Total, Used: sw.Used, UsedPercent: sw.UsedPercent}
}

func collectDisks() []diskInfo {
	paths := []string{"/"}
	if Cfg.UploadDir != "" {
		paths = append(paths, Cfg.UploadDir)
	}
	seen := map[string]bool{}
	var out []diskInfo
	for _, p := range paths {
		if seen[p] {
			continue
		}
		seen[p] = true
		u, err := disk.Usage(p)
		if err != nil {
			log.Println("healthz disk usage error:", p, err)
			continue
		}
		out = append(out, diskInfo{
			Path:        p,
			Total:       u.Total,
			Used:        u.Used,
			Free:        u.Free,
			UsedPercent: u.UsedPercent,
		})
	}
	return out
}

func collectGo() *goInfo {
	return &goInfo{Version: runtime.Version(), Goroutines: runtime.NumGoroutine()}
}

// collectCert read the TLS cert from quic_cert_path (same file nginx serves)
// and report its expiry.
func collectCert() *certInfo {
	path := Cfg.QuicCertPath
	if path == "" {
		return &certInfo{Error: "quic_cert_path is not configured"}
	}
	info := &certInfo{Path: path}
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		info.Error = err.Error()
		return info
	}
	// cert files usually hold the certificate first, but skip any
	// non-CERTIFICATE blocks (e.g. the private key) to be safe.
	rest := pemBytes
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			info.Error = "no CERTIFICATE pem block found"
			return info
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		crt, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			info.Error = err.Error()
			return info
		}
		info.Subject = crt.Subject.CommonName
		info.NotAfter = crt.NotAfter
		info.DaysLeft = int(time.Until(crt.NotAfter).Hours() / 24)
		return info
	}
}

func collectVnstat() *vnstatInfo {
	vs, err := GetVnstat()
	if err != nil {
		return &vnstatInfo{Error: err.Error()}
	}
	return vnstatSummary(vs)
}

// vnstatSummary reduces full vnstat data to today/month/total per interface.
func vnstatSummary(vs *Vnstat) *vnstatInfo {
	info := &vnstatInfo{
		Available:     true,
		VnstatVersion: vs.VnstatVersion,
	}
	now := time.Now()
	today := now.Format("2006-01-02")
	thisMonth := now.Format("2006-01")
	for i := range vs.Interfaces {
		intf := &vs.Interfaces[i]
		it := vnstatIface{
			Name:    intf.Name,
			Alias:   intf.Alias,
			Updated: DateString(&intf.Updated.Date) + " " + TimeString(&intf.Updated.Time),
			TotalRx: intf.Traffic.Total.Rx,
			TotalTx: intf.Traffic.Total.Tx,
		}
		for _, d := range intf.Traffic.Day {
			ds := DateString(&d.Date)
			if ds == today {
				it.TodayRx, it.TodayTx = d.Rx, d.Tx
			}
		}
		for _, m := range intf.Traffic.Month {
			if DateString(&m.Date)[:7] == thisMonth {
				it.MonthRx, it.MonthTx = m.Rx, m.Tx
			}
		}
		days := intf.Traffic.Day
		if len(days) > 7 {
			days = days[len(days)-7:]
		}
		for _, d := range days {
			it.Days = append(it.Days, vnstatDay{Date: DateString(&d.Date), Rx: d.Rx, Tx: d.Tx})
		}
		info.Interfaces = append(info.Interfaces, it)
	}
	return info
}

// HealthzHandler reports liveness for probes: ok + uptime + build revision,
// plus host machine info, TLS cert expiry and vnstat traffic summary.
// The expensive collection is cached for healthzCacheTTL (#8); machine-locale
// fields (host/disks/cert) are only returned to authenticated requests (#8).
func HealthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	snap := cachedHealthzSnapshot()
	info := healthzInfo{
		OK:          true,
		UptimeS:     int64(time.Since(startTime).Seconds()),
		VCSRevision: cachedVCSRevision,
		Host:        snap.Host,
		CPU:         snap.CPU,
		Mem:         snap.Mem,
		Swap:        snap.Swap,
		Disks:       snap.Disks,
		Go:          collectGo(),
		Cert:        snap.Cert,
		Vnstat:      snap.Vnstat,
	}
	if !Authorized(r) {
		// hide hostname, kernel, disk usage and cert paths/errors from
		// unauthenticated callers; keep the probe-visible fields (#8)
		info.Host = nil
		info.Disks = nil
		info.Cert = nil
	}
	json.NewEncoder(w).Encode(info)
}

// cachedHealthzSnapshot returns the cached machine snapshot, refreshing it
// when the previous one is older than healthzCacheTTL.
func cachedHealthzSnapshot() *healthzSnapshot {
	healthzCacheMu.Lock()
	defer healthzCacheMu.Unlock()
	if healthzCache != nil && time.Since(healthzCacheAt) < healthzCacheTTL {
		return healthzCache
	}
	mi, si := collectMem()
	snap := &healthzSnapshot{
		Host:   collectHost(),
		CPU:    collectCPU(),
		Mem:    mi,
		Swap:   si,
		Disks:  collectDisks(),
		Cert:   collectCert(),
		Vnstat: collectVnstat(),
	}
	healthzCache = snap
	healthzCacheAt = time.Now()
	return snap
}
