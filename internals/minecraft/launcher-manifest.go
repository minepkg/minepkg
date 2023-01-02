package minecraft

import (
	"strings"
)

// SupportedLauncherVersion indicates the maximum Launch Manifest version that is supported
const SupportedLauncherVersion = 21

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
	Libraries   Libraries `json:"libraries"`
	JavaVersion struct {
		Component    string `json:"component"`    // "java-runtime-beta" currently not used
		MajorVersion int    `json:"majorVersion"` // number like 16 or 17
	} `json:"javaVersion"`
	Type       string `json:"type"`
	MainClass  string `json:"mainClass"`
	Jar        string `json:"jar"`
	Assets     string `json:"assets"`
	AssetIndex struct {
		ID        string `json:"id"`
		Sha1      string `json:"sha1"`
		Size      int    `json:"size"`
		TotalSize int    `json:"totalSize"`
		URL       string `json:"url"`
	} `json:"assetIndex"`
	InheritsFrom           string `json:"inheritsFrom"`
	ID                     string `json:"id"`
	MinimumLauncherVersion int    `json:"minimumLauncherVersion"`
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

	l.JavaVersion = merge.JavaVersion

	// hack
	l.Downloads = merge.Downloads
}

// JarName returns this manifests jar name (for example `1.12.0.jar`)
func (l *LaunchManifest) JarName() string {
	return l.MinecraftVersion() + ".jar"
}

// MinecraftVersion returns the minecraft version
func (l *LaunchManifest) MinecraftVersion() string {
	v := l.Jar
	if v == "" {
		v = l.InheritsFrom
	}
	if v == "" {
		v = l.ID
	}
	return v
}

// LaunchArgs returns the launch arguments defined in the manifest as a string
func (l *LaunchManifest) LaunchArgs() []string {
	// easy minecraft versions before 1.13
	if l.MinecraftArguments != "" {
		return strings.Split(l.MinecraftArguments, "")
	}

	// TODO: missing jvm
	args := make([]string, 0)
OUTER:
	for _, arg := range l.Arguments.Game {
		for _, rule := range arg.Rules {
			// skip here rules do not apply
			if !rule.Applies() {
				continue OUTER
			}
		}
		args = append(args, strings.Join(arg.Value, ""))
	}

	return args
}

type argument struct {
	// Value is the actual argument
	Value stringSlice `json:"value"`
	Rules []Rule      `json:"rules"`
}
