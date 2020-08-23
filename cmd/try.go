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
	"github.com/spf13/viper"
)

var (
	tryBase                string
	overwriteMcVersion     string
	overwriteFabricVersion string
	overwriteCompanion     string
	plain                  bool
	photosession           bool
)

func init() {
	tryCmd.Flags().StringVarP(&tryBase, "base", "b", "test-mansion", "Base modpack to use for testing")
	tryCmd.Flags().BoolVarP(&serverMode, "server", "s", false, "Start a server instead of a client")
	tryCmd.Flags().StringVarP(&overwriteMcVersion, "requirements.minecraft", "", "", "Overwrite the required Minecraft version")
	tryCmd.Flags().StringVarP(&overwriteFabricVersion, "requirements.fabric", "", "", "Overwrite the required fabric version")
	tryCmd.Flags().StringVarP(&overwriteCompanion, "requirements.minepkgCompanion", "", "", "Overwrite the required minepkg companion version")
	tryCmd.Flags().BoolVarP(&plain, "plain", "p", false, "Do not include default mods for testing")
	tryCmd.Flags().BoolVarP(&photosession, "photosession", "", false, "Upload all screenshots (take with F2) to the project")
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
			GlobalDir:  globalDir,
			Directory:  tempDir,
			Lockfile:   manifest.NewLockfile(),
			MinepkgAPI: apiClient,
		}

		creds, err := ensureMojangAuth()
		if err != nil {
			logger.Fail(err.Error())
		}
		instance.MojangCredentials = creds

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
		if err != nil && err != api.ErrNotMatchingRelease {
			logger.Fail(err.Error())
		}
		if release == nil {
			// TODO: check if this was a 404
			project := searchFallback(context.TODO(), name)
			if project == nil {
				logger.Info("Could not find package " + name + "@" + version)
				os.Exit(1)
			}

			release, err = apiClient.FindRelease(context.TODO(), project.Name, reqs)
			if err != nil || release == nil {
				logger.Info("Could not find package " + name + "@" + version)
				os.Exit(1)
			}
		}

		// set instance details
		instance.Manifest = manifest.NewInstanceLike(release.Manifest)
		fmt.Println("Creating temporary modpack with " + release.Identifier())

		// overwrite some instance launch options with flags
		instanceReqOverwrites(&instance)

		if instance.Manifest.Requirements.Minecraft == "*" {
			fmt.Println("mc * resolved to: " + release.LatestTestedMinecraftVersion())
			instance.Manifest.Requirements.Minecraft = release.LatestTestedMinecraftVersion()
		}

		startSave := ""
		if plain != true && instance.Manifest.Package.Type != manifest.TypeModpack && instance.Manifest.PlatformString() == "fabric" {
			instance.Manifest.AddDependency(tryBase, "*")
			// TODO: make this generic
			if tryBase == "test-mansion" {
				startSave = "test-mansion"
			}
		}

		// add/overwrite the wanted mod or modpack
		instance.Manifest.AddDependency(release.Package.Name, release.Package.Version)

		if viper.GetBool("useSystemJava") == true {
			instance.UseSystemJava()
		}

		cliLauncher := launch.CLILauncher{Instance: &instance, ServerMode: serverMode, NonInteractive: viper.GetBool("nonInteractive")}
		if err := cliLauncher.Prepare(); err != nil {
			logger.Fail(err.Error())
		}
		launchManifest := cliLauncher.LaunchManifest

		fmt.Println("\n[launch settings]")
		fmt.Println("directory: " + instance.Directory)
		fmt.Println("mc directory: " + instance.McDir())
		fmt.Println("platform: " + instance.Manifest.PlatformString())
		fmt.Println("minecraft: " + instance.Manifest.Requirements.Minecraft)
		if instance.Manifest.PlatformString() == "fabric" {
			fmt.Printf(
				"fabric: %s / %s (loader / mapping)\n",
				instance.Lockfile.Fabric.FabricLoader,
				instance.Lockfile.Fabric.Mapping,
			)
		}

		depNames := make([]string, len(instance.Lockfile.Dependencies))
		i := 0
		for name, lock := range instance.Lockfile.Dependencies {
			depNames[i] = name + "@" + lock.Version
			i++
		}
		fmt.Println("[dependencies] \n - " + strings.Join(depNames, "\n - "))

		// TODO: This is just a hack
		if serverMode == true {
			launchManifest.MainClass = strings.Replace(launchManifest.MainClass, "Client", "Server", -1)

			// TODO: better handling
			if viper.GetBool("acceptMinecraftEula") == true {
				eula := "# generated by minepkg\n# https://account.mojang.com/documents/minecraft_eula\neula=true\n"
				ioutil.WriteFile(filepath.Join(instance.McDir(), "./eula.txt"), []byte(eula), 0644)
			}
		}

		fmt.Println("\n== Launching Minecraft ==")
		opts := &instances.LaunchOptions{
			LaunchManifest: launchManifest,
			Server:         serverMode,
			StartSave:      startSave,
		}
		err = cliLauncher.Launch(opts)
		if err != nil {
			logger.Fail(err.Error())
		}

		if photosession == true {
			screenshotDir := filepath.Join(instance.McDir(), "./screenshots")
			entries, err := ioutil.ReadDir(screenshotDir)
			if err != nil {
				fmt.Println("No screenshots taken, skipping upload")
				return
			}
			fmt.Println("uploading screenshots now")
			for _, e := range entries {
				if e.IsDir() != true {
					fmt.Println("uploading " + e.Name())
					f, err := os.Open(filepath.Join(screenshotDir, e.Name()))
					if err != nil {
						fmt.Println("Could not open screenshot " + e.Name())
						fmt.Println(err)
						continue
					}
					err = apiClient.PostProjectMedia(context.TODO(), release.Package.Name, f)
					if err != nil {
						fmt.Println("Could not upload screenshot: " + err.Error())
					}
				}
			}
		}
	},
}
