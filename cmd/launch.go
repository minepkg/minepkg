package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/erikgeiser/promptkit/confirmation"
	"github.com/jwalton/gchalk"
	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/internals/instances"
	"github.com/minepkg/minepkg/internals/launcher"
	"github.com/minepkg/minepkg/internals/patch"
	"github.com/minepkg/minepkg/pkg/manifest"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	runner := &launchRunner{}

	cmd := commands.New(&cobra.Command{
		Use:   "launch [modpack]",
		Short: "Launch the given or local modpack.",
		Long: `If a modpack name or URL is supplied, that modpack will be launched.
Alternatively: Can be used in directories containing a minepkg.toml manifest to launch that modpack.
		`,
		Aliases: []string{"run", "start", "play"},
		Args:    cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			// do not complete if we have an argument
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return root.AutoCompleter.CompleteModpacks(toComplete)
		},
	}, runner)

	cmd.Flags().BoolVarP(&runner.serverMode, "server", "s", false, "Start a server instead of a client")
	cmd.Flags().BoolVarP(&runner.forceUpdate, "update", "u", false, "Force check for updates before starting")
	cmd.Flags().BoolVar(&runner.debugMode, "debug", false, "Do not start, just debug")
	cmd.Flags().BoolVar(&runner.offlineMode, "offline", false, "Start the server in offline mode (server only)")
	cmd.Flags().BoolVar(&runner.onlyPrepare, "only-prepare", false, "Only prepare, skip launching")
	cmd.Flags().BoolVar(&runner.crashTest, "crashtest", false, "Stop server after it's online (can be used for testing)")
	cmd.Flags().BoolVar(&runner.noBuild, "no-build", false, "Skip build (if any)")
	cmd.Flags().BoolVar(&runner.clean, "clean", false, "Removes any instance data except for savegames")
	cmd.Flags().StringArrayVar(&runner.patch, "patch", runner.patch, "Apply a patch to the instance before launching")
	runner.overwrites = launcher.CmdOverwriteFlags(cmd.Command)

	rootCmd.AddCommand(cmd.Command)
}

type launchRunner struct {
	serverMode  bool
	debugMode   bool
	offlineMode bool
	onlyPrepare bool
	crashTest   bool
	noBuild     bool
	forceUpdate bool
	clean       bool
	patch       []string

	overwrites *launcher.OverwriteFlags

	instance *instances.Instance
}

var (
	errCanOnlyLaunchModpacks = &commands.CliError{
		Text: "can only launch modpacks",
		Suggestions: []string{
			fmt.Sprintf("use %s to test mods", gchalk.Bold("minepkg try <modname>")),
		},
	}
	vanillaManifest = manifest.New()
)

func (l *launchRunner) RunE(cmd *cobra.Command, args []string) error {
	var err error

	vanillaManifest.Requirements.Minecraft = "*"
	vanillaManifest.Requirements.MinepkgCompanion = "none"

	if len(args) == 0 {
		log.Println("no modpack supplied, trying to launch local modpack")
		l.instance, err = root.LocalInstance()
		if err != nil {
			return err
		}
		// we validate the local manifest
		if err := root.validateManifest(l.instance.Manifest); err != nil {
			return err
		}
	} else {
		if args[0] == "vanilla" {
			log.Println("launching vanilla")
			l.instance = instances.New()
			l.instance.Manifest = vanillaManifest
			l.instance.Directory = filepath.Join(l.instance.InstancesDir(), "vanilla")

		} else {
			log.Println("launching online modpack")
			l.instance, err = l.instanceFromModpack(args[0])
		}
		if err != nil {
			return err
		}
	}

	switch {
	case l.crashTest && !l.serverMode:
		logger.Fail("Can only crashtest servers. append --server to crashtest")
	case l.instance.Manifest.PlatformString() == "forge":
		logger.Fail("Can not launch forge modpacks for now. Sorry.")
	}

	// we need login credentials to launch the client
	// the server needs no creds
	if !l.serverMode {
		creds, err := root.getLaunchCredentialsOrLogin()
		if err != nil {
			return err
		}
		l.instance.SetLaunchCredentials(creds)
	}

	if l.clean {
		if !root.NonInteractive {
			input := confirmation.New(
				"Removing ALL local data from this modpack except savegames. Continue?",
				confirmation.Yes,
			)
			overwrite, err := input.RunPrompt()
			if !overwrite || err != nil {
				logger.Info("Aborting")
				return nil
			}
		}
		err = l.instance.Clean()
		if err != nil {
			return err
		}
	}

	cliLauncher := launcher.Launcher{
		Instance:       l.instance,
		ServerMode:     l.serverMode,
		OfflineMode:    l.offlineMode,
		ForceUpdate:    l.forceUpdate,
		MinepkgVersion: rootCmd.Version,
		NonInteractive: viper.GetBool("nonInteractive"),
		UseSystemJava:  viper.GetBool("useSystemJava"),
	}

	cliLauncher.ApplyOverWrites(l.overwrites)

	// 1 minute timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// fetch patches
	patches := make([]*patch.Patch, len(l.patch))
	for i, patchLocation := range l.patch {
		log.Println("Fetching patch", patchLocation)
		p, err := patch.FetchPatch(ctx, patchLocation)
		if err != nil {
			return fmt.Errorf("could not load patch: %w", err)
		}
		patches[i] = p
	}

	cliLauncher.Patches = patches

	if err := cliLauncher.Prepare(); err != nil {
		return err
	}

	log.Println("Preparation done")

	if len(args) != 0 {
		if err := l.instance.SaveManifest(); err != nil {
			return err
		}
	}

	// build and add the local jar
	if l.instance.Manifest.Package.Type == manifest.TypeMod {
		if err := l.buildMod(); err != nil {
			return err
		}
	}

	launchManifest := cliLauncher.LaunchManifest

	if l.onlyPrepare {
		fmt.Println("Skipping launch as requested")
		os.Exit(0)
	}

	opts := &instances.LaunchOptions{
		LaunchManifest: launchManifest,
		Server:         l.serverMode,
		Debug:          l.debugMode,
		RamMiB:         l.overwrites.Ram,
	}

	launchErr := make(chan error)
	crashErr := make(chan error)

	if l.crashTest {
		go func() {
			crashErr <- crashTest()
		}()
	}

	go func() {
		// finally, start the instance
		launchErr <- cliLauncher.Run(opts)
	}()

	stopAfterCrashtest := func() {
		p, err := os.FindProcess(cliLauncher.Cmd.Process.Pid)
		if err != nil {
			fmt.Println("Could not stop minecraft after crashtest. It's probably already stopped … which is not good")
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
			// was this a crash test and minecraft just stopped?
			return err
		}
		if l.crashTest {
			fmt.Println("Crashtest: Minecraft unexpectedly stopped before we could connect to it")
			os.Exit(69)
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

	return nil
}

func crashTest() error {
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
			return nil
		}

		// could not connect, can we try again? send error in channel otherwise
		if tries >= 30 {
			return err
		}

		// wait up to 15 seconds before retrying
		wait := math.Min(15, float64(3*tries))
		time.Sleep(time.Duration(wait) * time.Second)
	}
}

