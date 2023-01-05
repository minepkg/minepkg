package minecraft

import (
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
)

// Library is a minecraft library
type Library struct {
	// Name can be used to identify the library, but is not required otherwise.
	Name      string `json:"name"`
	Downloads struct {
		Artifact Artifact `json:"artifact"`
		// Classifiers is a list of additional artifacts.
		// It is used to download native libraries.
		// The `Natives` field is used to determine which classifier to use.
		// This field is no longer used starting with 1.19
		Classifiers map[string]Artifact `json:"classifiers"`
	} `json:"downloads,omitempty"`
	URL string `json:"url"`
	// Rules is a list of rules that determine whether this library should be included.
	// If no rules are specified, the library is included by default.
	Rules []Rule `json:"rules"`
	// Natives is a map of OS names to native library names.
	// This field is no longer used starting with 1.19
	// Newer library versions extract the native library from a jar at runtime.
	Natives map[string]string `json:"natives"`
}

// Applies returns true if every [Rule] in this argument applies (to this OS).
func (l *Library) Applies() bool {
	for _, rule := range l.Rules {
		if !rule.Applies() {
			return false
		}
	}
	return true
}

// Filepath returns the target filepath for this library
func (l *Library) Filepath() string {

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
func (l *Library) DownloadURL() string {
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
		joined, err := url.JoinPath(l.URL, filepath.ToSlash(l.Filepath()))
		if err != nil {
			panic(err)
		}
		return joined
	default:
		return "https://libraries.minecraft.net/" + filepath.ToSlash(l.Filepath())
	}
}

// RequiredLibraries returns a slice of libraries that are required for the current platform
func RequiredLibraries(libraries []Library) []Library {
	required := make([]Library, 0)

	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "osx"
	}

	for _, lib := range libraries {

		// include if all rules apply
		include := lib.Applies()

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
