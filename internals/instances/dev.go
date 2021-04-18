package instances

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/minepkg/minepkg/internals/commands"
)

var (
	ErrNoBuildFiles = &commands.CliError{
		Text: "no jar files found",
		Suggestions: []string{
			`Set the "dev.jar" field in your minepkg.toml`,
			`Checkout https://preview.minepkg.io/docs/manifest#devjar`,
			"Make sure that your build is outputing jar files",
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

	// TODO: I don't think this is multi platform
	build := exec.Command("sh", []string{"-c", buildScript}...)
	build.Env = os.Environ()

	if runtime.GOOS == "windows" {
		// hack windows compatibility – space after gradlew ensures that this does not have .bat there anyway
		if strings.Contains(buildScript, "gradlew ") {
			buildScript = strings.Replace(buildScript, "gradlew ", "gradlew.bat ", 1)
		}
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
		return files[a].stat.ModTime().After(files[b].stat.ModTime())
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

	files := make([]MatchedJar, 0, len(matches))
	for _, file := range matches {
		stat, err := os.Stat(file)
		if err != nil {
			return nil, err
		}

		// filter out directories
		if !stat.IsDir() {
			matched := MatchedJar{
				path: file,
				stat: stat,
			}
			files = append(files, matched)
		}
	}

	return files, nil
}

func (i *Instance) findModJarCandidates() ([]MatchedJar, error) {
	files, err := ioutil.ReadDir("./build/libs")
	if err != nil {
		return nil, ErrNoBuildFiles
	}
	if len(files) == 0 {
		return nil, ErrNoBuildFiles
	}

	preFiltered := []MatchedJar{}

	for _, file := range files {
		name := file.Name()
		base := filepath.Base(name)

		// filter out dirs, dev and sources jars
		switch {
		case file.IsDir():
			continue
		case strings.HasSuffix(base, "dev.jar"):
			continue
		case strings.HasSuffix(base, "sources.jar"):
			continue
		default:
			matched := MatchedJar{
				path: filepath.Join("./build/libs", name),
				stat: file,
			}
			preFiltered = append(preFiltered, matched)
		}
	}

	return preFiltered, nil
}
