package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/briandowns/spinner"
	"github.com/minepkg/minepkg/internals/downloadmgr"
	"github.com/minepkg/minepkg/internals/instances"
)

// installManifest installs dependencies from the minepkg.toml
func installManifest(instance *instances.Instance) {
	cacheDir := filepath.Join(globalDir, "cache")

	task := logger.NewTask(2)

	task.Info("Installing minepkg.toml dependencies")
	s := spinner.New(spinner.CharSets[9], 300*time.Millisecond) // Build our new spinner
	s.Prefix = " "

	mgr := downloadmgr.New()
	mgr.OnProgress = func(p int) {
		s.Suffix = fmt.Sprintf(" Downloading %v", p) + "%"
	}

	task.Step("🔎", "Resolving Dependencies")
	err := instance.UpdateLockfileDependencies(context.TODO())
	if err != nil {
		logger.Fail(err.Error())
	}
	for _, dep := range instance.Lockfile.Dependencies {
		fmt.Printf(" - %s@%s\n", dep.Name, dep.Version)
	}
	missingFiles, err := instance.FindMissingDependencies()
	if err != nil {
		logger.Fail(err.Error())
	}

	task.Step("🚚", fmt.Sprintf("Downloading %d Packages", len(missingFiles)))
	for _, m := range missingFiles {
		fmt.Printf(" - %s@%s\n", m.Name, m.Version)
		p := filepath.Join(cacheDir, m.Name, m.Version+".jar")
		mgr.Add(downloadmgr.NewHTTPItem(m.URL, p))
	}

	s.Start()
	if err := mgr.Start(context.TODO()); err != nil {
		logger.Fail(err.Error())
	}

	instance.LinkDependencies()

	s.Stop()
	instance.SaveLockfile()
	fmt.Println("You can now launch Minecraft using \"minepkg launch\"")
}
