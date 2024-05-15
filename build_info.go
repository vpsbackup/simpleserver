package main

import (
	"fmt"
	"runtime/debug"
)

func Version() {
	bi, _ := debug.ReadBuildInfo()
	// log.Println("ok:", ok)
	fmt.Println("go version:", bi.GoVersion)
	// log.Println("path:", bi.Path)
	// log.Println("main:", bi.Main)
	// log.Println("deps:", bi.Deps)
	// log.Println("settings:")
	for _, s := range bi.Settings {
		if s.Key == "vcs.revision" || s.Key == "vcs.time" {
			fmt.Println(s.Key, s.Value)
		}
	}
}
