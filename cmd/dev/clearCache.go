package dev

import (
	"os"
	"path/filepath"

	"github.com/minepkg/minepkg/internals/commands"
	"github.com/spf13/cobra"
)

func init() {
	cmd := commands.New(&cobra.Command{
		Use:    "clear-cache",
		Short:  "Clears the minepkg cache",
		Hidden: false,
	}, &clearCacheRunner{})

	SubCmd.AddCommand(cmd.Command)
}

type clearCacheRunner struct{}

func (i *clearCacheRunner) RunE(cmd *cobra.Command, args []string) error {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	os.RemoveAll(filepath.Join(cacheDir, "minepkg"))
	os.Mkdir(filepath.Join(cacheDir, "minepkg"), 0755)

	return nil
}
