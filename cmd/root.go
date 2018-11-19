package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fiws/minepkg/internals/cmdlog"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var logger *cmdlog.Logger = cmdlog.New()
var globalDir = "/tmp"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Version: "0.0.1",
	Use:     "minepkg",
	Short:   "Minepkg at your service.",
	Long: `Manage Minecraft mods with ease.

Examples:
  minepkg install rftools
  minepkg install https://minecraft.curseforge.com/projects/storage-drawers
  minepkg install https://github.com/McJtyMods/XNet/archive/1.12.zip
`,
}

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Fetches all mods that are available",
	Long: `minepkg uses a local db to resolve all dependencies. 
When these become out of date, you should run this.`,
	Run: func(cmd *cobra.Command, args []string) {
		refreshDb()
	},
}

var completionCmd = &cobra.Command{
	Use:   "completion",
	Args:  cobra.MaximumNArgs(1),
	Short: "Output shell completion code for bash",
	Long: `To load completion run

. <(minepkg completion)

You can add that line to your ~/.bashrc or ~/.profile to
keep enable completion in your shell.
`,
	Run: func(cmd *cobra.Command, args []string) {
		rootCmd.GenBashCompletion(os.Stdout)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	globalDir = filepath.Join(home, ".minepkg")

	// TODO: remove this after a few releases (fixes #61)
	os.Chmod(globalDir, 0755)

	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(refreshCmd)
	rootCmd.AddCommand(completionCmd)
	cobra.OnInitialize(initConfig)

	// Global flags
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.minepkg/config.toml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".minepkg" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".minepkg")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
