package cmd

import (
	"fmt"
	"time"

	"github.com/fiws/minepkg/internals/instances"
	"github.com/fiws/minepkg/pkg/manifest"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(buildCmd)
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Runs the set buildCmd (falls back to gradle build)",
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()

		instance, err := instances.NewInstanceFromWd()
		if err != nil {
			logger.Fail(err.Error())
		}

		m := instance.Manifest

		if m.Package.Type != manifest.TypeMod {
			logger.Fail("Only mods can be build (for now)")
		}

		buildCmd := m.Dev.BuildCommand
		if buildCmd != "" {
			logger.Log("Using custom build hook")
			logger.Log("» " + buildCmd)
		} else {
			logger.Log("Using default build step (gradle --build-cache build)")
		}

		build := instance.BuildMod()
		fmt.Println("build started")
		cmdTerminalOutput(build)
		build.Start()

		err = build.Wait()
		if err != nil {
			// TODO: output logs or something
			fmt.Println(err)
			logger.Fail("Build step failed. Aborting")
		}

		logger.Info("Finished build in " + time.Now().Sub(startTime).String())
	},
}
