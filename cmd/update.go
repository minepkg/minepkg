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

func init() {
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(updateReqCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "updates all installed dependencies",
	Long: `
This updates the local mods according to the minepkg.toml. 
Edit the minepkg.toml to change the version requirements.
`,
	Aliases: []string{"upd"},
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		instance, err := instances.DetectInstance()
		instance.MinepkgAPI = apiClient
		if err != nil {
			logger.Fail("Instance problem: " + err.Error())
		}
		fmt.Printf("Installing to %s\n", instance.Desc())
		fmt.Println() // empty line

		installManifest(instance)
	},
}

var updateReqCmd = &cobra.Command{
	Use:   "update-requirements",
	Short: "updates installed requirements (minecraft & loader version)",
	Long: `
This updates the installed requirements according to the minepkg.toml. 
Edit the minepkg.toml to change the version requirements.
`,
	Aliases: []string{"update-req"},
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		instance, err := instances.DetectInstance()
		instance.MinepkgAPI = apiClient
		if err != nil {
			logger.Fail("Instance problem: " + err.Error())
		}
		fmt.Printf("Installing to %s\n\n", instance.Desc())

		logger.Headline("Updating installed requirements")

		instance.UpdateLockfileRequirements(context.TODO())
		lock := instance.Lockfile
		switch {
		case lock.Fabric != nil:
			fmt.Println("  Fabric loader: " + lock.Fabric.FabricLoader)
			fmt.Println("  Fabric mapping: " + lock.Fabric.Mapping)
			fmt.Println("  Minecraft: " + lock.Fabric.Minecraft)
		case lock.Forge != nil:
			fmt.Println("  Forge loader: " + lock.Forge.ForgeLoader)
			fmt.Println("  Minecraft: " + lock.Forge.Minecraft)
		default:
			fmt.Println("  Minecraft: " + lock.Vanilla.Minecraft)
		}
		instance.SaveLockfile()
		fmt.Println()

		s := spinner.New(spinner.CharSets[9], 300*time.Millisecond) // Build our new spinner
		s.Prefix = " "
		s.Start()

		mgr := downloadmgr.New()
		mgr.OnProgress = func(p int) {
			s.Suffix = fmt.Sprintf(" Downloading %v", p) + "%"
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
			target := filepath.Join(instance.GlobalDir, "assets/objects", asset.UnixPath())
			mgr.Add(downloadmgr.NewHTTPItem(asset.DownloadURL(), target))
		}

		for _, lib := range missingLibs {
			target := filepath.Join(instance.GlobalDir, "libraries", lib.Filepath())
			mgr.Add(downloadmgr.NewHTTPItem(lib.DownloadURL(), target))
		}

		if err = mgr.Start(context.TODO()); err != nil {
			logger.Fail(err.Error())
		}
		s.Stop()

		installManifest(instance)
	},
}
