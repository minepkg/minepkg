package main

import (
	"fmt"

	"github.com/fiws/minepkg/cmd"
)

// set by goreleaser
var version string

func main() {
	fmt.Println(version)
	cmd.Version = version
	cmd.Execute()
}
