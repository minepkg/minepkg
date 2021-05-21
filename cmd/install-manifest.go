package cmd

import (
	"fmt"

	"github.com/minepkg/minepkg/cmd/launch"
	"github.com/minepkg/minepkg/internals/instances"
	"github.com/spf13/viper"
)

// installManifest installs dependencies from the minepkg.toml
func installManifest(instance *instances.Instance) error {
	cliLauncher := launch.CLILauncher{
		Instance:       instance,
		MinepkgVersion: rootCmd.Version,
		NonInteractive: viper.GetBool("nonInteractive"),
	}

	if err := cliLauncher.Prepare(); err != nil {
		return err
	}

	fmt.Println("You can now launch Minecraft using \"minepkg launch\"")
	return nil
}
