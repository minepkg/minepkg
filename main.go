package main

import (
	"net/http"

	"github.com/minepkg/minepkg/cmd"
	"github.com/minepkg/minepkg/internals/ownhttp"
)

// set by goreleaser
var (
	version string
	commit  string
	Commit  string
)

func main() {

	// replace default http client
	http.DefaultClient = ownhttp.New()

	cmd.Version = version
	cmd.Commit = commit
	cmd.Execute()
}
