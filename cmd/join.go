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

	"github.com/fiws/minepkg/pkg/api"
	"github.com/fiws/minepkg/pkg/manifest"

	"github.com/briandowns/spinner"
	"github.com/fiws/minepkg/internals/downloadmgr"
	"github.com/fiws/minepkg/internals/instances"
	"github.com/spf13/cobra"
)

// TODO: use launch logic (save setup)
var joinCmd = &cobra.Command{
	Use:     "join <ip/hostname>",
	Short:   "Joins a compatible server without any setup.",
	Long:    `Servers have to be started with \"minepkg launch --server\" for this to work. (For now)`,
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
			Directory:     globalDir,
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

		// Prepare launch
		s := spinner.New(spinner.CharSets[9], 300*time.Millisecond) // Build our new spinner
		s.Prefix = " "
		s.Start()
		s.Suffix = " Preparing launch"

		java := javaBin(instance.Directory)
		if java == "" {
			s.Suffix = " Preparing launch – Downloading java"
			var err error
			java, err = downloadJava(instance.Directory)
			if err != nil {
				logger.Fail(err.Error())
			}
		}

		mgr := downloadmgr.New()
		mgr.OnProgress = func(p int) {
			s.Suffix = fmt.Sprintf(" Preparing launch – Downloading %v", p) + "%"
		}

		launchManifest, err := instance.GetLaunchManifest()
		if err != nil {
			logger.Fail(err.Error())
		}

		missingLibs, err := instance.FindMissingLibraries(launchManifest)
		if err != nil {
			logger.Fail(err.Error())
		}

		for _, lib := range missingLibs {
			target := filepath.Join(instance.Directory, "libraries", lib.Filepath())
			mgr.Add(downloadmgr.NewHTTPItem(lib.DownloadURL(), target))
		}

		if err = mgr.Start(context.TODO()); err != nil {
			logger.Fail(err.Error())
		}

		s.Suffix = " Downloading dependencies"
		if err := instance.EnsureDependencies(context.TODO()); err != nil {
			logger.Fail(err.Error())
		}

		s.Stop()

		fmt.Println("\nLaunching Minecraft …")
		opts := &instances.LaunchOptions{
			LaunchManifest: launchManifest,
			Java:           java,
			JoinServer:     ip,
		}
		err = instance.Launch(opts)
		if err != nil {
			logger.Fail(err.Error())
		}
	},
}
