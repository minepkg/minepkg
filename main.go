package main

import (
	"github.com/fiws/minepkg/cmd"
)

// set by goreleaser
var version string

func main() {
	cmd.Version = version
	cmd.Execute()
}
