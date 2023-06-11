package cmd

import (
	"fmt"

	"github.com/minepkg/minepkg/gui"
	"github.com/spf13/cobra"
)

// outdatedCmd represents the outdated command
var guiCmd = &cobra.Command{
	Use:    "gui",
	Short:  "Starts the GUI",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Starting GUI")
		gui.Start(root.MinepkgAPI, root.ProviderStore)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(guiCmd)
}
