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
	"noninteractive":      {configKindBool, ""},
	"usesystemjava":       {configKindBool, ""},
	"verboselogging":      {configKindBool, ""},
	"acceptminecrafteula": {configKindBool, ""},
	"init.defaultsource":  {configKindBool, ""},
}

var SubCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage global config options",
}
