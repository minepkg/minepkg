package dev

import (
	"fmt"
	"time"

	"github.com/fiws/minepkg/internals/commands"
	"github.com/fiws/minepkg/internals/instances"
	"github.com/fiws/minepkg/pkg/manifest"
	"github.com/spf13/cobra"
)

func init() {
	cmd := commands.New(
		&cobra.Command{
			Use:   "build",
			Short: "Runs the set buildCommand (falls back to gradle build)",
		},
		&buildRunner{},
	)

	SubCmd.AddCommand(cmd.Command)
}

type buildRunner struct{}

func (b *buildRunner) RunE(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	instance, err := instances.NewInstanceFromWd()
	if err != nil {
		return err
	}

	m := instance.Manifest

	if m.Package.Type != manifest.TypeMod {
		return fmt.Errorf("only mods can be build (for now)")
	}

	buildCmd := m.Dev.BuildCommand
	if buildCmd != "" {
		fmt.Println("Using custom build hook")
		fmt.Println("» " + buildCmd)
	} else {
		fmt.Println("Using default build step (gradle --build-cache build)")
	}

	build := instance.BuildMod()
	fmt.Println("build started")
	// cmdTerminalOutput(build)
	build.Start()

	err = build.Wait()
	if err != nil {
		// TODO: output logs or something
		return fmt.Errorf("build step failed")
	}

	fmt.Println("Finished build in " + time.Since(startTime).String())

	return nil
}
