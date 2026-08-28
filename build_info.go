package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
)

type versionModule struct {
	Path    string         `json:"path"`
	Version string         `json:"version"`
	Sum     string         `json:"sum,omitempty"`
	Replace *versionModule `json:"replace,omitempty"`
}

type versionSetting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type versionInfo struct {
	OK        bool             `json:"ok"`
	GoVersion string           `json:"go_version,omitempty"`
	Path      string           `json:"path,omitempty"`
	Main      *versionModule   `json:"main,omitempty"`
	Deps      []versionModule  `json:"deps,omitempty"`
	Settings  []versionSetting `json:"settings,omitempty"`
	Error     string           `json:"error,omitempty"`
}

func moduleFrom(m *debug.Module) *versionModule {
	if m == nil {
		return nil
	}
	out := &versionModule{
		Path:    m.Path,
		Version: m.Version,
		Sum:     m.Sum,
	}
	if m.Replace != nil {
		out.Replace = moduleFrom(m.Replace)
	}
	return out
}

func collectVersionInfo() versionInfo {
	bi, ok := debug.ReadBuildInfo()
	if !ok || bi == nil {
		return versionInfo{
			OK:    false,
			Error: "build info unavailable",
		}
	}
	info := versionInfo{
		OK:        true,
		GoVersion: bi.GoVersion,
		Path:      bi.Path,
		Main:      moduleFrom(&bi.Main),
	}
	if len(bi.Deps) > 0 {
		info.Deps = make([]versionModule, 0, len(bi.Deps))
		for _, d := range bi.Deps {
			if d == nil {
				continue
			}
			info.Deps = append(info.Deps, *moduleFrom(d))
		}
	}
	if len(bi.Settings) > 0 {
		info.Settings = make([]versionSetting, 0, len(bi.Settings))
		for _, s := range bi.Settings {
			info.Settings = append(info.Settings, versionSetting{Key: s.Key, Value: s.Value})
		}
	}
	return info
}

// PrintVersionJSON prints the full Go debug.BuildInfo as JSON.
func PrintVersionJSON() {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(collectVersionInfo()); err != nil {
		fmt.Fprintln(os.Stderr, "encode version json:", err)
		os.Exit(1)
	}
}

// Version prints a short human-readable summary for -h.
func Version() {
	info := collectVersionInfo()
	if !info.OK {
		fmt.Println("build info unavailable")
		return
	}
	fmt.Println("go version:", info.GoVersion)
	if info.Path != "" {
		fmt.Println("path:", info.Path)
	}
	if info.Main != nil {
		fmt.Println("main:", info.Main.Path, info.Main.Version)
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" || s.Key == "vcs.time" || s.Key == "vcs.modified" {
			fmt.Println(s.Key, s.Value)
		}
	}
}
