package minecraft

import "runtime"

type OS struct {
	Name string `json:"name"`
	// Version of the os (can be a regex string)
	Version string `json:"version"`
	// Arch of the system
	Arch string `json:"arch"`
}

type Rule struct {
	Action   string          `json:"action"`
	OS       OS              `json:"os"`
	Features map[string]bool `json:"features"`
}

func (r Rule) Applies() bool {
	return r.appliesFor(runtime.GOOS, runtime.GOARCH)
}

func (r Rule) appliesFor(os string, arch string) bool {
	if os == "darwin" {
		os = "osx"
	}

	if arch == "amd64" {
		arch = "x86"
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
		// TODO: check version (regex)

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

		// disallow block does not match os (or is empty)
		return true
	}

	// unknown action
	return true
}
