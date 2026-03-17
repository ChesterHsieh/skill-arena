package main

import (
	"runtime/debug"

	"github.com/ChesterHsieh/skill-arena/cmd"
)

// Injected at build time via ldflags (GoReleaser):
//   -X main.version=v1.0.0
//   -X main.commit=abc1234
//   -X main.date=2026-03-16
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// For `go install` builds, ldflags are not set — read the module version
	// from the binary's embedded build info instead.
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				version = info.Main.Version
			}
			for _, s := range info.Settings {
				if s.Key == "vcs.revision" && len(s.Value) >= 7 {
					commit = s.Value[:7]
				}
				if s.Key == "vcs.time" {
					date = s.Value
				}
			}
		}
	}
	cmd.Execute(version, commit, date)
}
