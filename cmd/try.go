package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/fiws/minepkg/cmd/launch"
	"github.com/fiws/minepkg/internals/instances"
	"github.com/fiws/minepkg/pkg/api"
	"github.com/fiws/minepkg/pkg/manifest"
	"github.com/spf13/cobra"
)

var (
	overwriteMcVersion     string
	overwriteFabricVersion string
	overwriteCompanion     string
	plain                  bool
)

func init() {
	tryCmd.Flags().BoolVarP(&serverMode, "server", "s", false, "Start a server instead of a client")
	tryCmd.Flags().StringVarP(&overwriteMcVersion, "requirements.minecraft", "", "", "Overwrite the required Minecraft version")
	tryCmd.Flags().StringVarP(&overwriteFabricVersion, "requirements.fabric", "", "", "Overwrite the required fabric version")
	tryCmd.Flags().StringVarP(&overwriteCompanion, "requirements.minepkgCompanion", "", "", "Overwrite the required minepkg companion version")
	tryCmd.Flags().BoolVarP(&plain, "plain", "p", false, "Do not include default mods for testing")
	rootCmd.AddCommand(tryCmd)
}

var tryCmd = &cobra.Command{
	Use:   "try <package>",
	Short: "Lets you try a mod or modpack in Minecraft",
	Long: `
This creates a temporary Minecraft instance which includes the given mod or modpack. 
It will be deleted after testing.
`,
	Aliases: []string{"test"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		tempDir, err := ioutil.TempDir("", args[0])
		wd, _ := os.Getwd()
		os.Chdir(tempDir) // change working directory to temporary dir

		defer os.RemoveAll(tempDir) // cleanup dir after minecraft is closed
		defer os.Chdir(wd)          // move back to working directory
		if err != nil {
			logger.Fail(err.Error())
		}
		instance := instances.Instance{
			GlobalDir:     globalDir,
			ModsDirectory: filepath.Join(tempDir, "mods"),
			Lockfile:      manifest.NewLockfile(),
			MinepkgAPI:    apiClient,
		}

		creds, err := ensureMojangAuth()
		if err != nil {
			logger.Fail(err.Error())
		}
		instance.MojangCredentials = creds.Mojang

		comp := strings.Split(args[0], "@")
		name := comp[0]
		version := "latest"
		if len(comp) == 2 {
			version = comp[1]
		}

		reqs := &api.RequirementQuery{
			Version:   version,
			Minecraft: "*",
			Plattform: "fabric", // TODO!!!
		}
		release, err := apiClient.FindRelease(context.TODO(), name, reqs)
		if err != nil {
			logger.Fail(err.Error())
		}
		if release == nil {
			logger.Info("Could not find package " + name + "@" + version)
			os.Exit(1)
		}

		instance.Manifest = release.Manifest
		fmt.Println("Creating temporary modpack with " + release.Identifier())

		if overwriteFabricVersion != "" {
			instance.Manifest.Requirements.Fabric = overwriteFabricVersion
		}
		if overwriteMcVersion != "" {
			fmt.Println("mc version overwritten!")
			instance.Manifest.Requirements.Minecraft = overwriteMcVersion
		}
		if overwriteCompanion != "" {
			fmt.Println("companion overwritten!")
			instance.Manifest.Requirements.MinepkgCompanion = overwriteCompanion
		}

		if plain != true && instance.Manifest.PlatformString() == "fabric" {
			instance.Manifest.AddDependency("fabric", "*")
			instance.Manifest.AddDependency("roughlyenoughitems", "*")
			instance.Manifest.AddDependency("modmenu", "*")
		}
		instance.Manifest.AddDependency(release.Package.Name, release.Package.Version)

		if err := instance.UpdateLockfileRequirements(context.TODO()); err != nil {
			logger.Fail(err.Error())
		}
		if err := instance.UpdateLockfileDependencies(context.TODO()); err != nil {
			logger.Fail(err.Error())
		}

		instance.SaveLockfile()
		// os.Exit(0)

		cliLauncher := launch.CLILauncher{Instance: &instance, ServerMode: serverMode}
		cliLauncher.Prepare()
		launchManifest := cliLauncher.LaunchManifest

		// TODO: This is just a hack
		if serverMode == true {
			launchManifest.MainClass = strings.Replace(launchManifest.MainClass, "Client", "Server", -1)
		}

		fmt.Println("\nLaunching Minecraft …")
		opts := &instances.LaunchOptions{
			LaunchManifest: launchManifest,
			Server:         serverMode,
		}
		err = cliLauncher.Launch(opts)
		if err != nil {
			logger.Fail(err.Error())
		}
	},
}
