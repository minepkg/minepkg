package minecraft

import "runtime"

// Rule is a rule that can be applied to an argument or library.
// It can be used to determine if the argument or library should be applied to a specific OS.
type Rule struct {
	Action   string          `json:"action"`
	OS       OS              `json:"os"`
	Features map[string]bool `json:"features"`
}

// OS defines the feature of an OS that can be used in a [Rule] to determine if it should be applied.
type OS struct {
	Name string `json:"name"`
	// Version of the os (can be a regex string)
	Version string `json:"version"`
	// Arch of the system
	Arch string `json:"arch"`
}

func (r Rule) Applies() bool {
	return r.appliesFor(runtime.GOOS, runtime.GOARCH)
}

func (r Rule) appliesFor(os string, arch string) bool {
	if os == "darwin" {
		os = "osx"
	}

	if arch == "amd64" || arch == "x86_64" {
		arch = "x64"
	}

	if arch == "386" || arch == "i386" {
		arch = "x86"
	}

	if arch == "arm" {
		arch = "arm32"
	}

	// note: we don't know how other platforms are named

	// Features? Do not not know what to do with this. skip it
	if len(r.Features) != 0 {
		return false
	}

	if r.Action == "allow" {
		// check name
		if r.OS.Name != "" && r.OS.Name != os {
			return false
		}

		// TODO: check version (regex), we deny it for now
		if r.OS.Version != "" {
			return false
		}

		// check arch
		if r.OS.Arch != "" && r.OS.Arch != arch {
			return false
		}

		// allow block matches os (or is empty)
		return true
	}
	if r.Action == "disallow" {
		// check name
		if r.OS.Name != "" && r.OS.Name == os {
			return false
		}

		// check arch
		if r.OS.Arch != "" && r.OS.Arch == arch {
			return false
		}

		// TODO: check version (regex), we deny it for now
		// but only if the os matches
		if r.OS.Name == os && r.OS.Version != "" {
			return false
		}

		// disallow block does not match os (or is empty)
		return true
	}

	// unknown action
	return true
}
