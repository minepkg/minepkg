package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fiws/minepkg/internals/downloadmgr"
	"github.com/fiws/minepkg/internals/instances"
	"github.com/spf13/cobra"
)

var version string
var listVersions bool

var launchCmd = &cobra.Command{
	Use:     "launch",
	Short:   "Launch a minecraft instance",
	Long:    ``, // TODO
	Aliases: []string{"run", "start", "play"},
	Run: func(cmd *cobra.Command, args []string) {
		instance, err := instances.DetectInstance()
		if err != nil {
			logger.Fail("Instance problem: " + err.Error())
		}
		// list versions instead of launching
		if listVersions == true {
			logger.Headline("Available Versions:")
			for _, version := range instance.AvailableVersions() {
				logger.Log(" - " + version.String())
			}
			return
		}

		// launch instance
		fmt.Printf("Launching %s\n", instance.Desc())
		if loginData.Mojang == nil {
			logger.Info("You need to sign in with your mojang account to launch minecraft")
			login()
		}
		instance.MojangCredentials = loginData.Mojang

		// Prepare launch
		s := spinner.New(spinner.CharSets[9], 300*time.Millisecond) // Build our new spinner
		s.Prefix = " "
		s.Start()
		s.Suffix = " Preparing launch"

		mgr := downloadmgr.New()
		mgr.OnProgress = func(p int) {
			s.Suffix = fmt.Sprintf(" Preparing launch %v", p) + "%"
		}

		launchManifest, err := instance.GetLaunchManifest()
		if err != nil {
			logger.Fail(err.Error())
		}

		missingAssets, err := instance.FindMissingAssets(launchManifest)
		if err != nil {
			logger.Fail(err.Error())
		}

		missingLibs, err := instance.FindMissingLibraries(launchManifest)
		if err != nil {
			logger.Fail(err.Error())
		}

		for _, asset := range missingAssets {
			target := filepath.Join(instance.Directory, "assets/objects", asset.UnixPath())
			mgr.Add(downloadmgr.NewHTTPItem(asset.DownloadURL(), target))
		}

		for _, lib := range missingLibs {
			target := filepath.Join(instance.Directory, "libraries", lib.Filepath())
			mgr.Add(downloadmgr.NewHTTPItem(lib.DownloadURL(), target))
		}

		mgr.Start(context.TODO())
		s.Stop()

		opts := &instances.LaunchOptions{
			LaunchManifest: launchManifest,
			SkipDownload:   true,
		}
		err = instance.Launch(opts)
		if err != nil {
			logger.Fail(err.Error())
		}
	},
}

func init() {
	// launchCmd.Flags().StringVarP(&version, "run-version", "r", "", "Version to start. Uses the latest compatible if not present")
	launchCmd.Flags().BoolVarP(&listVersions, "list-versions", "", false, "List available versions instead of starting minecraft")
}
