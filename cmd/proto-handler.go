package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(protoHandlerCmd)
}

const prefixLength = len("minepkg://")

var protoHandlerCmd = &cobra.Command{
	Use:     "proto-handler",
	Aliases: []string{"signin"},
	Args:    cobra.ExactArgs(1),
	Hidden:  true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(args)

		realArg := args[0][prefixLength:]
		fmt.Println("real: " + realArg)
		parsed := strings.Split(realArg, ":")
		action := parsed[0]

		switch action {
		case "try":
			if len(parsed) != 2 {
				panic("Invalid try command")
			}
			rootCmd.SetArgs([]string{"try", parsed[1]})
			err := rootCmd.Execute()
			if err != nil {
				panic(err)
			}
		}

		fmt.Println()
	},
}
