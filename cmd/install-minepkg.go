package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
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

type cache struct {
	baseDir string
}

func (c *cache) store() {

}

func installFromMinepkg(mods []string, instance *instances.McInstance) error {

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

	for i, name := range mods {
		comp := strings.Split(name, "@")
		name = comp[0]
		version := "latest"
		if len(comp) == 2 {
			version = comp[1]
		}
		release, err := apiClient.FindRelease(context.TODO(), name, version)
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
		logger.Info("Installing " + releases[0].Project + "@" + releases[0].Version)
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
	res := api.NewResolver(apiClient)
	res.Resolve(releases)
	for _, release := range releases {
		instance.Manifest.AddDependency(release.Project, release.Version)
		target := filepath.Join(cacheDir, release.Filename())
		mgr.Add(downloadmgr.NewHTTPItem(release.DownloadURL(), target))
	}

	// logger.Info("The following Dependencies will be downloaded:")
	// logger.Info(strings.Join())
	task.Step("🚚", "Downloading Packages")

	files, err := ioutil.ReadDir(instance.ModsDirectory)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		switch mode := f.Mode(); {
		case mode.IsRegular():
			logger.Warn("ignoring file in mods not placed by minepkg: " + f.Name())
		case mode&os.ModeSymlink != 0:
			os.Remove(filepath.Join(instance.ModsDirectory, f.Name()))
		case mode&os.ModeNamedPipe != 0:
			fmt.Println("named pipe?! what is this")
		}
	}

	s.Start()
	if err := mgr.Start(context.TODO()); err != nil {
		logger.Fail(err.Error())
	}

	for _, release := range res.Resolved {
		from := filepath.Join(cacheDir, release.Filename())
		to := filepath.Join(instance.ModsDirectory, release.Filename())
		err := os.Symlink(from, to)
		if err != nil {
			panic(err)
		}
	}
	s.Stop()
	instance.Manifest.Save()
	fmt.Println("updated minepkg.toml")

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
