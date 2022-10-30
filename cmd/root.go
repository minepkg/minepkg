package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwalton/gchalk"
	"github.com/minepkg/minepkg/cmd/bump"
	"github.com/minepkg/minepkg/cmd/config"
	"github.com/minepkg/minepkg/cmd/dev"
	"github.com/minepkg/minepkg/cmd/initCmd"
	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/auth"
	"github.com/minepkg/minepkg/internals/cmdlog"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/internals/credentials"
	"github.com/minepkg/minepkg/internals/globals"
	"github.com/minepkg/minepkg/internals/instances"
	"github.com/minepkg/minepkg/internals/ownhttp"
	"github.com/minepkg/minepkg/internals/provider"
	"github.com/minepkg/minepkg/pkg/manifest"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

var (
	cfgFile string
	// Version is the current version. it should be set by goreleaser
	Version string
	// commit is also set by goreleaser (in main.go)
	Commit string
	// nextVersion is a placeholder version. only used for local dev
	nextVersion string = "0.1.0-dev-local"
)

type Root struct {
	HTTPClient         *http.Client
	MinepkgAPI         *api.MinepkgAPI
	authProvider       auth.AuthProvider
	minecraftAuthStore *credentials.Store
	minepkgAuthStore   *credentials.Store
	globalDir          string
	logger             *cmdlog.Logger
	NonInteractive     bool
	ProviderStore      *provider.Store
}

func newRoot() *Root {
	http := ownhttp.New()

	minepkgClient := api.NewWithClient(http)

	providers := map[string]provider.Provider{
		"minepkg":  &provider.MinepkgProvider{Client: minepkgClient},
		"modrinth": provider.NewModrinthProvider(),
		"https":    provider.NewHTTPSProvider(),
		"dummy":    provider.NewDummyProvider(),
	}

	return &Root{
		HTTPClient:     http,
		MinepkgAPI:     api.NewWithClient(http),
		logger:         globals.Logger,
		NonInteractive: false,
		ProviderStore:  provider.NewStore(providers),
	}
}

var root = newRoot()

func (r *Root) setMinepkgAuth(token *oauth2.Token) error {
	// TODO: no use of globals!
	globals.ApiClient.JWT = token.AccessToken
	return r.minepkgAuthStore.Set(token)
}

func (r *Root) validateManifest(man *manifest.Manifest) error {
	logger.Log("Validating minepkg.toml")
	problems := man.Validate()
	fatal := false
	for _, problem := range problems {
		if problem.Level == manifest.ErrorLevelFatal {
			fmt.Printf(
				"%s ERROR: %s\n",
				commands.Emoji("❌"),
				problem.Error(),
			)
			fatal = true
		} else {
			fmt.Printf(
				"%s WARNING: %s\n",
				commands.Emoji("⚠️ "),
				problem.Error(),
			)
		}
	}
	if fatal {
		return errors.New("validation of minepkg.toml failed")
	}
	return nil
}

func (r *Root) LocalInstance() (*instances.Instance, error) {
	var err error
	instance, err := instances.NewFromWd()
	if err != nil {
		return nil, err
	}
	instance.MinepkgAPI = globals.ApiClient
	instance.ProviderStore = r.ProviderStore

	return instance, err
}

var logger = root.logger

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	// Version gets set dynamically
	Use:   "minepkg",
	Short: "Minepkg at your service.",
	Long:  "Manage Minecraft mods with ease",

	Example: `
  minepkg init
  minepkg install modmenu@latest
  minepkg join demo.minepkg.host`,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	initRoot()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(commands.ErrorBox(err.Error(), ""))
		os.Exit(1)
	}
}

