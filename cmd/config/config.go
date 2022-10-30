package config

import (
	"strings"

	"github.com/spf13/cobra"
)

const (
	configKindString = iota
	configKindBool
	configKindInt
)

type configEntry struct {
	kind int
	help string
	key  string
}

var config = map[string]configEntry{
	"nonInteractive":      {configKindBool, "", ""},
	"useSystemJava":       {configKindBool, "", ""},
	"verboseLogging":      {configKindBool, "", ""},
	"acceptMinecraftEula": {configKindBool, "", ""},
	"init.defaultSource":  {configKindBool, "", ""},
	"updateChannel":       {configKindString, "", ""},
}

var SubCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage global config options",
}

func init() {
	// transform all config key to lower case
	for key, entry := range config {
		entry.key = key
		config[strings.ToLower(key)] = entry
		delete(config, key)
	}
}
