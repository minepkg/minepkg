package cmd

import (
	"context"
	"fmt"

	"github.com/fiws/minepkg/internals/globals"
	"github.com/fiws/minepkg/internals/instances"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Use:     "remove <package>",
	Short:   "Removes supplied package from the current directory & package",
	Aliases: []string{"delete", "un", "uninstall", "rm"},
	Run: func(cmd *cobra.Command, args []string) {
		instance, err := instances.NewInstanceFromWd()
		instance.MinepkgAPI = globals.ApiClient
		if err != nil {
			logger.Fail("Instance problem: " + err.Error())
		}

		if instance.Manifest == nil {
			logger.Fail("No minepkg.toml manifest in the current directory")
		}

		if len(args) == 0 {
			logger.Fail("You have to supply a package to remove.")
		}

		fmt.Printf("Removing %s\n", args[0])
		instance.Manifest.RemoveDependency(args[0])
		instance.UpdateLockfileDependencies(context.TODO())
		instance.SaveManifest()
		instance.SaveLockfile()
	},
}
