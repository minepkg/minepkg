package launch

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fiws/minepkg/internals/downloadmgr"
	"github.com/fiws/minepkg/internals/instances"
	"github.com/fiws/minepkg/internals/minecraft"
)

// CLILauncher can launch minepkg instances with CLI output
type CLILauncher struct {
	// Instance is the minepkg instance to be launched
	Instance *instances.Instance
	// ServerMode indicated if this instance should be started as a server
	ServerMode bool

	// LaunchManifest is a minecraft launcher manifest. it should be set after
	// calling `Prepare`
	LaunchManifest *minecraft.LaunchManifest
}

// Prepare ensures all requirements are met to launch the
// instance in the current directory
func (c *CLILauncher) Prepare() error {
	instance := c.Instance
	serverMode := c.ServerMode
	// Prepare launch
	s := spinner.New(spinner.CharSets[9], 300*time.Millisecond) // Build our new spinner
	s.Prefix = " "
	s.Start()
	s.Suffix = " Preparing launch"

	if instance.HasJava() == false {
		s.Suffix = " Preparing launch – Downloading java"
		if err := instance.UpdateJava(); err != nil {
			return err
		}
	}

	// resolve requirements
	if instance.Lockfile == nil || instance.Lockfile.HasRequirements() == false {
		s.Suffix = " Preparing launch – Resolving Requirements"
		instance.UpdateLockfileRequirements(context.TODO())
		instance.SaveLockfile()
	}

	mgr := downloadmgr.New()
	mgr.OnProgress = func(p int) {
		s.Suffix = fmt.Sprintf(" Preparing launch – Downloading %v", p) + "%"
	}

	launchManifest, err := instance.GetLaunchManifest()
	if err != nil {
		return err
	}
	c.LaunchManifest = launchManifest

	if serverMode != true {
		missingAssets, err := instance.FindMissingAssets(launchManifest)
		if err != nil {
			return err
		}

		for _, asset := range missingAssets {
			target := filepath.Join(instance.GlobalDir, "assets/objects", asset.UnixPath())
			mgr.Add(downloadmgr.NewHTTPItem(asset.DownloadURL(), target))
		}
	}

	missingLibs, err := instance.FindMissingLibraries(launchManifest)
	if err != nil {
		return err
	}

	for _, lib := range missingLibs {
		target := filepath.Join(instance.GlobalDir, "libraries", lib.Filepath())
		mgr.Add(downloadmgr.NewHTTPItem(lib.DownloadURL(), target))
	}

	if err = mgr.Start(context.TODO()); err != nil {
		return err
	}

	s.Suffix = " Downloading dependencies"
	if err := instance.EnsureDependencies(context.TODO()); err != nil {
		return err
	}

	s.Stop()
	return nil
}
