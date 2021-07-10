package launcher

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jwalton/gchalk"
	"github.com/minepkg/minepkg/internals/downloadmgr"
	"github.com/minepkg/minepkg/internals/minecraft"
	"github.com/spf13/viper"
)

// Prepare ensures all requirements are met to launch the
// instance in the current directory
func (l *Launcher) Prepare() error {
	instance := l.Instance
	ctx := context.Background()

	l.printIntro()
	l.introPrinted = true

	// update requirements if needed
	outdatedReqs, err := l.prepareRequirements()
	if err != nil {
		return err
	}

	// update java in the background if needed
	javaUpdate := l.prepareJavaBg(ctx)

	// update dependencies
	if err := l.prepareDependencies(ctx, outdatedReqs); err != nil {
		return err
	}

	// download minecraft (assets, libraries, main jar etc) if needed
	if err := l.prepareMinecraft(ctx); err != nil {
		return err
	}

	if err := instance.CopyLocalSaves(); err != nil {
		return err
	}

	if err := instance.EnsureDependencies(ctx); err != nil {
		return err
	}

	if err := instance.CopyOverwrites(); err != nil {
		return err
	}

	if l.ServerMode {
		fmt.Println(pipeText.Render("\nPreparing server"))
		l.prepareServer()
		if l.OfflineMode {
			pipeText.Render("  in offline mode")
			l.prepareOfflineServer()
		}
	}

	if err := <-javaUpdate; err != nil {
		return err
	}

	l.printOutro()

	return nil
}

// prepareRequirements will update the requirements section
// in the lockfile if needed
func (l Launcher) prepareRequirements() (bool, error) {
	instance := l.Instance
	// resolve requirements
	outdatedReqs, err := instance.AreRequirementsOutdated()
	if err != nil {
		return false, err
	}

	fmt.Print(pipeText.Render(gchalk.BgGray("Requirements")))
	if l.ForceUpdate || outdatedReqs {
		fmt.Print(gchalk.Gray("(updating)"))
		err := instance.UpdateLockfileRequirements(context.TODO())
		if err != nil {
			return false, err
		}
		instance.SaveLockfile()
	}
	fmt.Println()

	req := gchalk.Gray(fmt.Sprintf(" resolved from %s", instance.Manifest.Requirements.Minecraft))
	fmt.Println("│ Minecraft " + instance.Lockfile.MinecraftVersion() + req)
	if instance.Manifest.PlatformString() == "fabric" {
		fmt.Printf(
			"│ Fabric: %s / %s (loader / mapping)\n",
			instance.Lockfile.Fabric.FabricLoader,
			instance.Lockfile.Fabric.Mapping,
		)
	}
	fmt.Println("│")
	return outdatedReqs, nil
}

// prepareJava downloads java if needed and returns an error channel
func (l *Launcher) prepareJavaBg(ctx context.Context) chan error {
	javaUpdate := make(chan error, 1)
	if l.UseSystemJava {
		// nothing gets downloaded. this is a success
		javaUpdate <- nil
	} else {
		// we check if we need to download java
		java, err := l.Java(ctx)
		if err != nil {
			javaUpdate <- err
		}

		if java.NeedsDownloading() {
			fmt.Printf("│ %s\n", gchalk.Gray("[i] Starting Java download …"))
			go func() {
				javaUpdate <- java.Update(ctx)
			}()
		} else {
			// nothing to download
			javaUpdate <- nil
		}
	}
	return javaUpdate
}

// prepareDependencies downloads missing dependencies if needed
// passing true as the second parameter will make sure to check for available updates
func (l *Launcher) prepareDependencies(ctx context.Context, force bool) error {
	instance := l.Instance
	// resolve dependencies
	outdatedDependencies, err := instance.AreDependenciesOutdated()
	if err != nil {
		return err
	}

	// also update dependencies when requirements are outdated
	fmt.Print(pipeText.Render(gchalk.BgGray("Dependencies")))
	if force || l.ForceUpdate || outdatedDependencies {
		fmt.Print(gchalk.Gray("(updating)\n"))
		if err := l.fetchDependencies(ctx); err != nil {
			return err
		}
		instance.SaveLockfile()
	} else {
		fmt.Println()
		for _, dependency := range instance.Lockfile.Dependencies {
			fmt.Println(dependencyLine(dependency))
		}
	}
	fmt.Println("│")
	return nil
}

func (l *Launcher) prepareMinecraft(ctx context.Context) error {
	instance := l.Instance
	mgr := downloadmgr.New()

	launchManifest, err := instance.GetLaunchManifest()
	if err != nil {
		return err
	}
	l.LaunchManifest = launchManifest

	// check for JAR
	// TODO move more logic to internals
	mainJar := filepath.Join(l.Instance.VersionsDir(), l.LaunchManifest.MinecraftVersion(), l.LaunchManifest.JarName())
	if _, err := os.Stat(mainJar); os.IsNotExist(err) {
		mgr.Add(downloadmgr.NewHTTPItem(l.LaunchManifest.Downloads.Client.URL, mainJar))
	}

	if !l.ServerMode {
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

	if err = mgr.Start(ctx); err != nil {
		return err
	}
	return nil
}

func (c *Launcher) prepareServer() {
	c.LaunchManifest.MainClass = strings.Replace(c.LaunchManifest.MainClass, "Client", "Server", -1)
	instance := c.Instance

	// TODO: better handling
	if viper.GetBool("acceptMinecraftEula") {
		eula := "# accepted through minepkg\n# https://account.mojang.com/documents/minecraft_eula\neula=true\n"
		ioutil.WriteFile(filepath.Join(instance.McDir(), "./eula.txt"), []byte(eula), 0644)
	}
}

func (c *Launcher) prepareOfflineServer() {
	settingsFile := filepath.Join(c.Instance.McDir(), "server.properties")
	rawSettings, err := ioutil.ReadFile(settingsFile)

	// workaround to get server that was started in offline mode for the first time
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

var pipeText = lipgloss.NewStyle().
	Border(lipgloss.Border{Left: "│"}, false).
	BorderLeft(true).
	Padding(0, 1)

func (c *Launcher) printIntro() {
	title := lipgloss.NewStyle().
		Border(lipgloss.Border{Left: "┃"}, false).
		BorderLeft(true).
		Background(lipgloss.Color("#FFF")).
		Foreground(lipgloss.Color("#000")).
		Padding(0, 1).
		Render(c.Instance.Manifest.Package.Name)

	fmt.Println(title)
	fmt.Println("│")
	fmt.Println("│ Directory: " + c.Instance.Directory)
}

func (l *Launcher) printOutro() {
	javaDir := "(system java)"
	if !l.UseSystemJava {
		javaDir = l.java.Bin()
	}
	fmt.Println("│ minepkg " + l.MinepkgVersion)
	fmt.Println("│ Java " + javaDir)
}

func (c *Launcher) fetchDependencies(ctx context.Context) error {
	instance := c.Instance

	resolver, err := instance.GetResolver(ctx)
	if err != nil {
		return err
	}

	sub := resolver.Subscribe()
	resolverErrorC := make(chan error)
	go func() {
		resolverErrorC <- resolver.Resolve(ctx)
	}()

	for resolved := range sub {
		instance.Lockfile.AddDependency(resolved.Lock())
		fmt.Println(dependencyLine(resolved.Lock()))
	}

	if err := <-resolverErrorC; err != nil {
		return err
	}

	// TODO: print stats or something

	return nil
}
