package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fiws/minepkg/pkg/api"
	"github.com/fiws/minepkg/pkg/manifest"

	"github.com/briandowns/spinner"
	"github.com/fiws/minepkg/internals/downloadmgr"
	"github.com/fiws/minepkg/internals/instances"
	"github.com/spf13/cobra"
)

func init() {
	tryCmd.Flags().BoolVarP(&serverMode, "server", "s", false, "Start a server instead of a client")
}

var tryCmd = &cobra.Command{
	Use:     "try",
	Short:   "Try a mod or modpack withouth creating a modpack first",
	Long:    ``, // TODO
	Aliases: []string{"test"},
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
			Directory:     globalDir,
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

		// TODO: if fabric !!!
		{
			// TODO: use real requirements !!!
			instance.Manifest.Requirements.Fabric = "*"
			instance.Manifest.Requirements.Minecraft = "1.14.3"
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

		// Prepare launch
		s := spinner.New(spinner.CharSets[9], 300*time.Millisecond) // Build our new spinner
		s.Prefix = " "
		s.Start()
		s.Suffix = " Preparing launch"

		java := javaBin(instance.Directory)
		if java == "" {
			s.Suffix = " Preparing launch – Downloading java"
			var err error
			java, err = downloadJava(instance.Directory)
			if err != nil {
				logger.Fail(err.Error())
			}
		}

		mgr := downloadmgr.New()
		mgr.OnProgress = func(p int) {
			s.Suffix = fmt.Sprintf(" Preparing launch – Downloading %v", p) + "%"
		}

		launchManifest, err := instance.GetLaunchManifest()
		if err != nil {
			logger.Fail(err.Error())
		}

		if serverMode != true {
			missingAssets, err := instance.FindMissingAssets(launchManifest)
			if err != nil {
				logger.Fail(err.Error())
			}

			for _, asset := range missingAssets {
				target := filepath.Join(instance.Directory, "assets/objects", asset.UnixPath())
				mgr.Add(downloadmgr.NewHTTPItem(asset.DownloadURL(), target))
			}
		}

		missingLibs, err := instance.FindMissingLibraries(launchManifest)
		if err != nil {
			logger.Fail(err.Error())
		}

		for _, lib := range missingLibs {
			target := filepath.Join(instance.Directory, "libraries", lib.Filepath())
			mgr.Add(downloadmgr.NewHTTPItem(lib.DownloadURL(), target))
		}

		if err = mgr.Start(context.TODO()); err != nil {
			logger.Fail(err.Error())
		}

		s.Suffix = " Downloading dependencies"
		if err := instance.EnsureDependencies(context.TODO()); err != nil {
			logger.Fail(err.Error())
		}

		s.Stop()

		// TODO: This is just a hack
		if serverMode == true {
			launchManifest.MainClass = strings.Replace(launchManifest.MainClass, "Client", "Server", -1)
		}

		fmt.Println("\nLaunching Minecraft …")
		opts := &instances.LaunchOptions{
			LaunchManifest: launchManifest,
			Java:           java,
			Server:         serverMode,
		}
		err = instance.Launch(opts)
		if err != nil {
			logger.Fail(err.Error())
		}
	},
}
