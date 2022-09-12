package utils

import (
	"strings"

	"github.com/jwalton/gchalk"
)

// PrettyVersion returns a pretty colored version string for terminal printing
func PrettyVersion(version string) string {
	// we trim first to avoid broken colors
	if len(version) >= 22 {
		version = version[:18] + " …"
	}

	versionParts := strings.SplitN(version, "-", 2)
	prettyVersion := versionParts[0]

	if version == "none" {
		prettyVersion = gchalk.Gray("none (overwritten)")
	} else if len(versionParts) == 2 {
		prettyVersion += "-" + versionParts[1]
		// prettyVersion += gchalk.Dim("-" + versionParts[1])
	}

	return prettyVersion
}
