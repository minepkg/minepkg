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

	"github.com/fiws/minepkg/cmd/launch"
	"github.com/fiws/minepkg/pkg/api"
	"github.com/fiws/minepkg/pkg/manifest"

	"github.com/fiws/minepkg/internals/instances"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(joinCmd)
}

// TODO: use launch logic (save setup)
var joinCmd = &cobra.Command{
	Use:     "join <ip/hostname>",
	Short:   "Joins a compatible server without any setup",
	Long:    `Servers have to be started with \"minepkg launch --server\" for this to work. (For now)`,
	Example: `  minepkg join demoserver.minepkg.io`,
	Aliases: []string{"i-wanna-play-on"},
	Run: func(cmd *cobra.Command, args []string) {
		tempDir, err := ioutil.TempDir("", args[0])
		wd, _ := os.Getwd()
		os.Chdir(tempDir) // change working directory to temporary dir

		defer os.RemoveAll(tempDir) // cleanup dir after minecraft is closed
		defer os.Chdir(wd)          // move back to working directory
		if err != nil {
			logger.Fail(err.Error())
		}
		instance := instances.Instance{
			GlobalDir:     globalDir,
			ModsDirectory: filepath.Join(tempDir, "mods"),
			Lockfile:      manifest.NewLockfile(),
			MinepkgAPI:    apiClient,
		}

		creds, err := ensureMojangAuth()
		if err != nil {
			logger.Fail(err.Error())
		}
		instance.MojangCredentials = creds.Mojang

		host := args[0]
		rawIP, err := net.LookupHost(host)
		if err != nil {
			logger.Fail("Could not resolve host " + host)
		}

		ip := strings.Join(rawIP, ".")

		var server *MinepkgMapping

		req, _ := http.NewRequest("GET", "https://test-api.minepkg.io/v1/server-mappings/"+ip, nil)
		apiClient.DecorateRequest(req)
		res, err := apiClient.HTTP.Do(req)
		buf, err := ioutil.ReadAll(res.Body)
		if err != nil {
			logger.Fail(err.Error())
		}
		if res.StatusCode == 404 {
			logger.Fail("Server not in minepkg.io database (the server has to be started with minepkg")
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
			logger.Fail(err.Error())
		}
		if release == nil {
			logger.Info("Could not find the server modpack \"" + server.Modpack + "\"")
			os.Exit(1)
		}

		instance.Manifest = release.Manifest
		fmt.Println("Creating temporary modpack with " + release.Identifier())

		if err := instance.UpdateLockfileRequirements(context.TODO()); err != nil {
			logger.Fail(err.Error())
		}
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
