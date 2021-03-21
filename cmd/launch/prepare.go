package launch

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fiws/minepkg/internals/downloadmgr"
	"github.com/fiws/minepkg/internals/instances"
	"github.com/fiws/minepkg/internals/minecraft"
	"github.com/spf13/viper"
)

// CLILauncher can launch minepkg instances with CLI output
type CLILauncher struct {
	// Instance is the minepkg instance to be launched
	Instance *instances.Instance

	Cmd *exec.Cmd
	// ServerMode indicated if this instance should be started as a server
	ServerMode bool
	// OfflineMode indicates if this server should be started in offline mode
	OfflineMode bool

	// LaunchManifest is a minecraft launcher manifest. it should be set after
	// calling `Prepare`
	LaunchManifest *minecraft.LaunchManifest

	// NonInteractive determines if fancy spinners or prompts should be displayed
	NonInteractive bool

	originalServerProps []byte
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

	if !instance.HasJava() {
		s.Update("Preparing launch – Downloading java")
		if err := instance.UpdateJava(); err != nil {
			return err
		}
	}

	// resolve requirements
	outdatedReqs, err := instance.AreRequirementsOutdated()
	if err != nil {
		return err
	}
	if outdatedReqs {
		s.Update("Preparing launch – Resolving Requirements")
		err := instance.UpdateLockfileRequirements(context.TODO())
		if err != nil {
			return err
		}
		instance.SaveLockfile()
	}

	// resolve dependencies
	outdatedDeps, err := instance.AreDependenciesOutdated()
	if err != nil {
		return err
	}
	// also update deps when reqs are outdated
	if outdatedReqs || outdatedDeps {
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

	if !serverMode {
		missingAssets, err := instance.FindMissingAssets(launchManifest)
		if err != nil {
			return err
		}

		for _, asset := range missingAssets {
			target := filepath.Join(instance.CacheDir, "assets/objects", asset.UnixPath())
			mgr.Add(downloadmgr.NewHTTPItem(asset.DownloadURL(), target))
		}
	}

	missingLibs, err := instance.FindMissingLibraries(launchManifest)
	if err != nil {
		return err
	}

	for _, lib := range missingLibs {
		target := filepath.Join(instance.CacheDir, "libraries", lib.Filepath())
		mgr.Add(downloadmgr.NewHTTPItem(lib.DownloadURL(), target))
	}

	if err = mgr.Start(context.TODO()); err != nil {
		return err
	}

	s.Update("Copying local saves (if any)")
	if err := instance.CopyLocalSaves(); err != nil {
		return err
	}

	s.Update("Downloading dependencies")
	if err := instance.EnsureDependencies(context.TODO()); err != nil {
		return err
	}

	s.Update("Copying overwrites")
	if err := instance.CopyOverwrites(); err != nil {
		return err
	}

	if serverMode {
		s.Update("Preparing server files")
		c.prepareServer()
		if c.OfflineMode {
			s.Update("Preparing for offline mode")
			c.prepareOfflineServer()
		}
	}

	return nil
}

func (c *CLILauncher) prepareServer() {
	c.LaunchManifest.MainClass = strings.Replace(c.LaunchManifest.MainClass, "Client", "Server", -1)
	instance := c.Instance

	// TODO: better handling
	if viper.GetBool("acceptMinecraftEula") {
		eula := "# accepted through minepkg\n# https://account.mojang.com/documents/minecraft_eula\neula=true\n"
		ioutil.WriteFile(filepath.Join(instance.McDir(), "./eula.txt"), []byte(eula), 0644)
	}
}

func (c *CLILauncher) prepareOfflineServer() {
	settingsFile := filepath.Join(c.Instance.McDir(), "server.properties")
	rawSettings, err := ioutil.ReadFile(settingsFile)

	// workarround to get server that was started in offline mode for the first time
	// to start in online mode next time it is launched
	if err != nil {
		rawSettings = []byte("online-mode=true\n")
	}

	// save the original settings here, so we can write them back after server stops (in Launch function)
	c.originalServerProps = rawSettings

	settings := minecraft.ParseServerProps(rawSettings)
	settings["online-mode"] = "false"

	// write modified config file
	if err := ioutil.WriteFile(settingsFile, []byte(settings.String()), 0644); err != nil {
		panic(err)
	}
}
