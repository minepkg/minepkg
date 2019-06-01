package minecraft

import (
	"runtime"
	"strings"
)

// LaunchManifest is a version.json manifest that is used to launch minecraft instances
type LaunchManifest struct {
	// MinecraftArguments are used before 1.13 (?)
	MinecraftArguments string `json:"minecraftArguments"`
	// Arguments is the new (complicated) system
	Arguments struct {
		Game []stringArgument `json:"game"`
		JVM  []stringArgument `json:"jvm"`
	} `json:"arguments"`
	Downloads struct {
		Client mcJarDownload `json:"client"`
		Server mcJarDownload `json:"server"`
	} `json:"downloads"`
	Libraries  Libraries `json:"libraries"`
	Type       string    `json:"type"`
	MainClass  string    `json:"mainClass"`
	Jar        string    `json:"jar"`
	Assets     string    `json:"assets"`
	AssetIndex struct {
		ID        string `json:"id"`
		Sha1      string `json:"sha1"`
		Size      int    `json:"size"`
		TotalSize int    `json:"totalSize"`
		URL       string `json:"url"`
	} `json:"assetIndex"`
	InheritsFrom string `json:"inheritsFrom"`
}

type libRule struct {
	Action string `json:"action"`
	OS     struct {
		Name string `json:"name"`
	} `json:"os"`
	Features map[string]bool `json:"features"`
}

func (r libRule) Applies() bool {
	// Features? Do not not know what to do with this. skip it
	if len(r.Features) != 0 {
		return false
	}
	// TODO: there are more rules (arch for example)
	switch {
	// allow block but does not match os
	case r.Action == "allow" && r.OS.Name != runtime.GOOS:
		return false
	// disallow block matches os
	case r.Action == "disallow" && r.OS.Name == runtime.GOOS:
		return false
	// must match otherwise
	default:
		return true
	}
}

type mcJarDownload struct {
	Sha1 string `json:"sha1"`
	Size int    `json:"size"`
	URL  string `json:"url"`
}

// MergeWith merges important properties with another manifest
// if they are not present in the current one
// it also merges libraries by appending them.
// This is a simple implementation. it does not merge everything and
// does not care for duplicates in `Libraries`
func (l *LaunchManifest) MergeWith(merge *LaunchManifest) {
	l.Libraries = append(l.Libraries, merge.Libraries...)

	if l.MainClass == "" {
		l.MainClass = merge.MainClass
	}
	if l.Assets == "" {
		l.Assets = merge.Assets
	}
	if l.AssetIndex.ID == "" {
		l.AssetIndex = merge.AssetIndex
	}

	if len(l.Arguments.Game) == 0 {
		l.Arguments = merge.Arguments
	}
}

// LaunchArgs returns the launch arguments defined in the manifest as a string
func (l *LaunchManifest) LaunchArgs() string {
	// easy minecraft versions before 1.13
	if l.MinecraftArguments != "" {
		return l.MinecraftArguments
	}

	// TODO: missing jvm
	args := make([]string, 0)
OUTER:
	for _, arg := range l.Arguments.Game {
		for _, rule := range arg.Rules {
			// skip here rules do not apply
			if rule.Applies() != true {
				continue OUTER
			}
		}
		args = append(args, strings.Join(arg.Value, ""))
	}

	return strings.Join(args, " ")
}

type argument struct {
	// Value is the actual argument
	Value stringSlice `json:"value"`
	Rules []libRule   `json:"rules"`
}
