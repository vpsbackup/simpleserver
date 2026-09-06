package main

import (
	"testing"

	"github.com/miekg/dns"
)

func TestNormalizeDnsName(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"example.com", "example.com.", true},
		{"  example.com  ", "example.com.", true},
		{"example.com.", "example.com.", true},
		{"_dmarc.example.com", "_dmarc.example.com.", true},
		{"", "", false},
		{"bad char.com", "", false},
		{"bad@char.com", "", false},
		{strings_253_beyond(), "", false},
	}
	for _, c := range cases {
		got, err := normalizeDnsName(c.in)
		if c.ok && err != nil {
			t.Errorf("normalizeDnsName(%q) unexpected error: %v", c.in, err)
			continue
		}
		if !c.ok && err == nil {
			t.Errorf("normalizeDnsName(%q) expected error, got %q", c.in, got)
			continue
		}
		if c.ok && got != c.want {
			t.Errorf("normalizeDnsName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func strings_253_beyond() string {
	// 254 chars of 'a' separated by dots exceeds the 253 limit
	s := ""
	for len(s) < 254 {
		s += "a."
	}
	return s + "com"
}

func TestDnsTypeFromString(t *testing.T) {
	if qt, name, err := dnsTypeFromString("a"); err != nil || qt != dns.TypeA || name != "A" {
		t.Errorf("dnsTypeFromString(a) = %d %q %v", qt, name, err)
	}
	if _, name, err := dnsTypeFromString(""); err != nil || name != "A" {
		t.Errorf("empty should default to A, got %q %v", name, err)
	}
	if _, _, err := dnsTypeFromString("BOGUS"); err == nil {
		t.Error("BOGUS should be rejected")
	}
}

func TestRRValueCases(t *testing.T) {
	a := &dns.A{Hdr: dns.RR_Header{Name: "x.com.", Rrtype: dns.TypeA, Class: dns.ClassINET}}
	a.A = []byte{1, 2, 3, 4}
	if got := rrValue(a); got != "1.2.3.4" {
		t.Errorf("A value = %q", got)
	}
	cn := &dns.CNAME{Hdr: dns.RR_Header{Name: "x.com.", Rrtype: dns.TypeCNAME, Class: dns.ClassINET}, Target: "y.com."}
	if got := rrValue(cn); got != "y.com." {
		t.Errorf("CNAME value = %q", got)
	}
	mx := &dns.MX{Hdr: dns.RR_Header{Name: "x.com.", Rrtype: dns.TypeMX, Class: dns.ClassINET}, Preference: 10, Mx: "mail.y.com."}
	if got := rrValue(mx); got != "10 mail.y.com." {
		t.Errorf("MX value = %q", got)
	}
	txt := &dns.TXT{Hdr: dns.RR_Header{Name: "x.com.", Rrtype: dns.TypeTXT, Class: dns.ClassINET}, Txt: []string{"part1", "part2"}}
	if got := rrValue(txt); got != "part1 part2" {
		t.Errorf("TXT value = %q", got)
	}
	srv := &dns.SRV{Hdr: dns.RR_Header{Name: "_s._tcp.x.com.", Rrtype: dns.TypeSRV, Class: dns.ClassINET}, Priority: 1, Weight: 2, Port: 443, Target: "h.x.com."}
	if got := rrValue(srv); got != "1 2 443 h.x.com." {
		t.Errorf("SRV value = %q", got)
	}
}

func TestAnswersFromMsg(t *testing.T) {
	a := &dns.A{Hdr: dns.RR_Header{Name: "x.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300}}
	a.A = []byte{1, 2, 3, 4}
	m := new(dns.Msg)
	m.Answer = []dns.RR{a}
	answers := answersFromMsg(m)
	if len(answers) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(answers))
	}
	if answers[0].Type != "A" || answers[0].Value != "1.2.3.4" || answers[0].TTL != 300 {
		t.Errorf("bad answer: %+v", answers[0])
	}
	if empty := answersFromMsg(new(dns.Msg)); len(empty) != 0 {
		t.Errorf("expected empty slice, got %d", len(empty))
	}
}

func TestTraceReferral(t *testing.T) {
	m := new(dns.Msg)
	ns1 := &dns.NS{Hdr: dns.RR_Header{Name: "com.", Rrtype: dns.TypeNS, Class: dns.ClassINET}, Ns: "a.gtld-servers.net."}
	ns2 := &dns.NS{Hdr: dns.RR_Header{Name: "com.", Rrtype: dns.TypeNS, Class: dns.ClassINET}, Ns: "b.gtld-servers.net."}
	m.Ns = []dns.RR{ns1, ns2}
	// glue only for the first host
	glue := &dns.A{Hdr: dns.RR_Header{Name: "a.gtld-servers.net.", Rrtype: dns.TypeA, Class: dns.ClassINET}}
	glue.A = []byte{192, 5, 6, 30}
	m.Extra = []dns.RR{glue}

	zone, nsNames, servers := traceReferral(m)
	if zone != "com." {
		t.Errorf("zone = %q", zone)
	}
	if len(nsNames) != 2 {
		t.Fatalf("nsNames = %v", nsNames)
	}
	if len(servers) != 1 || servers[0] != "192.5.6.30" {
		t.Errorf("servers = %v, want only glue ip", servers)
	}

	// no glue and no answer: servers must be empty so caller resolves NS names
	m2 := new(dns.Msg)
	m2.Ns = []dns.RR{ns1}
	zone2, _, servers2 := traceReferral(m2)
	if zone2 != "com." || len(servers2) != 0 {
		t.Errorf("no-glue referral: zone=%q servers=%v", zone2, servers2)
	}
}
