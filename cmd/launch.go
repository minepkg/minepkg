package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fiws/minepkg/cmd/launch"
	"github.com/fiws/minepkg/internals/instances"
	"github.com/fiws/minepkg/pkg/api"
	"github.com/fiws/minepkg/pkg/manifest"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type launchCommandeer struct {
	cmd *cobra.Command

	version       string
	serverMode    bool
	useSystemJava bool
	debugMode     bool
	offlineMode   bool
	acceptEula    bool
	onlyPrepare   bool
	crashTest     bool

	overwrites *launch.OverwriteFlags
}

func init() {
	l := launchCommandeer{}
	l.cmd = &cobra.Command{
		Use:   "launch [modpack]",
		Short: "Launch a local or remote modpack.",
		Long: `If a modpack name or URL is supplied, that modpack will be launched.
	Alternativly: Can be used in directories containing a minepkg.toml manifest to launch that modpack.
		`,
		Aliases: []string{"run", "start", "play"},
		Args:    cobra.MaximumNArgs(1),
		Run:     l.run,
	}

	l.cmd.Flags().BoolVarP(&l.serverMode, "server", "s", false, "Start a server instead of a client")
	l.cmd.Flags().BoolVarP(&l.debugMode, "debug", "", false, "Do not start, just debug")
	l.cmd.Flags().BoolVarP(&l.offlineMode, "offline", "", false, "Start the server in offline mode (server only)")
	l.cmd.Flags().BoolVarP(&l.onlyPrepare, "only-prepare", "", false, "Only prepare, skip launching")
	l.cmd.Flags().BoolVarP(&l.crashTest, "crashtest", "", false, "Stop server after it's online (can be used for testing)")

	l.overwrites = launch.CmdOverwriteFlags(l.cmd)
	rootCmd.AddCommand(l.cmd)
}

