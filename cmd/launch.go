package cmd

import (
	"fmt"

	"github.com/fiws/minepkg/internals/instances"
	"github.com/spf13/cobra"
)

var version string
var listVersions bool

var launchCmd = &cobra.Command{
	Use:     "launch",
	Short:   "Launch a minecraft instance",
	Long:    ``, // TODO
	Aliases: []string{"run", "start", "play"},
	Run: func(cmd *cobra.Command, args []string) {
		instance, err := instances.DetectInstance()
		if err != nil {
			logger.Fail("Instance problem: " + err.Error())
		}
		// list versions instead of launching
		if listVersions == true {
			logger.Headline("Available Versions:")
			for _, version := range instance.AvailableVersions() {
				logger.Log(" - " + version.String())
			}
			return
		}

		// launch instance
		fmt.Printf("Launching %s\n", instance.Desc())
		if loginData.Mojang == nil {
			logger.Info("You need to sign in with your mojang account to launch minecraft")
			login()
		}
		instance.MojangCredentials = loginData.Mojang
		err = instance.Launch()
		if err != nil {
			logger.Fail(err.Error())
		}
	},
}

func init() {
	// launchCmd.Flags().StringVarP(&version, "run-version", "r", "", "Version to start. Uses the latest compatible if not present")
	launchCmd.Flags().BoolVarP(&listVersions, "list-versions", "", false, "List available versions instead of starting minecraft")
}
