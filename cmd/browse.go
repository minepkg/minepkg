package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(browseCmd)
}

var browseCmd = &cobra.Command{
	Use:   "browse",
	Short: "Opens a file browser in the minepkg instances directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		userConfig, err := os.UserConfigDir()
		if err != nil {
			panic(err)
		}

		instancesDir := filepath.Join(userConfig, "minepkg", "instances")

		fmt.Printf("Opening a file browser in:\n  %s\n", instancesDir)

		switch os := runtime.GOOS; os {
		case "darwin":
			err = exec.Command("open", instancesDir).Run()
		case "linux":
			err = exec.Command("xdg-open", instancesDir).Run()
		case "windows":
			err = exec.Command("explorer", instancesDir).Run()
		default:
			err = fmt.Errorf("unsupported platform %s", os)
		}
		return err
	},
}
