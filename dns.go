package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/miekg/dns"
)

const (
	dnsQueryTimeout      = 3 * time.Second
	dnsTraceTotalTimeout = 12 * time.Second
	dnsTraceQueryTimeout = 2 * time.Second
	dnsMaxTraceSteps     = 20
	dnsTraceResolver     = "1.1.1.1"
	dnsDefaultResolver   = "1.1.1.1"
	dnsPort              = "53"
)

var dnsTypes = map[string]uint16{
	"A":     dns.TypeA,
	"AAAA":  dns.TypeAAAA,
	"CNAME": dns.TypeCNAME,
	"NS":    dns.TypeNS,
	"MX":    dns.TypeMX,
	"TXT":   dns.TypeTXT,
	"SOA":   dns.TypeSOA,
	"SRV":   dns.TypeSRV,
	"PTR":   dns.TypePTR,
	"CAA":   dns.TypeCAA,
}

// root hints: a-m.root-servers.net (2026)
var dnsRootServers = []string{
	"198.41.0.4", "170.247.170.2", "192.33.4.12", "199.7.91.13",
	"192.203.230.10", "192.5.5.241", "192.112.36.4", "198.97.190.53",
	"192.36.148.17", "192.58.128.30", "193.0.14.129", "199.7.83.42",
	"202.12.27.33",
}

type dnsAnswer struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   uint32 `json:"ttl"`
}

type dnsTraceStep struct {
	Step      int    `json:"step"`
	Kind      string `json:"kind"`
	Server    string `json:"server,omitempty"`
	Query     string `json:"query"`
	Rcode     string `json:"rcode"`
	Summary   string `json:"summary"`
	ElapsedMS int64  `json:"elapsed_ms"`
}

// normalizeDnsName trim and validate a user-supplied domain, return fqdn.
func normalizeDnsName(name string) (string, error) {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, ".")
	if name == "" || len(name) > 253 {
		return "", fmt.Errorf("bad domain name")
	}
	for _, r := range name {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' ||
			r == '.' || r == '-' || r == '_'
		if !ok {
			return "", fmt.Errorf("bad character in domain name")
		}
	}
	return dns.Fqdn(name), nil
}

// dnsTypeFromString lookup record type, default A.
func dnsTypeFromString(s string) (uint16, string, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" {
		s = "A"
	}
	qt, ok := dnsTypes[s]
	if !ok {
		return 0, "", fmt.Errorf("unsupported type: %s", s)
	}
	return qt, s, nil
}

// rrValue render the record's value by concrete type.
func rrValue(rr dns.RR) string {
	switch v := rr.(type) {
	case *dns.A:
		return v.A.String()
	case *dns.AAAA:
		return v.AAAA.String()
	case *dns.CNAME:
		return v.Target
	case *dns.NS:
		return v.Ns
	case *dns.MX:
		return fmt.Sprintf("%d %s", v.Preference, v.Mx)
	case *dns.TXT:
		return strings.Join(v.Txt, " ")
	case *dns.SOA:
		return fmt.Sprintf("%s %s %d %d %d %d %d", v.Ns, v.Mbox, v.Serial, v.Refresh, v.Retry, v.Expire, v.Minttl)
	case *dns.SRV:
		return fmt.Sprintf("%d %d %d %s", v.Priority, v.Weight, v.Port, v.Target)
	case *dns.PTR:
		return v.Ptr
	case *dns.CAA:
		return fmt.Sprintf("%d %s \"%s\"", v.Flag, v.Tag, v.Value)
	}
	return rr.String()
}

func answersFromMsg(m *dns.Msg) []dnsAnswer {
	out := []dnsAnswer{}
	for _, rr := range m.Answer {
		h := rr.Header()
		out = append(out, dnsAnswer{
			Name:  h.Name,
			Type:  dns.TypeToString[h.Rrtype],
			Value: rrValue(rr),
			TTL:   h.Ttl,
		})
	}
	return out
}

func rcodeString(m *dns.Msg) string {
	if m == nil {
		return "TIMEOUT"
	}
	if s, ok := dns.RcodeToString[m.Rcode]; ok {
		return s
	}
	return fmt.Sprintf("RCODE%d", m.Rcode)
}

