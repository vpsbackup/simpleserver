package main

import (
	"strings"
	"testing"
	"time"
)

const testVnstatJson = `{
  "jsonversion": "2",
  "vnstatversion": "2.9",
  "interfaces": [
    {
      "name": "eth0",
      "alias": "wan",
      "created": {"date": {"year": 2025, "month": 1, "day": 1}},
      "updated": {"date": {"year": 2026, "month": 9, "day": 6}, "time": {"hour": 12, "minute": 30}},
      "traffic": {
        "fiveminute": [],
        "hour": [],
        "day": [
          {"id": 0, "date": {"year": 2026, "month": 8, "day": 31}, "rx": 1000, "tx": 2000},
          {"id": 1, "date": {"year": 2026, "month": 9, "day": 1}, "rx": 3000, "tx": 4000},
          {"id": 0, "date": {"year": 2026, "month": 9, "day": 6}, "rx": 500, "tx": 600}
        ],
        "month": [
          {"id": 0, "date": {"year": 2026, "month": 8, "day": 1}, "rx": 100000, "tx": 200000},
          {"id": 1, "date": {"year": 2026, "month": 9, "day": 1}, "rx": 300000, "tx": 400000}
        ],
        "top": [],
        "year": [],
        "total": {"rx": 123456789, "tx": 987654321}
      }
    }
  ]
}`

func TestParseVnstatJson(t *testing.T) {
	vs, err := ParseVnstatJson([]byte(testVnstatJson))
	if err != nil {
		t.Fatal(err)
	}
	if vs.VnstatVersion != "2.9" {
		t.Fatalf("bad vnstat version: %s", vs.VnstatVersion)
	}
	if len(vs.Interfaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(vs.Interfaces))
	}
	it := vs.Interfaces[0]
	if it.Name != "eth0" || it.Alias != "wan" {
		t.Fatalf("bad interface: %s %s", it.Name, it.Alias)
	}
	if it.Traffic.Total.Rx != 123456789 || it.Traffic.Total.Tx != 987654321 {
		t.Fatalf("bad total: %+v", it.Traffic.Total)
	}
	if len(it.Traffic.Day) != 3 || len(it.Traffic.Month) != 2 {
		t.Fatalf("bad day/month len: %d %d", len(it.Traffic.Day), len(it.Traffic.Month))
	}
	if _, err := ParseVnstatJson([]byte("not json")); err == nil {
		t.Fatal("expected parse error")
	}
	if _, err := ParseVnstatJson([]byte(`{"jsonversion":"2","interfaces":[]}`)); err == nil {
		t.Fatal("expected empty interfaces error")
	}
}

func TestVnstatSummary(t *testing.T) {
	vs, err := ParseVnstatJson([]byte(testVnstatJson))
	if err != nil {
		t.Fatal(err)
	}
	info := vnstatSummary(vs)
	if !info.Available {
		t.Fatal("expected available: true")
	}
	if info.VnstatVersion != "2.9" {
		t.Fatalf("bad version: %s", info.VnstatVersion)
	}
	if len(info.Interfaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(info.Interfaces))
	}
	it := info.Interfaces[0]
	if it.Name != "eth0" || it.Alias != "wan" {
		t.Fatalf("bad interface: %s %s", it.Name, it.Alias)
	}
	// today entry must match local today (2026-09-06 in fixture)
	if today := time.Now().Format("2006-01-02"); strings.HasPrefix(today, "2026-09-06") {
		if it.TodayRx != 500 || it.TodayTx != 600 {
			t.Fatalf("bad today: %d %d", it.TodayRx, it.TodayTx)
		}
	}
	// month entry must match current month only when the clock is 2026-09
	if month := time.Now().Format("2006-01"); month == "2026-09" {
		if it.MonthRx != 300000 || it.MonthTx != 400000 {
			t.Fatalf("bad month: %d %d", it.MonthRx, it.MonthTx)
		}
	}
	if it.TotalRx != 123456789 || it.TotalTx != 987654321 {
		t.Fatalf("bad total: %d %d", it.TotalRx, it.TotalTx)
	}
	if len(it.Days) != 3 {
		t.Fatalf("expected 3 days, got %d", len(it.Days))
	}
	if it.Days[2].Date != "2026-09-06" {
		t.Fatalf("bad last day: %s", it.Days[2].Date)
	}
}

func TestVnstatSummaryDayTrim(t *testing.T) {
	vs := &Vnstat{
		VnstatVersion: "2.9",
		Interfaces: []VnstatInterface{
			{Name: "eth0"},
		},
	}
	// 10 day entries, keep only last 7
	for i := 1; i <= 10; i++ {
		vs.Interfaces[0].Traffic.Day = append(vs.Interfaces[0].Traffic.Day, VnstatInterfaceTrafficDay{
			Date: VnstatDate{Year: 2026, Month: 9, Day: i},
			Rx:   int64(i) * 100,
			Tx:   int64(i) * 200,
		})
	}
	info := vnstatSummary(vs)
	if len(info.Interfaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(info.Interfaces))
	}
	days := info.Interfaces[0].Days
	if len(days) != 7 {
		t.Fatalf("expected 7 days, got %d", len(days))
	}
	if days[0].Date != "2026-09-04" || days[6].Date != "2026-09-10" {
		t.Fatalf("bad trim range: %s .. %s", days[0].Date, days[6].Date)
	}
}
