package launch

import (
	"fmt"
	"os"
	"runtime"

	"github.com/fiws/minepkg/internals/instances"
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
	// exit with special status code, so tools know that minecraft crashed
	// this is a "service is unavailable" error according to https://www.freebsd.org/cgi/man.cgi?query=sysexits

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
	os.Exit(69)
	return err
}
