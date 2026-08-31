package main

import (
	"encoding/json"
	"net/http"
	"time"
)

var startTime = time.Now()

type healthzInfo struct {
	OK          bool   `json:"ok"`
	UptimeS     int64  `json:"uptime_s"`
	VCSRevision string `json:"vcs_revision"`
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

// HealthzHandler reports liveness for probes: ok + uptime + build revision.
func HealthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthzInfo{
		OK:          true,
		UptimeS:     int64(time.Since(startTime).Seconds()),
		VCSRevision: vcsRevision(),
	})
}
