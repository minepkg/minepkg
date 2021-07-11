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

	cmd.Flags().BoolVarP(&runner.dev, "dev", "D", false, "Install as a dev dependency only.")
	cmd.Flags().BoolVar(&runner.dev, "save-dev", false, "Same as --dev (for you node devs)")

	rootCmd.AddCommand(cmd.Command)
}

type installRunner struct {
	dev bool

	instance *instances.Instance
}

func (i *installRunner) RunE(cmd *cobra.Command, args []string) error {
	instance, err := instances.NewFromWd()
	if err != nil {
		return err
	}
	instance.MinepkgAPI = globals.ApiClient
	i.instance = instance
	fmt.Printf("Installing to %s\n\n", instance.Desc())

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
			return i.installFromMinepkg([]string{projectname})
		}
		return fmt.Errorf("sorry. Don't know what to do with that url (yet)")
	}

	// fallback to minepkg
	return i.installFromMinepkg(args)
}
