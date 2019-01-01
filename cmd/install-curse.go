package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fiws/minepkg/internals/cmdlog"
	"github.com/fiws/minepkg/internals/curse"
	"github.com/fiws/minepkg/internals/instances"
	"github.com/manifoldco/promptui"
)

func installFromCurse(name string, instance *instances.McInstance) {

	task := logger.NewTask(3)
	task.Step("📚", "Searching local mod DB.")
	db := readDbOrDownload()

	// TODO: better search!
	mods := curse.Filter(db.Mods, func(m curse.Mod) bool {
		return strings.HasPrefix(strings.ToLower(m.Slug), name)
	})

	choosenMod := chooseMod(mods, task)

	task.Step("🔎", "Resolving Dependencies")
	resolver := curse.NewResolver()

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond) // Build our new spinner
	s.Prefix = "  "
	s.Suffix = "  Resolving " + choosenMod.Slug
	s.Start()
	instance.Manifest.AddDependency(choosenMod)
	err := resolver.ResolvePackage(choosenMod, instance.Manifest.Requirements.Minecraft)
	resolved := resolver.Resolved
	s.Stop()
	if err != nil {
		logger.Fail(err.Error())
	}

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
