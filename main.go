package main

import (
	"github.com/minepkg/minepkg/cmd"
)

// set by goreleaser
var (
	version string
	commit  string
	Commit  string
)

func main() {
	cmd.Version = version
	cmd.Commit = commit
	cmd.Execute()
}
