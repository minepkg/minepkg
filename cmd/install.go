package cmd

import (
	"fmt"
	"strings"

	"github.com/minepkg/minepkg/internals/commands"
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
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			// do not complete if we have an argument
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return root.AutoCompleter.Complete(toComplete)
		},
	}, runner)

	cmd.Flags().BoolVarP(&runner.dev, "dev", "D", false, "Install as a dev dependency only.")
	cmd.Flags().BoolVar(&runner.dev, "save-dev", false, "Alias for --dev")

	rootCmd.AddCommand(cmd.Command)
}

type installRunner struct {
	dev bool

	instance *instances.Instance
}

func (i *installRunner) RunE(cmd *cobra.Command, args []string) error {
	instance, err := root.LocalInstance()
	if err != nil {
		return err
	}
	i.instance = instance
	// we validate the local manifest
	if err := root.validateManifest(instance.Manifest); err != nil {
		return err
	}
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
			projectName := firstArg[28:] // url minus first bits (just the name)
			return i.installFromMinepkg([]string{projectName})
		}
		return fmt.Errorf("sorry. Don't know what to do with that url (yet)")
	}

	// fallback to minepkg
	return i.installFromMinepkg(args)
}
