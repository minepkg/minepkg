package launch

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/fiws/minepkg/internals/instances"
	"github.com/fiws/minepkg/pkg/api"
)

// Launch will launch the instance with the provided launchOptions
// and will set some fallback values
func (c *CLILauncher) Launch(opts *instances.LaunchOptions) error {
	switch {
	case opts.LaunchManifest == nil:
		opts.LaunchManifest = c.LaunchManifest
	case opts.Server == false:
		opts.Server = c.ServerMode
	}
	err := c.Instance.Launch(opts)
	if err == nil {
		return nil
	}

	platform := c.Instance.Manifest.PlatformString()

	fmt.Println("--------------------")
	fmt.Println("Minecraft crashed :(")
	fmt.Println("Here is some debug info")
	fmt.Println("[system]")
	fmt.Println("  OS: " + runtime.GOOS)
	fmt.Printf("  CPUs: %d\n", runtime.NumCPU())
	fmt.Println("[instance]")
	fmt.Printf("  package: %s@%s\n", c.Instance.Manifest.Package.Name, c.Instance.Manifest.Package.Version)
	fmt.Println("  platform: " + c.Instance.Manifest.PlatformString())
	fmt.Println("  minecraft: " + c.Instance.Manifest.Requirements.Minecraft)
	fmt.Println("[launch]")
	fmt.Println("  Java Path: " + opts.Java)
	fmt.Println("  minecraft version: " + c.Instance.Lockfile.MinecraftVersion())
	fmt.Println("  launch manifest: " + c.Instance.Lockfile.McManifestName())
	if platform == "fabric" {
		fmt.Printf(
			"  fabric: %s / %s (loader / mapping)\n",
			c.Instance.Lockfile.Fabric.FabricLoader,
			c.Instance.Lockfile.Fabric.Mapping,
		)
	}

	fmt.Println("\nSubmitting crash report to minepkg.io …")

	mods := make(map[string]string)

	for _, dep := range c.Instance.Lockfile.Dependencies {
		mods[dep.Project] = dep.Version
	}

	report := api.CrashReport{
		Package: api.CrashReportPackage{
			Platform: c.Instance.Manifest.Package.Platform,
			Name:     c.Instance.Manifest.Package.Name,
			Version:  c.Instance.Manifest.Package.Version,
		},
		Server:           c.ServerMode,
		MinecraftVersion: c.Instance.Lockfile.MinecraftVersion(),
		Mods:             mods,
		OS:               runtime.GOOS,
		Arch:             runtime.GOARCH,
		// TODO: this could be a lie
		ExitCode: 1,
	}

	if log, err := ioutil.ReadFile("./logs/latest.log"); err == nil {
		report.Logs = string(log)
	}

	if c.Instance.Platform() == instances.PlatformFabric {
		report.Fabric = &api.CrashReportFabricDetail{
			Loader:  c.Instance.Lockfile.Fabric.FabricLoader,
			Mapping: c.Instance.Lockfile.Fabric.Mapping,
		}
	}

	err = c.Instance.MinepkgAPI.PostCrashReport(context.TODO(), &report)
	if err != nil {
		fmt.Println(err)
	}

	// exit with special status code, so tools know that minecraft crashed
	// this is a "service is unavailable" error according to https://www.freebsd.org/cgi/man.cgi?query=sysexits

	os.Exit(69)
	return err
}
