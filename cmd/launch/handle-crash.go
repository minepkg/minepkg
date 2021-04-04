package launch

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/instances"
)

// HandleCrash handles a crash by submitting it to minepkg.io and outputting some debug info
func (c *CLILauncher) HandleCrash() error {
	// exit code was not 130 or 0, we output error info and submit a crash report
	man := c.Instance.Manifest
	platform := man.PlatformString()

	packageName := man.Package.Name
	packageVersion := man.Package.Version
	if man.Package.BasedOn != "" {
		if len(man.Dependencies) != 1 {
			fmt.Println("Customized modpacks do not support crash reports for now. Skipping")
			return nil
		}
		// looks like an unmodified modpack instance. use that as name
		packageName = man.Package.BasedOn
		packageVersion = man.Dependencies[packageName]
	}

	fmt.Println("--------------------")
	fmt.Println("Minecraft crashed :(")
	fmt.Println("Here is some debug info")
	fmt.Println("[system]")
	fmt.Println("  OS: " + runtime.GOOS)
	fmt.Printf("  CPUs: %d\n", runtime.NumCPU())
	fmt.Println("[instance]")
	fmt.Printf("  package: %s@%s\n", packageName, packageVersion)
	fmt.Println("  platform: " + man.PlatformString())
	fmt.Println("  minecraft: " + man.Requirements.Minecraft)
	fmt.Println("[launch]")
	// TODO: print java path
	// fmt.Println("  java path: " + c.Instance.HasJava())
	fmt.Println("  minecraft version: " + c.Instance.Lockfile.MinecraftVersion())
	fmt.Println("  launch manifest: " + c.Instance.Lockfile.McManifestName())
	if platform == "fabric" {
		fmt.Printf(
			"  fabric: %s / %s (loader / mapping)\n",
			c.Instance.Lockfile.Fabric.FabricLoader,
			c.Instance.Lockfile.Fabric.Mapping,
		)
	}
	fmt.Printf("  exit code: %d\n", c.Cmd.ProcessState.ExitCode())

	fmt.Println("\nSubmitting crash report to minepkg.io …")

	mods := make(map[string]string)

	for _, dep := range c.Instance.Lockfile.Dependencies {
		mods[dep.Name] = dep.Version
	}

	// map darwin to macos
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "macos"
	}

	report := api.CrashReport{
		Package: api.CrashReportPackage{
			Platform: man.Package.Platform,
			Name:     packageName,
			Version:  packageVersion,
		},
		Server:           c.ServerMode,
		MinecraftVersion: c.Instance.Lockfile.MinecraftVersion(),
		Mods:             mods,
		OS:               osName,
		Arch:             runtime.GOARCH,
		ExitCode:         c.Cmd.ProcessState.ExitCode(),
	}

	logPath := filepath.Join(c.Instance.McDir(), "logs/latest.log")
	if log, err := ioutil.ReadFile(logPath); err == nil {
		report.Logs = string(log)
	}

	if c.Instance.Platform() == instances.PlatformFabric {
		report.Fabric = &api.CrashReportFabricDetail{
			Loader:  c.Instance.Lockfile.Fabric.FabricLoader,
			Mapping: c.Instance.Lockfile.Fabric.Mapping,
		}
	}

	err := c.Instance.MinepkgAPI.PostCrashReport(context.TODO(), &report)
	if err != nil {
		fmt.Println("Could not submit crash report:")
		fmt.Println(err)
	}

	// exit with special status code, so tools know that minecraft crashed
	// this is a "service is unavailable" error according to https://www.freebsd.org/cgi/man.cgi?query=sysexits
	os.Exit(69)
	return err
}
