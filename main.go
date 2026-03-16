package main

import "github.com/ChesterHsieh/skill-arena/cmd"

// Injected at build time via ldflags:
//   -X main.version=v1.0.0
//   -X main.commit=abc1234
//   -X main.date=2026-03-16
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.Execute(version, commit, date)
}
