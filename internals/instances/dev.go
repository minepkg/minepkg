package instances

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	ErrNoBuildFiles = errors.New("No build files found in ./build/libs")
)

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

	// TODO: test this … weird thing
	if runtime.GOOS == "windows" {
		build = exec.Command("cmd", []string{"/C", buildScript}...)
	}

	return build
}

// FindModJar tries to find the right built mod jar
func (i *Instance) FindModJar() (string, error) {
	files, err := ioutil.ReadDir("./build/libs")
	if err != nil {
		return "", ErrNoBuildFiles
	}
	if len(files) == 0 {
		return "", ErrNoBuildFiles
	}

	chosen := files[0]

search:
	for _, file := range files[1:] {
		name := file.Name()
		base := filepath.Base(name)

		// filter out dev and sources jars
		switch {
		case strings.HasSuffix(base, "dev.jar"):
			continue
		case strings.HasSuffix(base, "sources.jar"):
			continue
		// worldedit uses dist for the runnable jars. lets hope this
		// does not break any other mods
		case strings.HasSuffix(base, "dist.jar"):
			// we choose this file and stop
			chosen = file
			break search
		}
		if len(file.Name()) < len(chosen.Name()) {
			chosen = file
		}
	}

	return filepath.Join("./build/libs", chosen.Name()), nil
}
