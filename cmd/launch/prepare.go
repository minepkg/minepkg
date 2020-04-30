package launch

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fiws/minepkg/internals/downloadmgr"
	"github.com/fiws/minepkg/internals/instances"
	"github.com/fiws/minepkg/internals/minecraft"
)

// CLILauncher can launch minepkg instances with CLI output
type CLILauncher struct {
	// Instance is the minepkg instance to be launched
	Instance *instances.Instance

	Cmd *exec.Cmd
	// ServerMode indicated if this instance should be started as a server
	ServerMode bool

	// LaunchManifest is a minecraft launcher manifest. it should be set after
	// calling `Prepare`
	LaunchManifest *minecraft.LaunchManifest

	// NonInteractive determines if fancy spinners or prompts should be displayed
	NonInteractive bool
}

// Prepare ensures all requirements are met to launch the
// instance in the current directory
func (c *CLILauncher) Prepare() error {
	instance := c.Instance
	serverMode := c.ServerMode
	// Prepare launch
	s := NewMaybeSpinner(!c.NonInteractive) // Build our new spinner
	s.Start()
	defer s.Stop()
	s.Update("Preparing launch")

	if instance.HasJava() == false {
		s.Update("Preparing launch – Downloading java")
		if err := instance.UpdateJava(); err != nil {
			return err
		}
	}

	// resolve requirements
	if instance.Lockfile == nil || instance.Lockfile.HasRequirements() == false {
		s.Update("Preparing launch – Resolving Requirements")
		err := instance.UpdateLockfileRequirements(context.TODO())
		if err != nil {
			return err
		}
		instance.SaveLockfile()
	}

	// resolve dependencies
	// TODO: len check does not account for same number but different mods
	if len(instance.Manifest.Dependencies) != len(instance.Lockfile.Dependencies) {
		s.Update("Preparing launch – Resolving Dependencies")
		err := instance.UpdateLockfileDependencies(context.TODO())
		if err != nil {
			return err
		}
		instance.SaveLockfile()
	}

	mgr := downloadmgr.New()
	mgr.OnProgress = func(p int) {
		s.Update(fmt.Sprintf("Preparing launch – Downloading %v", p) + "%")
	}

	launchManifest, err := instance.GetLaunchManifest()
	if err != nil {
		return err
	}
	c.LaunchManifest = launchManifest

	// check for JAR
	// TODO move more logic to internals
	mainJar := filepath.Join(c.Instance.VersionsDir(), c.LaunchManifest.MinecraftVersion(), c.LaunchManifest.JarName())
	if _, err := os.Stat(mainJar); os.IsNotExist(err) {
		mgr.Add(downloadmgr.NewHTTPItem(c.LaunchManifest.Downloads.Client.URL, mainJar))
	}

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

	s.Update("Downloading dependencies")
	if err := instance.EnsureDependencies(context.TODO()); err != nil {
		return err
	}

	return nil
}
