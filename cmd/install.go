package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fiws/minepkg/internals/instances"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:     "install [name/url/id]",
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

		// create mod dir if not already present
		if err := os.MkdirAll(instance.ModsDirectory, 0755); err != nil {
			panic(err)
		}

		// no args: installing minepkg.toml dependencies
		if len(args) == 0 {
			installManifest(instance)
			return
		}

		firstArg := args[0]
		if strings.HasPrefix(firstArg, "https://") {
			switch {
			// got a curseforge url
			case strings.HasPrefix(firstArg, "https://minepkg.io/projects/"):
				projectname := firstArg[28:] // url minus first bits (just the name)
				err = installFromMinepkg([]string{projectname}, instance)
				if err != nil {
					logger.Fail(err.Error())
				}
				return
			}
			logger.Fail("Sorry. Don't know what to do with that url")
		}

		// fallback to minepkg
		err = installFromMinepkg(args, instance)
		if err != nil {
			logger.Fail(err.Error())
		}
	},
}