func (l *launchRunner) instanceFromModpack(modpack string) (*instances.Instance, error) {
	apiClient := root.MinepkgAPI

	instance := instances.New()
	instance.MinepkgAPI = apiClient
	instance.ProviderStore = root.ProviderStore

	// fetch modpack
	query := &api.ReleasesQuery{
		Platform:     "fabric", // TODO: not static!
		Name:         modpack,
		VersionRange: "*", // TODO: get from id
	}
	if l.overwrites.McVersion != "" {
		query.Minecraft = l.overwrites.McVersion
	}

	release, err := apiClient.ReleasesQuery(context.TODO(), query)
	if err != nil {
		return nil, l.formatApiError(err)
	}

	if release.Package.Type == "mod" {
		return nil, errCanOnlyLaunchModpacks
	}

	// set instance details
	instance.Manifest = manifest.NewInstanceLike(release.Manifest)
	instance.Directory = filepath.Join(instance.InstancesDir(), release.Package.Name+"_"+release.Package.Platform)

	if l.overwrites.McVersion == "" {
		fmt.Println("Minecraft version '*' resolved to: " + release.LatestTestedMinecraftVersion())
		instance.Manifest.Requirements.Minecraft = release.LatestTestedMinecraftVersion()
	}

	return instance, nil
}

func (l *launchRunner) buildMod() error {
	if !l.noBuild {
		build := l.instance.BuildMod()
		cmdTerminalOutput(build)
		build.Start()
		err := build.Wait()
		if err != nil {
			// TODO: output logs or something
			return fmt.Errorf("build step failed: %w", err)
		}
	}

	// copy jar
	jar, err := getJarFileForInstance(l.instance)
	if err != nil {
		return err
	}

	fmt.Printf(" ► adding %s to temporary modpack\n", jar.Name())
	// TODO: mod could already be there if it has a circular dependency
	return CopyFile(jar.Path(), filepath.Join(l.instance.ModsDir(), jar.Name()))
}

func (l *launchRunner) formatApiError(err error) error {
	var notFoundErr *api.ErrNoMatchingRelease
	if errors.As(err, &notFoundErr) {
		switch notFoundErr.Err {
		case api.ErrProjectDoesNotExist:
			return &commands.CliError{
				Text: fmt.Sprintf("%s does not exist", notFoundErr.Package),
				Suggestions: []string{
					"Check if your have a typo in the packagename",
					"Make sure the wanted package is published",
				},
			}
		case api.ErrNoReleasesForPlatform:
			return &commands.CliError{
				Text: fmt.Sprintf(
					"%s has no releases for %s",
					notFoundErr.Package,
					notFoundErr.Requirements.Platform,
				),
			}
		case api.ErrNoReleaseForMinecraftVersion:
			return &commands.CliError{
				Text: fmt.Sprintf(
					"%s is not compatible with Minecraft %s",
					notFoundErr.Package,
					notFoundErr.Requirements.Minecraft,
				),
				Suggestions: []string{
					fmt.Sprintf("Supply a different minecraft version with %s", gchalk.Bold("--minecraft")),
					fmt.Sprintf("Wait until the author publishes a release for Minecraft %s", notFoundErr.Requirements.Minecraft),
				},
			}
		case api.ErrNoReleaseForVersion:
			return &commands.CliError{
				Text: fmt.Sprintf(
					"%s with version requirement %s not found",
					notFoundErr.Package,
					notFoundErr.Requirements.Version,
				),
				Suggestions: []string{
					"Use a different version of this package",
				},
			}
		case api.ErrNoReleaseForVersion:
			return &commands.CliError{
				Text: fmt.Sprintf(
					"%s is not compatible with the current requirements",
					notFoundErr.Package,
				),
				Suggestions: []string{
					"Use a different version of this package",
					fmt.Sprintf("Supply a different minecraft version with %s", gchalk.Bold("--minecraft")),
				},
			}
		}
	}
	return err
}
