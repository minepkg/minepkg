package cmd

import (
	"bufio"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/briandowns/spinner"

	"github.com/BurntSushi/toml"
	"github.com/fiws/minepkg/pkg/manifest"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Runs the build hook (or falls back to gradle build)",
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()

		minepkg, err := ioutil.ReadFile("./minepkg.toml")
		if err != nil {
			logger.Fail("Could not find a minepkg.toml in this directory")
		}

		m := manifest.Manifest{}
		_, err = toml.Decode(string(minepkg), &m)
		if err != nil {
			logger.Fail(err.Error())
		}

		if m.Package.Type != manifest.TypeMod {
			logger.Fail("Only mods can be build (for now)")
		}

		buildScript := "gradle --build-cache build"
		if m.Hooks.Build != "" {
			logger.Log("Using custom build hook")
			logger.Log("» " + m.Hooks.Build)
			buildScript = m.Hooks.Build
		} else {
			logger.Log("Using default build step (gradle --build-cache build)")
		}

		build := exec.Command("sh", []string{"-c", buildScript}...)
		build.Env = os.Environ()
		// TODO: test this … weird thing
		if runtime.GOOS == "windows" {
			build = exec.Command("cmd", []string{"/C", buildScript}...)
		}
		stdout, _ := build.StdoutPipe()
		err = build.Start()
		if err != nil {
			logger.Fail("Build step failed. Aborting")
		}

		logger.Info("Starting build")
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond) // Build our new spinner
		s.Prefix = " "
		s.Start()
		s.Suffix = " [no build output yet]"

		scanner := bufio.NewScanner(stdout)

		for scanner.Scan() {
			s.Suffix = " " + scanner.Text()
		}
		stdout.Close()
		err = build.Wait()
		if err != nil {
			logger.Fail("Build step failed. Aborting")
		}
		s.Suffix = ""
		s.Stop()

		logger.Info("Finished build in " + time.Now().Sub(startTime).String())
	},
}
