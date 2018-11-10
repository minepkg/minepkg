package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fiws/minepkg/internals/cmdlog"

	"github.com/briandowns/spinner"

	"github.com/fiws/minepkg/internals/curse"
	"github.com/fiws/minepkg/internals/instances"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:     "install [name/url/id ...]",
	Short:   "installz packages",
	Long:    `Just install them packages noaw`,
	Aliases: []string{"isntall", "i"},
	Run: func(cmd *cobra.Command, args []string) {
		instance, err := instances.DetectInstance()
		if err != nil {
			logger.Fail("Instance problem: " + err.Error())
		}
		fmt.Printf("Installing to %s\n", instance.Desc())
		if instance.Flavour == instances.FlavourMMC {
			logger.Warn("MultiMC support is not officialy endorsed.")
			// logger.Log("Please report possible bugs to http://github.com/fiws/minepkg/issues and NOT to MultiMC.")
		}
		fmt.Println() // empty line

		// installing minepkg.toml dependencies
		if len(args) == 0 {
			installManifest(instance)
			return
		}

		task := logger.NewTask(3)
		task.Step("📚", "Searching local mod DB.")
		db := readDbOrDownload()

		// TODO: better search!
		mods := curse.Filter(db.Mods, func(m curse.Mod) bool {
			return strings.HasPrefix(strings.ToLower(m.Name), strings.Join(args, " "))
		})

		choosenMod := chooseMod(mods, task)

		task.Step("🔎", "Resolving Dependencies")
		resolver := curse.NewResolver()

		instance.Manifest.AddDependency(choosenMod)
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond) // Build our new spinner
		s.Prefix = "  "
		s.Suffix = "  Resolving " + choosenMod.Slug
		s.Start()
		resolver.Resolve(choosenMod.ID)
		resolved := resolver.Resolved
		s.Stop()

		for _, mod := range resolved {
			task.Log(fmt.Sprintf("requires %s", mod.FileName))
		}

		// download mod phase
		task.Step("🚚", "Downloading Mods")
		for _, mod := range resolved {
			err := instance.Download(&mod)
			if err != nil {
				logger.Fail(fmt.Sprintf("Could not download %s (%s)"+mod.FileName, err))
			}
			task.Log(fmt.Sprintf("downloaded %s", mod.FileName))
		}

		// save to minepkg.toml
		if err := instance.Manifest.Save(); err != nil {
			logger.Fail("Could not update minepkg.toml: " + err.Error())
		}
		logger.Info("minepkg.toml has been updated")
	},
}

// installManifest installs dependencies from the minepkg.toml
func installManifest(instance *instances.McInstance) {
	task := logger.NewTask(3)

	task.Step("📚", "Searching local mod DB.")
	db := readDbOrDownload()

	deps, err := instance.Manifest.FullDependencies()
	if err != nil {
		task.Fail("Failed to extend " + err.Error())
	}

	mods := make([]*curse.Mod, len(*deps))

	i := 0
	for name := range *deps {
		resolved := db.modBySlug(name)
		if resolved == nil {
			task.Fail("Could not resolve " + name)
		}
		mods[i] = resolved
		i++
	}

	task.Step("🔎", "Resolving Dependencies")
	resolver := curse.NewResolver()

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond) // Build our new spinner
	s.Prefix = "  "
	s.Start()
	for _, mod := range mods {
		s.Suffix = "  Resolving " + mod.Slug
		resolver.Resolve(mod.ID)
	}
	s.Stop()
	resolved := resolver.Resolved

	for _, mod := range resolved {
		task.Log(fmt.Sprintf("requires %s", mod.FileName))
	}

	task.Step("🚚", "Downloading Mods")

	for _, mod := range resolved {
		err := instance.Download(&mod)
		if err != nil {
			logger.Fail(fmt.Sprintf("Could not download %s (%s)"+mod.FileName, err))
		}
		task.Log(fmt.Sprintf("downloaded %s", mod.FileName))
	}
}

func chooseMod(mods []curse.Mod, task *cmdlog.Task) *curse.Mod {
	modCount := len(mods)
	var choosen *curse.Mod
	switch {
	case modCount == 0:
		task.Fail("Found no matching packages by that name")
	case modCount == 1:
		choosen = &mods[0]
		prompt := promptui.Prompt{
			Label:     "Install " + choosen.Name,
			IsConfirm: true,
			Default:   "Y",
		}

		_, err := prompt.Run()
		if err != nil {
			logger.Info("Aborting installation")
			os.Exit(0)
		}
	default:
		task.Info("Found multiple packages by that name, please select one.")
		curse.SortByDownloadCount(mods)

		selectable := make([]string, modCount)
		for i, mod := range mods {
			selectable[i] = fmt.Sprintf("%s (%v)", mod.Name, HumanFloat32(mod.DownloadCount))
		}

		prompt := promptui.Select{
			Label: "Select Package",
			Items: selectable,
			Size:  8,
		}

		i, _, err := prompt.Run()
		if err != nil {
			fmt.Println("Aborting installation")
			os.Exit(0)
		}
		choosen = &mods[i]
	}
	return choosen
}
