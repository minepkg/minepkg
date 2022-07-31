package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Tnze/go-mc/bot"
	"github.com/erikgeiser/promptkit/confirmation"
	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/internals/globals"
	"github.com/minepkg/minepkg/internals/instances"
	"github.com/minepkg/minepkg/internals/launcher"
	"github.com/minepkg/minepkg/pkg/manifest"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	runner := &joinRunner{}
	cmd := commands.New(&cobra.Command{
		Use:     "join <ip/hostname>",
		Short:   "Joins a compatible server without any setup",
		Long:    `Servers have to be started with \"minepkg launch --server\" or include the minepkg-companion mod`,
		Example: `  minepkg join demo.minepkg.host`,
		Aliases: []string{"i-wanna-play-on", "connect"},
		Args:    cobra.ExactArgs(1),
	}, runner)

	cmd.Flags().IntVar(&runner.ramMiB, "ram", 0, "Overwrite the amount of RAM in MiB to use")
	cmd.Flags().BoolVar(&runner.clean, "clean", false, "Removes any instance data before launching")

	rootCmd.AddCommand(cmd.Command)
}

type joinRunner struct {
	ramMiB int
	clean  bool
}

func (j *joinRunner) RunE(cmd *cobra.Command, args []string) error {

	var resolvedModpack *api.Release
	ip := "127.0.0.1"
	connectionString := strings.Split(args[0], ":")
	host := connectionString[0]
	port := "25565"

	if len(connectionString) == 2 {
		port = connectionString[1]
	}

	if host != "localhost" {
		rawIP, err := net.LookupHost(host)
		if err != nil {
			logger.Fail("Could not resolve host " + host)
		}

		ip = strings.Join(rawIP, ".")
	}

	resolvedModpack = resolveViaSLP(ip, port)
	if resolvedModpack == nil {
		logger.Info("Could not determine the server modpack")
		os.Exit(1)
	}

	// looks like we can join this server, so we start initializing the instance stuff here
	instance := instances.New()
	instance.Lockfile = manifest.NewLockfile()
	instance.MinepkgAPI = globals.ApiClient

	instanceDir := filepath.Join(instance.InstancesDir(), "server."+ip+"."+resolvedModpack.Package.Name+"."+resolvedModpack.Package.Platform)
	os.MkdirAll(instanceDir, os.ModePerm)

	instance.Directory = instanceDir
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	// change dir to the instance
	if err := os.Chdir(instanceDir); err != nil {
		return err
	}

	if j.clean {
		if !root.NonInteractive {
			input := confirmation.New(
				"Removing ALL local data from this modpack. Continue?",
				confirmation.Yes,
			)
			overwrite, err := input.RunPrompt()
			if !overwrite || err != nil {
				logger.Info("Aborting")
				return nil
			}
		}
		err = instance.Clean()
		if err != nil {
			return err
		}
	}

	defer os.Chdir(wd) // move back to working directory

	creds, err := root.getLaunchCredentialsOrLogin()
	if err != nil {
		return err
	}
	instance.SetLaunchCredentials(creds)

	instance.Manifest = manifest.NewInstanceLike(resolvedModpack.Manifest)
	fmt.Println("Using modpack " + resolvedModpack.Identifier())

	cliLauncher := launcher.Launcher{
		Instance:       instance,
		ServerMode:     false,
		MinepkgVersion: rootCmd.Version,
		UseSystemJava:  viper.GetBool("useSystemJava"),
	}
	if err := cliLauncher.Prepare(); err != nil {
		return err
	}

	opts := &instances.LaunchOptions{
		JoinServer: ip + ":" + port,
		RamMiB:     j.ramMiB,
	}
	err = cliLauncher.Run(opts)
	if err != nil {
		return err
	}

	return nil
}

type modpackDescription struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Platform string `json:"platform"`
}

type slpData struct {
	MinepkgModpack *modpackDescription `json:"minepkgModpack"`
	Version        struct {
		Name     string `json:"name"`
		Protocol string `json:"protocol"`
	} `json:"version"`
}

func resolveViaSLP(ip string, port string) *api.Release {
	fmt.Println("Trying to connect to", fmt.Sprintf("%s:%s", ip, port))
	serverData, _, err := bot.PingAndListTimeout(fmt.Sprintf("%s:%s", ip, port), 10*time.Second)
	if err != nil {
		fmt.Println("could not reach server")
		return nil
	}

	data := slpData{}
	json.Unmarshal(serverData, &data)
	if data.MinepkgModpack == nil {
		fmt.Println("Server does not use minepkg-companion 0.2.0+ or has no valid modpack")
		return nil

	}

	if data.Version.Name == "" {
		logger.Fail("Server has no Minecraft version set. This usually means that the server is still starting up. Try again in a few seconds.")
	}

	fmt.Println("minepkg compatible server detected. Modpack: " + data.MinepkgModpack.Name)

	reqs := &api.RequirementQuery{
		Version:  data.MinepkgModpack.Version,
		Platform: data.MinepkgModpack.Platform,
		// raw version from minecraft slp.. might need to check that
		Minecraft: data.Version.Name,
	}
	release, err := globals.ApiClient.FindRelease(context.TODO(), data.MinepkgModpack.Name, reqs)
	if err != nil {
		logger.Fail("Could not fetch release: " + err.Error())
	}
	if release == nil {
		logger.Fail("Server modpack is not published on minepkg.io")
	}

	return release
}
