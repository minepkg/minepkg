package instances

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/minepkg/minepkg/internals/commands"
)

var (
	ErrNoBuildFiles = &commands.CliError{
		Text: "no jar files found",
		Suggestions: []string{
			`Set the "dev.jar" field in your minepkg.toml`,
			`Checkout https://preview.minepkg.io/docs/manifest#devjar`,
			"Make sure that your build is outputting jar files",
		},
	}
)

type MatchedJar struct {
	path string
	stat fs.FileInfo
}

// Name returns just the name of the jar
// eg "my-jar.jar"
func (m *MatchedJar) Name() string {
	return filepath.Base(m.path)
}

// Path returns the full path to the jar file
func (m *MatchedJar) Path() string {
	return m.path
}

// BuildMod uses the manifest "dev.buildCmd" script to build this package
// falls back to "gradle --build-cache build"
func (i *Instance) BuildMod() *exec.Cmd {
	buildScript := "gradle --build-cache build"
	buildCmd := i.Manifest.Dev.BuildCommand
	if buildCmd != "" {
		buildScript = buildCmd
	}

	build := exec.Command("sh", []string{"-c", buildScript}...)
	build.Env = os.Environ()

	if runtime.GOOS == "windows" {
		// hack windows compatibility – space after gradlew ensures that this does not have .bat there anyway
		buildScript = strings.Replace(buildScript, "gradlew ", "gradlew.bat ", 1)
		build = exec.Command("powershell", []string{"-Command", buildScript}...)
	}

	return build
}

// FindModJar tries to find the right built mod jar
func (i *Instance) FindModJar() ([]MatchedJar, error) {

	var files []MatchedJar
	var err error
	if i.Manifest.Dev.Jar != "" {
		files, err = i.findModJarCandidatesFromPattern(i.Manifest.Dev.Jar)

	} else {
		files, err = i.findModJarCandidates()
	}
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, ErrNoBuildFiles
	}

	sort.Slice(files, func(a int, b int) bool {
		fileA := files[a]
		fileB := files[b]
		timeDiff := math.Abs(float64(fileA.stat.ModTime().UnixNano() - fileB.stat.ModTime().UnixNano()))

		// 2 files written roughly at the same time. prefer the shorter one
		if timeDiff <= float64(time.Millisecond*10) {
			// prefer jars ending in -fabric.jar if platform is fabric
			if i.Platform() == PlatformFabric && strings.HasSuffix(fileA.Name(), "-fabric.jar") {
				return true
			}
			// prefer shortest jar otherwise
			return len(fileA.Name()) < len(fileB.Name())
		}

		// times differ, pick the newer one
		return fileA.stat.ModTime().After(fileB.stat.ModTime())
	})

	return files, nil
}

func (i *Instance) findModJarCandidatesFromPattern(pattern string) ([]MatchedJar, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	switch {
	case len(matches) == 0:
		return nil, ErrNoBuildFiles
	case len(matches) > 100:
		return nil, fmt.Errorf("aborting because of over 100 matched files")
	}

	jars := make([]MatchedJar, 0, len(matches))
	for _, match := range matches {
		fi, err := os.Stat(match)
		if err != nil {
			return nil, err
		}
		if jarCandidate(fi) {
			jars = append(jars, MatchedJar{
				path: match,
				stat: fi,
			})
		}
	}

	return jars, nil
}

func (i *Instance) findModJarCandidates() ([]MatchedJar, error) {
	files, err := ioutil.ReadDir("./build/libs")
	if err != nil {
		return nil, ErrNoBuildFiles
	}
	if len(files) == 0 {
		return nil, ErrNoBuildFiles
	}

	filtered := preFilteredFiles(files)
	jars := make([]MatchedJar, len(filtered))
	for ix, file := range filtered {
		jars[ix] = MatchedJar{
			path: filepath.Join("./build/libs", file.Name()),
			stat: file,
		}
	}

	return jars, nil
}

// preFilteredFiles filters out common dev files (dev, sources)
func preFilteredFiles(files []fs.FileInfo) []fs.FileInfo {
	filtered := []fs.FileInfo{}
	for _, file := range files {
		if jarCandidate(file) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func jarCandidate(file fs.FileInfo) bool {
	name := file.Name()
	base := filepath.Base(name)
	// filter out dirs, dev and sources jars
	switch {
	case file.IsDir():
		return false
	case strings.HasSuffix(base, "dev.jar"):
		return false
	case strings.HasSuffix(base, "sources.jar"):
		return false
	case strings.HasSuffix(base, "javadoc.jar"):
		return false
	case strings.HasSuffix(base, "-api.jar"):
		return false
	default:
		return true
	}
}
