package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/fiws/minepkg/pkg/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(infoCmd)
	infoCmd.Flags().String("minecraft", "*", "Overwrite the required Minecraft version")
	infoCmd.Flags().String("platform", "fabric", "Overwrite the wanted platform")
}

var infoCmd = &cobra.Command{
	Use:    "info [name/url/id]",
	Short:  "returns information on a single package",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		comp := strings.Split(args[0], "@")
		name := comp[0]
		version := "latest"
		reqsMinecraft, _ := cmd.Flags().GetString("minecraft")
		platform, _ := cmd.Flags().GetString("platform")
		if len(comp) == 2 {
			version = comp[1]
		}

		fmt.Println("Searching for:")
		fmt.Printf(
			"  provider: %s\n  name: %s\n  version: %s\n  reqs.minecraft: %s\n",
			"minepkg",
			name,
			version,
			reqsMinecraft,
		)

		r, err := apiClient.FindRelease(context.TODO(), name, &api.RequirementQuery{
			Minecraft: reqsMinecraft,
			Plattform: platform,
			Version:   version,
		})

		if err != nil {
			logger.Fail(err.Error())
		}

		fmt.Println(r)

		fmt.Println("tested working with:")
		for _, test := range r.Tests {
			if test.Works {
				fmt.Printf(" %s ", test.Minecraft)
			}
		}
		fmt.Println()
	},
}
