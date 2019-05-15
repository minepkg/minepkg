package instances

import (
	"encoding/json"
	"path/filepath"
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

// Libraries as a collection of minecraft libs
type Libraries []lib

// Required returns only the required library file (matching rules)
func (l Libraries) Required() Libraries {
	required := make(Libraries, 0)

OUTER:
	for _, lib := range l {

		for _, rule := range lib.Rules {
			// skip here rules do not apply
			if rule.Applies() != true {
				continue OUTER
			}
		}

		// copy natives. not sure if this implementation is complete
		if len(lib.Natives) != 0 {
			_, ok := lib.Natives[runtime.GOOS]
			// skip native not available for this platform
			if ok != true {
				continue
			}
		}

		// not skipped. append this
		required = append(required, lib)
	}

	return required
}

type lib struct {
	Name      string `json:"name"`
	Downloads struct {
		Artifact    artifact            `json:"artifact"`
		Classifiers map[string]artifact `json:"classifiers"`
	} `json:"downloads,omitempty"`
	URL     string            `json:"url"`
	Rules   []libRule         `json:"rules"`
	Natives map[string]string `json:"natives"`
}

func (l *lib) Filepath() string {
	libPath := l.Downloads.Artifact.Path
	if libPath == "" {
		grouped := strings.Split(l.Name, ":")
		basePath := filepath.Join(strings.Split(grouped[0], ".")...)
		name := grouped[1]
		version := grouped[2]

		libPath = filepath.Join(basePath, name, version, name+"-"+version+".jar")
	}
	return libPath
}

func (l *lib) DownloadURL() string {
	switch {
	case l.Downloads.Artifact.URL != "":
		return l.Downloads.Artifact.URL
	case l.URL != "":
		return l.URL + filepath.ToSlash(l.Filepath())
	default:
		return "https://libraries.minecraft.net/" + filepath.ToSlash(l.Filepath())
	}
}

type artifact struct {
	Path string      `json:"path"`
	Sha1 string      `json:"sha1"`
	Size json.Number `json:"size"`
	URL  string      `json:"url"`
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

type mcAssetsIndex struct {
	Objects map[string]McAssetObject
}

// McAssetObject is one minecraft asset
type McAssetObject struct {
	Hash string
	Size int
}

// UnixPath returns the path including the folder
// example: /fe/fe32f3b8…
func (a *McAssetObject) UnixPath() string {
	return a.Hash[:2] + "/" + a.Hash
}

// DownloadURL returns the download url for this asset
func (a *McAssetObject) DownloadURL() string {
	return "https://resources.download.minecraft.net/" + a.UnixPath()
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

type stringSlice []string

func (w *stringSlice) String() string {
	return strings.Join(*w, " ")
}

// UnmarshalJSON is needed because argument sometimes is a string
func (w *stringSlice) UnmarshalJSON(data []byte) (err error) {
	var arg []string

	if string(data[0]) == "[" {
		err := json.Unmarshal(data, &arg)
		if err != nil {
			return err
		}
		*w = arg
	}

	*w = []string{string(data)}
	return nil
}

type stringArgument struct{ argument }

// UnmarshalJSON is needed because argument sometimes is a string
func (w *stringArgument) UnmarshalJSON(data []byte) (err error) {
	var arg argument
	if string(data[0]) == "{" {
		err := json.Unmarshal(data, &arg)
		if err != nil {
			return err
		}
		w.argument = arg
		return nil
	}

	var str string
	err = json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	w.Value = []string{str}
	return nil
}
