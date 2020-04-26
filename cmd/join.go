package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Tnze/go-mc/bot"
	"github.com/fiws/minepkg/cmd/launch"
	"github.com/fiws/minepkg/internals/instances"
	"github.com/fiws/minepkg/pkg/api"
	"github.com/fiws/minepkg/pkg/manifest"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(joinCmd)
}

// TODO: use launch logic (save setup)
var joinCmd = &cobra.Command{
	Use:     "join <ip/hostname>",
	Short:   "Joins a compatible server without any setup",
	Long:    `Servers have to be started with \"minepkg launch --server\" or include the minepkg-companion mod`,
	Example: `  minepkg join demoserver.minepkg.io`,
	Aliases: []string{"i-wanna-play-on", "connect"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		var resolvedModpack *api.Release
		ip := "127.0.0.1"
		host := args[0]

		if host != "localhost" {
			rawIP, err := net.LookupHost(host)
			if err != nil {
				logger.Fail("Could not resolve host " + host)
			}

			ip = strings.Join(rawIP, ".")
		}

		resolvedModpack = resolveViaSLP(ip)
		if resolvedModpack == nil {
			resolvedModpack = resolveFromAPI(ip)
		}

		if resolvedModpack == nil {
			logger.Info("Could not determine the server modpack")
			os.Exit(1)
		}

		// looks like we can join this server, so we start initializing the instance stuff here
		instance := instances.Instance{
			GlobalDir:  globalDir,
			Lockfile:   manifest.NewLockfile(),
			MinepkgAPI: apiClient,
		}

		instanceDir := filepath.Join(instance.InstancesDir(), "server."+ip+"."+resolvedModpack.Package.Name+"."+resolvedModpack.Package.Platform)
		os.MkdirAll(instanceDir, os.ModePerm)

		instance.ModsDirectory = filepath.Join(instanceDir, "mods")
		wd, err := os.Getwd()
		if err != nil {
			logger.Fail(err.Error())
		}
		// change dir to the instance
		if err := os.Chdir(instanceDir); err != nil {
			logger.Fail(err.Error())
		}

		defer os.Chdir(wd) // move back to working directory

		creds, err := ensureMojangAuth()
		if err != nil {
			logger.Fail(err.Error())
		}
		instance.MojangCredentials = creds

		instance.Manifest = resolvedModpack.Manifest
		fmt.Println("Using modpack " + resolvedModpack.Identifier())

		// force the latest requirements
		if err := instance.UpdateLockfileRequirements(context.TODO()); err != nil {
			logger.Fail(err.Error())
		}

		// force the latest dependencies
		if err := instance.UpdateLockfileDependencies(context.TODO()); err != nil {
			logger.Fail(err.Error())
		}

		instance.SaveLockfile()

		cliLauncher := launch.CLILauncher{Instance: &instance, ServerMode: serverMode}
		cliLauncher.Prepare()

		fmt.Println("\nLaunching Minecraft …")
		opts := &instances.LaunchOptions{
			JoinServer: ip,
		}
		err = cliLauncher.Launch(opts)
		if err != nil {
			logger.Fail(err.Error())
		}
	},
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

func resolveViaSLP(ip string) *api.Release {
	fmt.Println("Trying to query server modpack …")
	serverData, _, err := bot.PingAndListTimeout(ip, 25565, 10*time.Second)
	if err != nil {
		fmt.Println("could not reach server")
		return nil
	}

	data := slpData{}
	json.Unmarshal(serverData, &data)
	if data.MinepkgModpack == nil {
		fmt.Println("Server does not use minepkg-companion 0.2.0+ – Please upgrade the mod on the server")
		return nil
	}
	fmt.Println("minepkg compatible server detected. Modpack: " + data.MinepkgModpack.Name)
	reqs := &api.RequirementQuery{
		Version:   version,
		Plattform: data.MinepkgModpack.Platform,
	}
	release, err := apiClient.FindRelease(context.TODO(), data.MinepkgModpack.Name, reqs)
	if err != nil {
		return nil
	}

	return release
}

func resolveFromAPI(ip string) *api.Release {
	var server *MinepkgMapping
	req, _ := http.NewRequest("GET", "https://test-api.minepkg.io/v1/server-mappings/"+ip, nil)
	apiClient.DecorateRequest(req)
	res, err := apiClient.HTTP.Do(req)
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logger.Fail(err.Error())
	}
	if res.StatusCode == 404 {
		logger.Fail("Server not in minepkg.io database (the server has to be started with minepkg)")
	}
	if res.StatusCode != 200 {
		logger.Fail("minepkg.io server database not reachable")
	}
	json.Unmarshal(buf, &server)

	name, version := splitPackageName(server.Modpack)

	reqs := &api.RequirementQuery{
		Version:   version,
		Minecraft: "*",
		Plattform: server.Platform,
	}

	// TODO: get release instead of find
	release, err := apiClient.FindRelease(context.TODO(), name, reqs)
	if err != nil {
		return nil
	}
	return release
}
