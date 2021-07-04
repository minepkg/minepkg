package launch

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jwalton/gchalk"
	"github.com/minepkg/minepkg/internals/downloadmgr"
	"github.com/minepkg/minepkg/internals/instances"
	"github.com/minepkg/minepkg/internals/java"
	"github.com/minepkg/minepkg/internals/minecraft"
	"github.com/spf13/viper"
)

// CLILauncher can launch minepkg instances with CLI output
type CLILauncher struct {
	// Instance is the minepkg instance to be launched
	Instance *instances.Instance

	// MinepkgVersion is the version number of minepkg
	MinepkgVersion string

	Cmd *exec.Cmd
	// ServerMode indicated if this instance should be started as a server
	ServerMode bool
	// OfflineMode indicates if this server should be started in offline mode
	OfflineMode bool

	// ForceUpdate will force a full dependency resolve if set to true
	ForceUpdate bool

	// LaunchManifest is a minecraft launcher manifest. it should be set after
	// calling `Prepare`
	LaunchManifest *minecraft.LaunchManifest

	// NonInteractive determines if fancy spinners or prompts should be displayed
	NonInteractive bool

	// UseSystemJava sets the instance to use the system java
	// instead of the internal installation. This skips downloading java
	UseSystemJava bool

	javaFactory         *java.Factory
	java                *java.Java
	introPrinted        bool
	originalServerProps []byte
}

// Prepare ensures all requirements are met to launch the
// instance in the current directory
func (c *CLILauncher) Prepare() error {
	instance := c.Instance
	serverMode := c.ServerMode

	ctx := context.Background()

	c.printIntro()
	c.introPrinted = true

	javaUpdate := make(chan error, 1)

	// resolve requirements
	outdatedReqs, err := instance.AreRequirementsOutdated()
	if err != nil {
		return err
	}

	fmt.Print(pipeText.Render(gchalk.BgGray("Requirements")))
	if c.ForceUpdate || outdatedReqs {
		fmt.Print(gchalk.Gray("(updating)"))
		err := instance.UpdateLockfileRequirements(context.TODO())
		if err != nil {
			return err
		}
		instance.SaveLockfile()
	}
	fmt.Println()

	if c.UseSystemJava {
		// nothing gets downloaded. this is a success
		javaUpdate <- nil
	} else {
		// we check if we need to download java
		java, err := instance.Java()
		if err != nil {
			return err
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

	req := gchalk.Gray(fmt.Sprintf(" resolved from %s", c.Instance.Manifest.Requirements.Minecraft))
	fmt.Println("│ Minecraft " + c.Instance.Lockfile.MinecraftVersion() + req)
	if instance.Manifest.PlatformString() == "fabric" {
		fmt.Printf(
			"│ Fabric: %s / %s (loader / mapping)\n",
			instance.Lockfile.Fabric.FabricLoader,
			instance.Lockfile.Fabric.Mapping,
		)
	}
	fmt.Println("│")

	// resolve dependencies
	outdatedDeps, err := instance.AreDependenciesOutdated()
	if err != nil {
		return err
	}

	// also update deps when reqs are outdated
	fmt.Print(pipeText.Render(gchalk.BgGray("Dependencies")))
	if c.ForceUpdate || outdatedReqs || outdatedDeps {
		fmt.Print(gchalk.Gray("(updating)\n"))
		if err := c.newFetchDependencies(ctx); err != nil {
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

	mgr := downloadmgr.New()

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

	if err := instance.CopyLocalSaves(); err != nil {
		return err
	}

	// TODO: still needed?
	if err := instance.EnsureDependencies(context.TODO()); err != nil {
		return err
	}

	if err := instance.CopyOverwrites(); err != nil {
		return err
	}

	if serverMode {
		fmt.Println(pipeText.Render("\nPreparing server"))
		c.prepareServer()
		if c.OfflineMode {
			pipeText.Render("  in offline mode")
			c.prepareOfflineServer()
		}
	}

	if err := <-javaUpdate; err != nil {
		return err
	}

	c.printOutro()

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

func (c *CLILauncher) printIntro() {
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

func (c *CLILauncher) printOutro() {
	instance := c.Instance

	javaDir := "(system java)"
	if !c.UseSystemJava {
		java, err := instance.Java()
		if err != nil {
			panic(err)
		}
		javaDir = java.Bin()
	}
	fmt.Println("│ minepkg " + c.MinepkgVersion)
	fmt.Println("│ Java " + javaDir)
}

func (c *CLILauncher) newFetchDependencies(ctx context.Context) error {
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
