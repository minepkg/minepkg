package config

import (
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
}

var config = map[string]configEntry{
	"nonInteractive":      {configKindBool, ""},
	"useSystemJava":       {configKindBool, ""},
	"verboseLogging":      {configKindBool, ""},
	"acceptMinecraftEula": {configKindBool, ""},
	"init.defaultSource":  {configKindBool, ""},
	"updateChannel":       {configKindString, ""},
}

var SubCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage global config options",
}
