package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCollectVersionInfoJSON(t *testing.T) {
	info := collectVersionInfo()
	if !info.OK {
		t.Fatalf("expected build info: %+v", info)
	}
	if info.GoVersion == "" {
		t.Fatal("missing go_version")
	}
	bt, err := json.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	s := string(bt)
	for _, key := range []string{`"ok"`, `"go_version"`, `"settings"`} {
		if !strings.Contains(s, key) {
			t.Fatalf("json missing %s: %s", key, s)
		}
	}
}
