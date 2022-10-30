package minecraft

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
)

// Libraries as a collection of minecraft libs
type Libraries []Lib

// Required returns only the required library file (matching rules)
func (l Libraries) Required() Libraries {
	required := make(Libraries, 0)

	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "osx"
	}

	for _, lib := range l {

		include := true
		for _, rule := range lib.Rules {
			include = rule.Applies()
		}
		// did some rules not apply? skip this library
		if !include {
			continue
		}

		// copy natives. not sure if this implementation is complete
		if len(lib.Natives) != 0 {
			_, ok := lib.Natives[osName]
			// skip native not available for this platform
			if !ok {
				continue
			}
		}

		// not skipped. append this library
		required = append(required, lib)
	}

	return required
}

// Lib is a minecraft library
type Lib struct {
	// Name can be used to identify the library, but is not required otherwise.
	Name      string `json:"name"`
	Downloads struct {
		Artifact artifact `json:"artifact"`
		// Classifiers is a list of additional artifacts.
		// It is used to download native libraries.
		// The `Natives` field is used to determine which classifier to use.
		// This field is no longer used after 1.19
		Classifiers map[string]artifact `json:"classifiers"`
	} `json:"downloads,omitempty"`
	URL string `json:"url"`
	// Rules is a list of rules that determine whether this library should be included.
	// If no rules are specified, the library is included by default.
	Rules []libRule `json:"rules"`
	// Natives is a map of OS names to native library names.
	// This field is no longer used after 1.19
	// Newer library versions extract the native library from a jar at runtime.
	Natives map[string]string `json:"natives"`
}

// Filepath returns the target filepath for this library
func (l *Lib) Filepath() string {

	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "osx"
	}

	if l.Natives[osName] != "" {
		nativeID := l.Natives[osName]
		native := l.Downloads.Classifiers[nativeID]
		return native.Path
	}

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

// DownloadURL returns the Download URL this library
func (l *Lib) DownloadURL() string {
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "osx"
	}

	switch {
	case l.Natives[osName] != "":
		nativeID := l.Natives[osName]
		return l.Downloads.Classifiers[nativeID].URL
	case l.Downloads.Artifact.URL != "":
		return l.Downloads.Artifact.URL
	case l.URL != "":
		return l.URL + filepath.ToSlash(l.Filepath())
	default:
		return "https://libraries.minecraft.net/" + filepath.ToSlash(l.Filepath())
	}
}

type artifact struct {
	// Path of the jar file relative to the libraries folder
	Path string `json:"path"`
	Sha1 string `json:"sha1"`
	// Size in bytes
	Size json.Number `json:"size"`
	// URL to download the jar file
	URL string `json:"url"`
}
