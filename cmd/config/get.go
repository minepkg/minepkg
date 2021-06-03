package config

import (
	"fmt"
	"strings"

	"github.com/minepkg/minepkg/internals/commands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmd := commands.New(&cobra.Command{
		Use:   "get <value>",
		Short: "Gets a global config value",
		Args:  cobra.ExactArgs(1),
	}, &getRunner{})

	SubCmd.AddCommand(cmd.Command)
}

type getRunner struct{}

func (i *getRunner) RunE(cmd *cobra.Command, args []string) error {
	key := strings.ToLower(args[0])

	_, ok := config[key]
	if !ok {
		return fmt.Errorf("config key \"%s\" does not exist", key)
	}

	fmt.Println("Printing config entry:")
	fmt.Printf("  %s: %v\n", key, viper.Get(key))

	return nil
}
