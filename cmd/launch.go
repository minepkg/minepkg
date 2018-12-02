package cmd

import (
	"fmt"

	"github.com/fiws/minepkg/internals/instances"
	"github.com/spf13/cobra"
)

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
		fmt.Printf("Launching %s\n", instance.Desc())
		err = instance.Launch()
		if err != nil {
			logger.Fail(err.Error())
		}
	},
}
