package launch

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fiws/minepkg/internals/instances"
	"github.com/fiws/minepkg/pkg/api"
)

// HandleCrash handles a crash by submitting it to minepkg.io and outputting some debug info
func (c *CLILauncher) HandleCrash() error {
	// exit code was not 130 or 0, we output error info and submit a crash report
	man := c.Instance.Manifest
	platform := man.PlatformString()

	fmt.Println("--------------------")
	fmt.Println("Minecraft crashed :(")
	fmt.Println("Here is some debug info")
	fmt.Println("[system]")
	fmt.Println("  OS: " + runtime.GOOS)
	fmt.Printf("  CPUs: %d\n", runtime.NumCPU())
	fmt.Println("[instance]")
	fmt.Printf("  package: %s@%s\n", man.Package.Name, man.Package.Version)
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

	packageName := man.Package.Name
	if strings.HasPrefix(packageName, "_instance") {
		// looks like an unmodified modpack instance. use that as name
		// TODO: check is not too percise
		if len(man.Dependencies) == 1 {
			// this basically is man.Depdendencies.GetFirstKey()
			// because Dependencies only has one key at this point
			for key := range man.Dependencies {
				packageName = key
			}
		} else {
			fmt.Println("Customized modpacks do not support crash reports for now. Skipping")
			return nil
		}
	}

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
			Version:  man.Package.Version,
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