func initRoot() {
	// include commit if this is next version
	if strings.HasSuffix(Version, "-next") {
		Version = Version + "+" + Commit
	}
	rootCmd.Version = Version
	if rootCmd.Version == "" {
		rootCmd.Version = nextVersion
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	configPath := filepath.Join(configDir, "minepkg")

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default is %s/config.toml)", configPath))
	rootCmd.PersistentFlags().BoolP("accept-minecraft-eula", "a", false, "Accept Minecraft's eula. See https://www.minecraft.net/en-us/eula/")
	rootCmd.PersistentFlags().BoolP("verbose", "", false, "More verbose logging. Not really implemented yet")
	rootCmd.PersistentFlags().BoolP("non-interactive", "", false, "Do not prompt for anything (use defaults instead)")

	viper.BindPFlag("useSystemJava", rootCmd.PersistentFlags().Lookup("system-java"))
	viper.BindPFlag("acceptMinecraftEula", rootCmd.PersistentFlags().Lookup("accept-minecraft-eula"))
	viper.BindPFlag("verboseLogging", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("nonInteractive", rootCmd.PersistentFlags().Lookup("non-interactive"))
	cobra.OnInitialize(initConfig)
	// viper.SetDefault("init.defaultSource", "https://github.com/")

	// subcommands
	rootCmd.AddCommand(dev.SubCmd)
	rootCmd.AddCommand(config.SubCmd)
	rootCmd.AddCommand(initCmd.New())
	rootCmd.AddCommand(bump.New())
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if viper.GetBool("noColor") {
		gchalk.ForceLevel(gchalk.LevelNone)
	}

	if !viper.GetBool("verboseLogging") {
		log.Default().SetOutput(ioutil.Discard)
	} else {
		log.Println("Verbose logging enabled")
	}

	if os.Getenv("CI") != "" {
		gchalk.ForceLevel(gchalk.LevelNone)
		viper.Set("nonInteractive", true)
	}

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		configDir, err := os.UserConfigDir()
		if err != nil {
			panic(err)
		}

		viper.SetConfigName("config")
		viper.SetConfigType("toml")
		viper.AddConfigPath("/etc/minepkg/")                     // path to look for the config file in
		viper.AddConfigPath(filepath.Join(configDir, "minepkg")) // call multiple times to add many search paths
		viper.AddConfigPath(".")                                 // optionally look for config in the working directory
	}

	viper.SetEnvPrefix("MINEPKG")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil && viper.GetBool("verboseLogging") {
		log.Println("Using config file:", viper.ConfigFileUsed())
	}

	if viper.GetString("apiUrl") != "" {
		logger.Warn("NOT using default minepkg API URL: " + viper.GetString("apiUrl"))
		globals.ApiClient.APIUrl = viper.GetString("apiUrl")
	}

	homeConfigs, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}
	apiKey := os.Getenv("MINEPKG_API_KEY")
	root.globalDir = filepath.Join(homeConfigs, "minepkg")
	root.minecraftAuthStore = credentials.New(root.globalDir, "minecraft_auth")
	root.NonInteractive = viper.GetBool("nonInteractive")
	parsedUrl, err := url.Parse(globals.ApiClient.APIUrl)
	if err != nil {
		panic(fmt.Errorf("invalid minepkg API URL: %w", err))
	}
	// use the host part of the url as the minepkg auth store key (eg. api.minepkg.io)
	root.minepkgAuthStore = credentials.New(root.globalDir, parsedUrl.Host)
	if err != nil {
		if apiKey != "" {
			logger.Warn("Could not initialize credential store: " + err.Error())
		} else {
			logger.Fail("Could not initialize credential store: " + err.Error())
		}
	}

	var minepkgAuth *oauth2.Token
	root.minepkgAuthStore.Get(&minepkgAuth)

	if minepkgAuth != nil {
		log.Println("Using minepkg JWT")
		globals.ApiClient.JWT = minepkgAuth.AccessToken
	} else {
		log.Println("No minepkg JWT found")
	}

	if apiKey != "" {
		globals.ApiClient.APIKey = apiKey
		fmt.Println("Using MINEPKG_API_KEY for authentication")
	}
}
