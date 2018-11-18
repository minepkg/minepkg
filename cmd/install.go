package cmd

import (
	"fmt"
	"strings"

	"github.com/fiws/minepkg/internals/instances"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:     "install [name/url/id ...]",
	Short:   "installz packages",
	Long:    `Just install them packages noaw`,
	Aliases: []string{"isntall", "i"},
	Run: func(cmd *cobra.Command, args []string) {
		instance, err := instances.DetectInstance()
		if err != nil {
			logger.Fail("Instance problem: " + err.Error())
		}
		fmt.Printf("Installing to %s\n", instance.Desc())
		if instance.Flavour == instances.FlavourMMC {
			logger.Warn("MultiMC support is not officialy endorsed.")
			logger.Log("Report bugs to http://github.com/fiws/minepkg/issues")
		}
		fmt.Println() // empty line

		// no args: installing minepkg.toml dependencies
		if len(args) == 0 {
			installManifest(instance)
			return
		}

		// looks like a source zip file. install from source
		if strings.HasPrefix(args[0], "https://") && strings.HasSuffix(args[0], ".zip") {
			installFromSource(args[0], instance)
			return
		}

		// fallback to curseforge
		installFromCurse(args, instance)
	},
}
