package cmd

import (
	"fmt"

	"github.com/minepkg/minepkg/internals/instances"
	"github.com/minepkg/minepkg/internals/launcher"
	"github.com/spf13/viper"
)

// installManifest installs dependencies from the minepkg.toml
func installManifest(instance *instances.Instance) error {
	cliLauncher := launcher.Launcher{
		Instance:       instance,
		MinepkgVersion: rootCmd.Version,
		NonInteractive: viper.GetBool("nonInteractive"),
		UseSystemJava:  viper.GetBool("useSystemJava"),
	}

	if err := cliLauncher.Prepare(); err != nil {
		return err
	}

	fmt.Println("You can now launch Minecraft using \"minepkg launch\"")
	return nil
}