// dnsExchange query server:53 for name/qtype, retry over TCP on truncation.
func dnsExchange(server, name string, qtype uint16, timeout time.Duration) (*dns.Msg, time.Duration, error) {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(name), qtype)
	m.RecursionDesired = true
	c := &dns.Client{Timeout: timeout, Net: "udp"}
	addr := net.JoinHostPort(server, dnsPort)
	in, elapsed, err := c.Exchange(m, addr)
	if err != nil {
		return nil, elapsed, err
	}
	if in.Truncated {
		c.Net = "tcp"
		in, elapsed, err = c.Exchange(m, addr)
	}
	return in, elapsed, err
}

// DnsQueryHandler resolve name via one resolver, return answers as json
// get only
func DnsQueryHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("request is:", r.Method, r.RequestURI)
	if r.Method != http.MethodGet {
		http.Error(w, "bad method for dns query", http.StatusMethodNotAllowed)
		return
	}
	name, err := normalizeDnsName(r.URL.Query().Get("name"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	qt, typeName, err := dnsTypeFromString(r.URL.Query().Get("type"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resolver := strings.TrimSpace(r.URL.Query().Get("resolver"))
	if resolver == "" {
		resolver = dnsDefaultResolver
	}
	if net.ParseIP(resolver) == nil {
		http.Error(w, "bad resolver ip", http.StatusBadRequest)
		return
	}
	start := time.Now()
	m, _, err := dnsExchange(resolver, name, qt, dnsQueryTimeout)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		log.Println("dns query error:", name, resolver, err)
		writeJson(w, map[string]interface{}{
			"name": name, "type": typeName, "resolver": resolver,
			"rcode": "ERROR", "answers": []dnsAnswer{},
			"elapsed_ms": elapsed, "error": err.Error(),
		})
		return
	}
	writeJson(w, map[string]interface{}{
		"name": name, "type": typeName, "resolver": resolver,
		"rcode": rcodeString(m), "answers": answersFromMsg(m),
		"elapsed_ms": elapsed,
	})
}

// traceReferral extract delegation nameservers and glue ips from a referral msg.
// returns (zone, nsNames, servers); each server prefers A (v4) over AAAA so
// v4-only boxes still work when both exist.
func traceReferral(m *dns.Msg) (string, []string, []string) {
	zone := ""
	var nsNames []string
	for _, rr := range m.Ns {
		if ns, ok := rr.(*dns.NS); ok {
			zone = ns.Header().Name
			nsNames = append(nsNames, ns.Ns)
		}
	}
	v4 := map[string]string{}
	v6 := map[string]string{}
	for _, rr := range m.Extra {
		switch g := rr.(type) {
		case *dns.A:
			v4[strings.ToLower(g.Header().Name)] = g.A.String()
		case *dns.AAAA:
			v6[strings.ToLower(g.Header().Name)] = g.AAAA.String()
		}
	}
	var servers []string
	seen := map[string]bool{}
	for _, n := range nsNames {
		key := strings.ToLower(n)
		ip, ok := v4[key]
		if !ok {
			ip, ok = v6[key]
		}
		if ok && !seen[ip] {
			seen[ip] = true
			servers = append(servers, ip)
		}
	}
	return zone, nsNames, servers
}

// resolveNsName resolve a nameserver hostname via the trace resolver
// (fallback when the referral carries no glue record).
func resolveNsName(nsHost string) []string {
	m, _, err := dnsExchange(dnsTraceResolver, nsHost, dns.TypeA, dnsTraceQueryTimeout)
	if err != nil || m == nil {
		return nil
	}
	var out []string
	for _, rr := range m.Answer {
		if a, ok := rr.(*dns.A); ok {
			out = append(out, a.A.String())
		}
	}
	return out
}

// DnsTraceHandler iterative resolution from the root servers, dig +trace style
// get only
func DnsTraceHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("request is:", r.Method, r.RequestURI)
	if r.Method != http.MethodGet {
		http.Error(w, "bad method for dns trace", http.StatusMethodNotAllowed)
		return
	}
	name, err := normalizeDnsName(r.URL.Query().Get("name"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	qt, typeName, err := dnsTypeFromString(r.URL.Query().Get("type"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	start := time.Now()
	deadline := start.Add(dnsTraceTotalTimeout)
	steps := []dnsTraceStep{}
	servers := dnsRootServers
	var finalAnswers []dnsAnswer
	finalRcode := ""

	for step := 1; step <= dnsMaxTraceSteps; step++ {
		if time.Now().After(deadline) {
			steps = append(steps, dnsTraceStep{Step: len(steps) + 1, Kind: "error",
				Query: name, Rcode: "TIMEOUT", Summary: "trace 总超时"})
			break
		}
		// ask servers of this level in order until one answers usefully
		var m *dns.Msg
		var elapsed time.Duration
		var lastErr string
		for _, srv := range servers {
			m, elapsed, err = dnsExchange(srv, name, qt, dnsTraceQueryTimeout)
			if err != nil {
				lastErr = err.Error()
				steps = append(steps, dnsTraceStep{Step: len(steps) + 1, Kind: "error", Server: srv,
					Query: name, Rcode: "ERROR", Summary: lastErr,
					ElapsedMS: elapsed.Milliseconds()})
				continue
			}
			if m.Rcode == dns.RcodeServerFailure || m.Rcode == dns.RcodeRefused {
				lastErr = rcodeString(m)
				steps = append(steps, dnsTraceStep{Step: len(steps) + 1, Kind: "error", Server: srv,
					Query: name, Rcode: rcodeString(m), Summary: "服务器拒绝或失败，换下一台",
					ElapsedMS: elapsed.Milliseconds()})
				continue
			}
			m.RecursionDesired = false
			break
		}
		if m == nil {
			steps = append(steps, dnsTraceStep{Step: len(steps) + 1, Kind: "error", Query: name,
				Rcode: "ERROR", Summary: "本级别所有服务器都失败: " + lastErr})
			break
		}

		// final answer (positive or NXDOMAIN) ends the trace
		if len(m.Answer) > 0 {
			finalAnswers = answersFromMsg(m)
			finalRcode = rcodeString(m)
			steps = append(steps, dnsTraceStep{Step: len(steps) + 1, Kind: "answer",
				Server: servers[0], Query: name, Rcode: finalRcode,
				Summary:   fmt.Sprintf("获得 %d 条答案", len(finalAnswers)),
				ElapsedMS: elapsed.Milliseconds()})
			break
		}
		if m.Rcode == dns.RcodeNameError {
			finalRcode = rcodeString(m)
			steps = append(steps, dnsTraceStep{Step: len(steps) + 1, Kind: "answer",
				Server: servers[0], Query: name, Rcode: finalRcode,
				Summary:   "域名不存在 (NXDOMAIN)",
				ElapsedMS: elapsed.Milliseconds()})
			break
		}

		// referral: follow the delegation chain
		zone, nsNames, next := traceReferral(m)
		if zone == "" || len(nsNames) == 0 {
			finalRcode = rcodeString(m)
			steps = append(steps, dnsTraceStep{Step: len(steps) + 1, Kind: "error",
				Server: servers[0], Query: name, Rcode: finalRcode,
				Summary: "没有答案也没有引用，追踪停止", ElapsedMS: elapsed.Milliseconds()})
			break
		}
		summary := fmt.Sprintf("引用 → %s NS: %s", zone, strings.Join(nsNames, ", "))
		if len(next) == 0 {
			summary += "（无 glue）"
		}
		steps = append(steps, dnsTraceStep{Step: len(steps) + 1, Kind: "referral",
			Server: servers[0], Query: name, Rcode: rcodeString(m),
			Summary: summary, ElapsedMS: elapsed.Milliseconds()})

		// fill in missing addresses by resolving NS hostnames recursively
		if len(next) == 0 {
			for _, n := range nsNames {
				if time.Now().After(deadline) {
					break
				}
				ips := resolveNsName(n)
				if len(ips) == 0 {
					steps = append(steps, dnsTraceStep{Step: len(steps) + 1, Kind: "resolve-ns",
						Server: dnsTraceResolver, Query: n, Rcode: "ERROR",
						Summary: "无法解析 NS 主机名"})
					continue
				}
				steps = append(steps, dnsTraceStep{Step: len(steps) + 1, Kind: "resolve-ns",
					Server: dnsTraceResolver, Query: n, Rcode: "NOERROR",
					Summary: "解析 NS 主机名 → " + strings.Join(ips, ", ")})
				next = append(next, ips...)
			}
		}
		if len(next) == 0 {
			steps = append(steps, dnsTraceStep{Step: len(steps) + 1, Kind: "error",
				Query: name, Rcode: "ERROR", Summary: "下一级没有可用服务器，追踪停止"})
			break
		}
		servers = next
	}

	if finalRcode == "" {
		finalRcode = "INCOMPLETE"
	}
	writeJson(w, map[string]interface{}{
		"name": name, "type": typeName, "rcode": finalRcode,
		"steps": steps, "answers": finalAnswers,
		"elapsed_ms": time.Since(start).Milliseconds(),
	})
}
