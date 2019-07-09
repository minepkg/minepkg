package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fiws/minepkg/pkg/api"

	"github.com/fiws/minepkg/internals/downloadmgr"
	"github.com/fiws/minepkg/internals/instances"
	"github.com/manifoldco/promptui"
)

func installFromMinepkg(mods []string, instance *instances.Instance) error {

	cacheDir := filepath.Join(globalDir, "cache")
	os.MkdirAll(cacheDir, os.ModePerm)

	task := logger.NewTask(3)
	task.Step("📚", "Searching requested package")

	releases := make([]*api.Release, len(mods))

	s := spinner.New(spinner.CharSets[9], 300*time.Millisecond) // Build our new spinner
	s.Prefix = " "

	mgr := downloadmgr.New()
	mgr.OnProgress = func(p int) {
		s.Suffix = fmt.Sprintf(" Downloading %v", p) + "%"
	}

	// resolve requirements
	if instance.Lockfile == nil || instance.Lockfile.HasRequirements() == false {
		s.Suffix = " Resolving Requirements"
		instance.UpdateLockfileRequirements(context.TODO())
		instance.SaveLockfile()
	}

	for i, name := range mods {
		comp := strings.Split(name, "@")
		name = comp[0]
		version := "latest"
		if len(comp) == 2 {
			version = comp[1]
		}

		reqs := &api.RequirementQuery{
			Version:   version,
			Minecraft: instance.Lockfile.MinecraftVersion(),
			Plattform: instance.Manifest.PlatformString(),
		}

		release, err := apiClient.FindRelease(context.TODO(), name, reqs)
		if err != nil {
			return err
		}
		if release == nil {
			logger.Info("Could not find package " + name + "@" + version)
			os.Exit(1)
		}
		releases[i] = release
	}

	if len(releases) == 1 {
		logger.Info("Installing " + releases[0].Package.Name + "@" + releases[0].Package.Version)
	} else {
		// TODO: list mods
		prompt := promptui.Prompt{
			Label:     fmt.Sprintf("Install %d mods", len(releases)),
			IsConfirm: true,
			Default:   "Y",
		}

		_, err := prompt.Run()
		if err != nil {
			logger.Info("Aborting installation")
			os.Exit(0)
		}
	}

	task.Step("🔎", "Resolving Dependencies")
	for _, release := range releases {
		instance.Manifest.AddDependency(release.Package.Name, release.Package.Version)
	}
	instance.UpdateLockfileDependencies(context.TODO())
	for _, dep := range instance.Lockfile.Dependencies {
		fmt.Printf(" - %s@%s\n", dep.Project, dep.Version)
	}
	missingFiles, err := instance.FindMissingDependencies()
	if err != nil {
		logger.Fail(err.Error())
	}

	task.Step("🚚", fmt.Sprintf("Downloading %d Packages", len(missingFiles)))
	for _, m := range missingFiles {
		p := filepath.Join(cacheDir, m.Project, m.Version+".jar")
		mgr.Add(downloadmgr.NewHTTPItem(m.URL, p))
	}

	s.Start()
	if err := mgr.Start(context.TODO()); err != nil {
		logger.Fail(err.Error())
	}

	instance.LinkDependencies()

	s.Stop()
	instance.SaveManifest()
	instance.SaveLockfile()
	fmt.Println("updated minepkg.toml")
	fmt.Println("You can now launch Minecraft using \"minepkg launch\"")

	return nil
}

// func chooseMod(mods []curse.Mod, task *cmdlog.Task) *curse.Mod {
// 	modCount := len(mods)
// 	var choosen *curse.Mod
// 	switch {
// 	case modCount == 0:
// 		task.Fail("Found no matching packages by that name")
// 	case modCount == 1:
// 		choosen = &mods[0]
// 		prompt := promptui.Prompt{
// 			Label:     "Install " + choosen.Name,
// 			IsConfirm: true,
// 			Default:   "Y",
// 		}

// 		_, err := prompt.Run()
// 		if err != nil {
// 			logger.Info("Aborting installation")
// 			os.Exit(0)
// 		}
// 	default:
// 		task.Info("Found multiple packages by that name, please select one.")
// 		curse.SortByDownloadCount(mods)

// 		selectable := make([]string, modCount)
// 		for i, mod := range mods {
// 			selectable[i] = fmt.Sprintf("%s (%v)", mod.Name, HumanFloat32(mod.DownloadCount))
// 		}

// 		prompt := promptui.Select{
// 			Label: "Select Package",
// 			Items: selectable,
// 			Size:  8,
// 		}

// 		i, _, err := prompt.Run()
// 		if err != nil {
// 			fmt.Println("Aborting installation")
// 			os.Exit(0)
// 		}
// 		choosen = &mods[i]
// 	}
// 	return choosen
// }
