package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/fiws/minepkg/pkg/manifest"
	"github.com/spf13/cobra"
)

func init() {
	buildCmd.Flags().BoolVarP(&nonInteractive, "non-interactive", "y", false, "Answers all interactive questions with the default")
	rootCmd.AddCommand(buildCmd)
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Runs the build hook (falls back to gradle build)",
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()

		if os.Getenv("CI") != "" {
			nonInteractive = true
		}

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
		buildCmd := m.Dev.BuildCommand
		if buildCmd != "" {
			logger.Log("Using custom build hook")
			logger.Log("» " + buildCmd)
			buildScript = buildCmd
		} else {
			logger.Log("Using default build step (gradle --build-cache build)")
		}

		build := exec.Command("sh", []string{"-c", buildScript}...)
		build.Env = os.Environ()
		// TODO: test this … weird thing
		if runtime.GOOS == "windows" {
			build = exec.Command("cmd", []string{"/C", buildScript}...)
		}

		if nonInteractive == true {
			terminalOutput(build)
		}

		var spinner func()
		if nonInteractive != true {
			spinner = spinnerOutput(build)
		}

		build.Start()
		if nonInteractive != true {
			spinner()
		}

		err = build.Wait()
		if err != nil {
			// TODO: output logs or something
			fmt.Println(err)
			logger.Fail("Build step failed. Aborting")
		}

		logger.Info("Finished build in " + time.Now().Sub(startTime).String())
	},
}
