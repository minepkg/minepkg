package cmd

import (
	"fmt"
	"strings"

	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/internals/globals"
	"github.com/minepkg/minepkg/internals/instances"
	"github.com/spf13/cobra"
)

func init() {
	runner := &installRunner{}
	cmd := commands.New(&cobra.Command{
		Use:     "install [name/url/id]",
		Short:   "Installs one or more packages in your current modpack or mod",
		Long:    `Adds package(s) to your local modpack or mod. Launch the modpack with minepkg launch`,
		Aliases: []string{"isntall", "i", "add"},
	}, runner)

	rootCmd.AddCommand(cmd.Command)
}

type installRunner struct{}

func (i *installRunner) RunE(cmd *cobra.Command, args []string) error {
	instance, err := instances.NewInstanceFromWd()
	if err != nil {
		return err
	}
	instance.MinepkgAPI = globals.ApiClient
	fmt.Printf("Installing to %s\n", instance.Desc())
	fmt.Println() // empty line

	// no args: installing minepkg.toml dependencies
	if len(args) == 0 {
		return installManifest(instance)
	}

	firstArg := args[0]
	if strings.HasPrefix(firstArg, "https://") {
		switch {
		// got a minepkg url
		case strings.HasPrefix(firstArg, "https://minepkg.io/projects/"):
			projectname := firstArg[28:] // url minus first bits (just the name)
			return installFromMinepkg([]string{projectname}, instance)
		}
		return fmt.Errorf("sorry. Don't know what to do with that url (yet)")
	}

	// fallback to minepkg
	return installFromMinepkg(args, instance)
}