func (l *launchCommandeer) run(cmd *cobra.Command, args []string) {
	var instance *instances.Instance
	var instanceDir string

	if len(args) == 0 {
		var err error
		wd, _ := os.Getwd()
		instance, err = instances.NewInstanceFromWd()
		instanceDir = wd

		if err != nil {
			logger.Fail(err.Error())
		}
		instance.MinepkgAPI = apiClient
		launch.ApplyInstanceOverwrites(instance, l.overwrites)
	} else {
		instance = instances.NewEmptyInstance()
		reqs := &api.RequirementQuery{
			Plattform: "fabric", // TODO: not static!
			Minecraft: "*",
			Version:   "latest", // TODO: get from id
		}
		if l.overwrites.McVersion != "" {
			reqs.Minecraft = l.overwrites.McVersion
		}

		release, err := apiClient.FindRelease(context.TODO(), args[0], reqs)
		if err != nil {
			logger.Fail(err.Error())
		}
		if release == nil {
			logger.Fail("No release found")
		}

		instanceDir = filepath.Join(instance.InstancesDir(), release.Package.Name+"_"+release.Package.Platform)
		os.MkdirAll(instanceDir, os.ModePerm)

		// set instance details
		instance.Manifest = manifest.NewInstanceLike(release.Manifest)

		instance.MinepkgAPI = apiClient
		instance.Directory = instanceDir

		// overwrite some instance launch options with flags
		launch.ApplyInstanceOverwrites(instance, l.overwrites)

		if l.overwrites.McVersion == "" {
			fmt.Println("mc * resolved to: " + release.LatestTestedMinecraftVersion())
			instance.Manifest.Requirements.Minecraft = release.LatestTestedMinecraftVersion()
		}

		// TODO: only show when there actually is a update. ask user?
		logger.Headline("Updating instance")
		// maybe not update requirements every time
		if err := instance.UpdateLockfileRequirements(context.TODO()); err != nil {
			logger.Fail(err.Error())
		}
		if err := instance.UpdateLockfileDependencies(context.TODO()); err != nil {
			logger.Fail(err.Error())
		}

		instance.SaveManifest()
		instance.SaveLockfile()
	}

	switch {
	case l.crashTest && l.serverMode != true:
		logger.Fail("Can only crashtest servers. append --server to crashtest")
	case instance.Manifest.PlatformString() == "forge":
		logger.Fail("Can not launch forge modpacks for now. Sorry.")
	}

	if viper.GetBool("useSystemJava") == true {
		instance.UseSystemJava()
	}

	// launch instance
	fmt.Printf("Launching %s\n", instance.Desc())
	fmt.Printf("Instance location: %s\n", instanceDir)

	// we need login credentials to launch the client
	// the server needs no creds
	if l.serverMode != true {
		creds, err := ensureMojangAuth()
		if err != nil {
			logger.Fail(err.Error())
		}
		instance.MojangCredentials = creds
	}

	cliLauncher := launch.CLILauncher{
		Instance:       instance,
		ServerMode:     l.serverMode,
		OfflineMode:    l.offlineMode,
		NonInteractive: viper.GetBool("nonInteractive"),
	}

	if err := cliLauncher.Prepare(); err != nil {
		logger.Fail(err.Error())
	}

	// build and add the local jar
	if instance.Manifest.Package.Type == manifest.TypeMod {
		build := instance.BuildMod()
		cmdTerminalOutput(build)
		build.Start()
		err := build.Wait()
		if err != nil {
			// TODO: output logs or something
			fmt.Println(err)
			logger.Fail("Build step failed. Aborting")
		}
		// copy jar
		jar, err := instance.FindModJar()
		if err != nil {
			logger.Fail(err.Error())
		}
		_, targetName := filepath.Split(jar)
		fmt.Printf("adding %s to temporary modpack\n", jar)
		// TODO: mod could already be there if it has a circular dependency
		err = CopyFile(jar, filepath.Join(instance.ModsDir(), targetName))
		if err != nil {
			logger.Fail(err.Error())
		}
	}

	launchManifest := cliLauncher.LaunchManifest

	if l.onlyPrepare == true {
		fmt.Println("Skipping launch as requested")
		os.Exit(0)
	}

	fmt.Println("\n== Launching Minecraft ==")
	opts := &instances.LaunchOptions{
		LaunchManifest: launchManifest,
		SkipDownload:   true,
		Server:         l.serverMode,
		Debug:          l.debugMode,
	}

	launchErr := make(chan error)
	crashErr := make(chan error)

	if l.crashTest == true {

		go func() {
			tries := 0

			// server takes at least 500ms to startup
			time.Sleep(500 * time.Millisecond)

			// try to connect every 3 seconds for 30 times (1.5 minutes to start server)
			for {
				tries++
				// TODO: no hardcoded port
				// 10 seconds to connect to socket (usually does not take that long)
				conn, err := net.DialTimeout("tcp", ":25565", time.Duration(10)*time.Second)

				// no error, close connection and send nil in err channel
				if err == nil {
					// sleeping to let the server finish some initial setup after port was opened
					// TODO: do not sleep, check if server responds here
					time.Sleep(3 * time.Second)
					defer conn.Close()
					crashErr <- nil
					break
				}

				// could not connect, can we try again? send error in channel otherwise
				if tries >= 30 {
					crashErr <- err
					break
				}

				// wait 3 seconds before retrying
				time.Sleep(3 * time.Second)
			}
		}()
	}

	go func() {
		// finally, start the instance
		if err := cliLauncher.Launch(opts); err != nil {
			launchErr <- err
		}
		launchErr <- nil
	}()

	stopAfterCrashtest := func() {
		p, err := os.FindProcess(cliLauncher.Cmd.Process.Pid)
		if err != nil {
			fmt.Println("Could not stop minecraft after crashtest. Its'probably already stopped … which is not good")
			os.Exit(1)
		}
		if err := p.Signal(syscall.SIGTERM); err != nil {
			p.Signal(syscall.SIGKILL)
		}

		select {
		case <-launchErr:
			return
		case <-time.After(5 * time.Second):
			fmt.Println("Timed out stopping minecraft. Killing it")
			if err := p.Signal(syscall.SIGKILL); err != nil {
				fmt.Println("Could not kill minecraft")
			}
		}
	}

	select {
	// normal launch & minecraft was stopped
	case err := <-launchErr:
		if err != nil {
			// TODO: this stops any defer from running !!!
			logger.Fail(err.Error())
		}
	// crashtest and we got a response from the crash go routine
	case err := <-crashErr:
		// stop the minecraft server, crashtest went well or timed out
		if err != nil {
			fmt.Printf("Crashtest: could not connect to server (%s)\n", err)
			stopAfterCrashtest()
			os.Exit(69)
		}
		fmt.Println("Crashtest went fine! Waiting for server to shut down")
		stopAfterCrashtest()
		// normal exit
	}
}
