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
		osName = "macos"
	}

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
			_, ok := lib.Natives[osName]
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

// Lib is one minecraft library
type Lib struct {
	Name      string `json:"name"`
	Downloads struct {
		Artifact    artifact            `json:"artifact"`
		Classifiers map[string]artifact `json:"classifiers"`
	} `json:"downloads,omitempty"`
	URL     string            `json:"url"`
	Rules   []libRule         `json:"rules"`
	Natives map[string]string `json:"natives"`
}

// Filepath returns the target filepath for this library
func (l *Lib) Filepath() string {

	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "macos"
	}

	if l.Natives[osName] != "" {
		nativeID, _ := l.Natives[osName]
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
		osName = "macos"
	}

	switch {
	case l.Natives[osName] != "":
		nativeID, _ := l.Natives[osName]
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
	Path string      `json:"path"`
	Sha1 string      `json:"sha1"`
	Size json.Number `json:"size"`
	URL  string      `json:"url"`
}
