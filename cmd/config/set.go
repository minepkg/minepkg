package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jwalton/gchalk"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmd := commands.New(&cobra.Command{
		Use:   "set <value>",
		Short: "Sets a global config value",
		Args:  cobra.ExactArgs(2),
	}, &setRunner{})

	SubCmd.AddCommand(cmd.Command)
}

type setRunner struct{}

func (i *setRunner) RunE(cmd *cobra.Command, args []string) error {
	key := strings.ToLower(args[0])
	value := args[1]

	var newValue interface{}
	entry, ok := config[key]
	if !ok {
		return fmt.Errorf("config key \"%s\" does not exist", key)
	}

	switch entry.kind {
	case configKindBool:
		val, err := parseBool(value)
		if err != nil {
			return err
		}
		newValue = val
	case configKindString:
		newValue = value
	case configKindInt:
		num, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		newValue = num
	default:
		return fmt.Errorf("what? uncovered config values type")
	}
	previousValue := viper.Get(entry.key)
	previousStringValue := fmt.Sprintf("%v", previousValue)
	if previousValue == nil {
		previousStringValue = "(unset)"
	}
	viper.Set(entry.key, newValue)

	fmt.Printf(
		"Changing config entry:\n  %s: %s → %v\n",
		entry.key,
		gchalk.Strikethrough(previousStringValue),
		gchalk.Bold(fmt.Sprintf("%v", newValue)),
	)

	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	if err := viper.WriteConfigAs(filepath.Join(configDir, "minepkg/config.toml")); err == nil {
		return err
	}

	return nil
}

func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "yes", "ja", "on", "1":
		return true, nil
	case "false", "no", "nein", "off", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value. Use \"true\" or \"false\"")
	}
}
