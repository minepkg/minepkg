package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Use:     "remove <package>",
	Short:   "Removes supplied package from the current directory & package",
	Aliases: []string{"delete", "un", "uninstall", "rm"},
	RunE: func(cmd *cobra.Command, args []string) error {
		instance, err := root.LocalInstance()
		if err != nil {
			return err
		}

		// we validate the local manifest
		if err := root.validateManifest(instance.Manifest); err != nil {
			return err
		}

		if instance.Manifest == nil {
			return fmt.Errorf("no manifest found in current directory")
		}

		if len(args) == 0 {
			return fmt.Errorf("no package name supplied")
		}

		fmt.Printf("Removing %s\n", args[0])
		instance.Manifest.RemoveDependency(args[0])
		instance.Manifest.RemoveDevDependency(args[0])
		instance.UpdateLockfileDependencies(context.TODO())
		instance.SaveManifest()
		instance.SaveLockfile()

		return nil
	},
}
