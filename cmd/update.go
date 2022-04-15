package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Updates all installed dependencies",
	Long: `
This updates the local mods according to the minepkg.toml. 
Edit the minepkg.toml to change the version requirements.
`,
	Hidden:  true,
	Aliases: []string{"upd"},
	Args:    cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("not implemented (previous implementation was worse than nothing)")
	},
}
