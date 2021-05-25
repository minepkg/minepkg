package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/minepkg/minepkg/cmd/launch"
	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/internals/globals"
	"github.com/minepkg/minepkg/internals/instances"
	"github.com/minepkg/minepkg/pkg/manifest"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	runner := &tryRunner{}
	cmd := commands.New(&cobra.Command{
		Use:   "try <package>",
		Short: "Lets you try a mod or modpack in Minecraft",
		Long: `
This creates a temporary Minecraft instance which includes the given mod or modpack.
It will be deleted after testing.
	`,
		Aliases: []string{"test"},
		Args:    cobra.ExactArgs(1),
	}, runner)

	cmd.Flags().StringVarP(&runner.tryBase, "base", "b", "test-mansion", "Base modpack to use for testing")
	cmd.Flags().BoolVarP(&runner.serverMode, "server", "s", false, "Start a server instead of a client")
	cmd.Flags().BoolVarP(&runner.offlineMode, "offline", "", false, "Start the server in offline mode (server only)")
	cmd.Flags().BoolVarP(&runner.plain, "plain", "p", false, "Do not include default mods for testing")
	cmd.Flags().BoolVarP(&runner.photosession, "photosession", "", false, "Upload all screenshots (take with F2) to the project")

	runner.overwrites = launch.CmdOverwriteFlags(cmd.Command)

	rootCmd.AddCommand(cmd.Command)
}

type tryRunner struct {
	tryBase      string
	plain        bool
	photosession bool
	serverMode   bool
	offlineMode  bool

	overwrites *launch.OverwriteFlags
}

func (t *tryRunner) RunE(cmd *cobra.Command, args []string) error {
	apiClient := globals.ApiClient

	tempDir, err := ioutil.TempDir("", args[0])
	wd, _ := os.Getwd()
	os.Chdir(tempDir) // change working directory to temporary dir

	defer os.RemoveAll(tempDir) // cleanup dir after minecraft is closed
	defer os.Chdir(wd)          // move back to working directory
	if err != nil {
		return err
	}
	instance := instances.NewEmptyInstance()
	instance.Directory = tempDir
	instance.Lockfile = manifest.NewLockfile()
	instance.MinepkgAPI = apiClient

	creds, err := ensureMojangAuth()
	if err != nil {
		return err
	}
	instance.MojangCredentials = creds

	comp := strings.Split(args[0], "@")
	name := comp[0]
	version := "latest"
	if len(comp) == 2 {
		version = comp[1]
	}

	var mcVersion string
	if t.overwrites.McVersion != "" {
		mcVersion = t.overwrites.McVersion
	}

	release, err := apiClient.ReleasesQuery(context.TODO(), &api.ReleasesQuery{
		Platform:     "fabric", // TODO!!!
		Name:         name,
		Minecraft:    mcVersion,
		VersionRange: version,
	})
	var e *api.ErrNoMatchingRelease
	if err != nil && !errors.As(err, &e) {
		return err
	}
	if release == nil {
		// TODO: check if this was a 404
		project := searchFallback(context.TODO(), name)
		if project == nil {
			logger.Info("Could not find package " + name + "@" + version)
			os.Exit(1)
		}

		release, err = apiClient.ReleasesQuery(context.TODO(), &api.ReleasesQuery{
			Platform:     "fabric", // TODO!!!
			Name:         project.Name,
			Minecraft:    mcVersion,
			VersionRange: version,
		})
		if err != nil || release == nil {
			logger.Info("Could not find package " + name + "@" + version)
			os.Exit(1)
		}
	}

	// set instance details
	instance.Manifest = manifest.NewInstanceLike(release.Manifest)
	fmt.Println("Creating temporary modpack with " + release.Identifier())

	// overwrite some instance launch options with flags
	launch.ApplyInstanceOverwrites(instance, t.overwrites)

	if t.overwrites.McVersion == "" {
		fmt.Println("mc * resolved to: " + release.LatestTestedMinecraftVersion())
		instance.Manifest.Requirements.Minecraft = release.LatestTestedMinecraftVersion()
	}

	startSave := ""
	if !t.plain && instance.Manifest.Package.Type != manifest.TypeModpack && instance.Manifest.PlatformString() == "fabric" {
		instance.Manifest.AddDependency(t.tryBase, "*")
		// TODO: make this generic
		if t.tryBase == "test-mansion" {
			startSave = "test-mansion"
		}
	}

	// add/overwrite the wanted mod or modpack
	instance.Manifest.AddDependency(release.Package.Name, release.Package.Version)

	if viper.GetBool("useSystemJava") {
		instance.UseSystemJava()
	}

	cliLauncher := launch.CLILauncher{
		Instance:       instance,
		ServerMode:     t.serverMode,
		OfflineMode:    t.offlineMode,
		MinepkgVersion: rootCmd.Version,
		NonInteractive: viper.GetBool("nonInteractive"),
	}
	if err := cliLauncher.Prepare(); err != nil {
		return err
	}
	launchManifest := cliLauncher.LaunchManifest

	opts := &instances.LaunchOptions{
		LaunchManifest: launchManifest,
		Server:         t.serverMode,
		Offline:        t.offlineMode,
		StartSave:      startSave,
	}
	err = cliLauncher.Launch(opts)
	if err != nil {
		return err
	}

	if t.photosession {
		screenshotDir := filepath.Join(instance.McDir(), "./screenshots")
		entries, err := ioutil.ReadDir(screenshotDir)
		if err != nil {
			fmt.Println("No screenshots taken, skipping upload")
			return nil
		}
		fmt.Println("uploading screenshots now")
		for _, e := range entries {
			if !e.IsDir() {
				fmt.Println("uploading " + e.Name())
				f, err := os.Open(filepath.Join(screenshotDir, e.Name()))
				if err != nil {
					fmt.Println("Could not open screenshot " + e.Name())
					fmt.Println(err)
					continue
				}
				defer f.Close()
				err = apiClient.PostProjectMedia(context.TODO(), release.Package.Name, f)
				if err != nil {
					fmt.Println("Could not upload screenshot: " + err.Error())
				}
			}
		}
	}
	return err
}
