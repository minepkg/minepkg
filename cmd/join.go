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
	"strconv"
	"strings"
	"time"

	"github.com/Tnze/go-mc/bot"
	"github.com/fiws/minepkg/cmd/launch"
	"github.com/fiws/minepkg/internals/api"
	"github.com/fiws/minepkg/internals/commands"
	"github.com/fiws/minepkg/internals/globals"
	"github.com/fiws/minepkg/internals/instances"
	"github.com/fiws/minepkg/pkg/manifest"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmd := commands.New(&cobra.Command{
		Use:     "join <ip/hostname>",
		Short:   "Joins a compatible server without any setup",
		Long:    `Servers have to be started with \"minepkg launch --server\" or include the minepkg-companion mod`,
		Example: `  minepkg join demo.minepkg.host`,
		Aliases: []string{"i-wanna-play-on", "connect"},
		Args:    cobra.ExactArgs(1),
	}, &joinRunner{})

	rootCmd.AddCommand(cmd.Command)
}

type joinRunner struct{}

func (i *joinRunner) RunE(cmd *cobra.Command, args []string) error {

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
		MinepkgAPI: globals.ApiClient,
	}

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

	defer os.Chdir(wd) // move back to working directory

	creds, err := ensureMojangAuth()
	if err != nil {
		return err
	}
	instance.MojangCredentials = creds

	instance.Manifest = resolvedModpack.Manifest
	fmt.Println("Using modpack " + resolvedModpack.Identifier())

	cliLauncher := launch.CLILauncher{Instance: &instance, ServerMode: false}
	cliLauncher.Prepare()

	if viper.GetBool("useSystemJava") {
		instance.UseSystemJava()
	}

	fmt.Println("\nLaunching Minecraft …")
	opts := &instances.LaunchOptions{
		JoinServer: ip + ":" + port,
	}
	err = cliLauncher.Launch(opts)
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
	fmt.Println("Trying to query server modpack …")
	intPort, _ := strconv.Atoi(port)
	serverData, _, err := bot.PingAndListTimeout(ip, intPort, 10*time.Second)
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
		Version:   data.MinepkgModpack.Version,
		Plattform: data.MinepkgModpack.Platform,
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

func resolveFromAPI(ip string) *api.Release {
	var server *MinepkgMapping
	req, _ := http.NewRequest("GET", "https://test-api.minepkg.io/v1/server-mappings/"+ip, nil)
	globals.ApiClient.DecorateRequest(req)
	res, err := globals.ApiClient.HTTP.Do(req)
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
	release, err := globals.ApiClient.FindRelease(context.TODO(), name, reqs)
	if err != nil {
		return nil
	}
	return release
}
