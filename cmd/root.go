package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fiws/minepkg/cmd/dev"
	"github.com/fiws/minepkg/internals/cmdlog"
	"github.com/fiws/minepkg/internals/credentials"
	"github.com/fiws/minepkg/internals/globals"
	"github.com/gookit/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// TODO: this logger is not so great – also: it should not be global
var logger *cmdlog.Logger = cmdlog.New()

// Version is the current version. it should be set by goreleaser
var Version string

// nextVersion is a placeholder version. only used for local dev
var nextVersion string = "0.1.0-dev-local"

var (
	cfgFile       string
	globalDir     = "/tmp"
	disableColors bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	// Version gets set dynamically
	Use:   "minepkg",
	Short: "Minepkg at your service.",
	Long:  "Manage Minecraft mods with ease",

	Example: `
  minepkg init -l fabric
  minepkg install modmenu@latest
  minepkg join demo.minepkg.host`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	initRoot()

	if err := rootCmd.Execute(); err != nil {
		logger.Fail(err.Error())
		os.Exit(1)
	}
}

func initRoot() {
	rootCmd.Version = Version
	if rootCmd.Version == "" {
		rootCmd.Version = nextVersion
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	token := os.Getenv("MINEPKG_API_TOKEN")
	globalDir = filepath.Join(home, ".minepkg")
	credStore, err := credentials.New(globalDir, globals.ApiClient.APIUrl)
	if err != nil {
		if token != "" {
			logger.Warn("Could not initialize credential store: " + err.Error())
		} else {
			logger.Fail("Could not initialize credential store: " + err.Error())
		}
	}
	globals.CredStore = credStore

	if credStore.MinepkgAuth != nil {
		globals.ApiClient.JWT = credStore.MinepkgAuth.AccessToken
	}

	if token != "" {
		globals.ApiClient.JWT = token
		fmt.Println("Using MINEPKG_API_TOKEN for authentication")
	}

	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&disableColors, "no-color", "", false, "disable color output")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.minepkg/config.toml)")
	rootCmd.PersistentFlags().BoolP("accept-minecraft-eula", "a", false, "Accept Minecraft's eula. See https://www.minecraft.net/en-us/eula/")
	rootCmd.PersistentFlags().BoolP("system-java", "", false, "Use system java instead of internal installation for launching Minecraft server or client")
	rootCmd.PersistentFlags().BoolP("verbose", "", false, "More verbose logging. Not really implented yet")
	rootCmd.PersistentFlags().BoolP("non-interactive", "", false, "Use default answer for all prompts")

	viper.BindPFlag("noColor", rootCmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("useSystemJava", rootCmd.PersistentFlags().Lookup("system-java"))
	viper.BindPFlag("acceptMinecraftEula", rootCmd.PersistentFlags().Lookup("accept-minecraft-eula"))
	viper.BindPFlag("verboseLogging", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("nonInteractive", rootCmd.PersistentFlags().Lookup("non-interactive"))

	// subcommands
	rootCmd.AddCommand(dev.SubCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if viper.GetBool("noColor") || os.Getenv("CI") != "" {
		color.Disable()
		viper.Set("nonInteractive", true)
	}

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".minepkg" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
		viper.AddConfigPath("/etc/minepkg/")  // path to look for the config file in
		viper.AddConfigPath("$HOME/.minepkg") // call multiple times to add many search paths
		viper.AddConfigPath(".")              // optionally look for config in the working directory
	}

	viper.SetEnvPrefix("MINEPKG")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil && viper.GetBool("verboseLogging") {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	if viper.GetString("apiUrl") != "" {
		logger.Warn("NOT using default minepkg API URL: " + viper.GetString("apiUrl"))
		globals.ApiClient.APIUrl = viper.GetString("apiUrl")
	}
}
