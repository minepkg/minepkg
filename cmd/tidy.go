package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// outdatedCmd represents the outdated command
var tidyCmd = &cobra.Command{
	Use:    "tidy",
	Short:  "Cleans up your minepkg.toml (currently only converts modrinth urls)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		instance, err := root.LocalInstance()
		if err != nil {
			return err
		}

		dependencies := instance.GetDependencyList()
		for _, dependency := range dependencies {
			if dependency.ID.Provider == "https" {
				fmt.Println("trying to convert", dependency.ID)
				new, err := root.ProviderStore.ConvertURL(context.Background(), dependency.ID.Version)
				if err != nil {
					logger.Warn("failed to convert", dependency.ID.Name, err.Error())
				} else {
					fmt.Println("  ✅ converted to", new)
					instance.Manifest.Dependencies[dependency.ID.Name] = new
				}
			}
		}

		instance.SaveManifest()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(tidyCmd)
}
