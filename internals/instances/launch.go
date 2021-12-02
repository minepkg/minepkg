package instances

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/minepkg/minepkg/internals/minecraft"
	"github.com/minepkg/minepkg/internals/mojang"
	"github.com/minepkg/minepkg/pkg/manifest"
	"github.com/pbnjay/memory"
)

var (
	// ErrLaunchNotImplemented is returned if attemting to start a non vanilla instance
	ErrLaunchNotImplemented = errors.New("can only launch vanilla & fabric instances (for now)")
	// ErrNoCredentials is returned when an instance is launched without `MojangProfile` beeing set
	ErrNoCredentials = errors.New("can not launch without mojang credentials")
	// ErrNoPaidAccount is returned when an instance is launched without `MojangProfile` beeing set
	ErrNoPaidAccount = errors.New("you need to buy Minecraft to launch it")
	// ErrNoVersion is returned if no mc version was detected
	ErrNoVersion = errors.New("could not detect minecraft version")
	// ErrNoJava is returned if no java runtime is available to launch
	ErrNoJava = errors.New("no java runtime set to launch instance")
)

// GetLaunchManifest returns the merged manifest for the instance
func (i *Instance) GetLaunchManifest() (*minecraft.LaunchManifest, error) {
	man, err := i.launchManifest()
	if err != nil {
		return nil, err
	}

	if man.InheritsFrom != "" {
		parent, err := i.getVanillaManifest(man.InheritsFrom)
		if err != nil {
			return nil, err
		}
		man.MergeWith(parent)
	}
	return man, nil
}

// LaunchOptions are options for launching
type LaunchOptions struct {
	LaunchManifest *minecraft.LaunchManifest
	// Offline is not implemented
	Offline bool
	Java    string
	Server  bool
	// Demo launches the client in demo mode. should have no effect on a server
	Demo bool
	// JoinServer can be a server address to join after startup
	JoinServer string
	// StartSave can be a savegame name to start after startup
	StartSave string
	Debug     bool
	// RamMiB can be set to the amount of ram in MiB to start Minecraft with
	// 0 determins the amount by modcount + available system ram
	RamMiB int
}

// Launch will launch the minecraft instance
// prefer BuildLaunchCmd if you need more control over the process
func (i *Instance) Launch(opts *LaunchOptions) error {
	cmd, err := i.BuildLaunchCmd(opts)
	if err != nil {
		return err
	}

	// TODO: detatch from process if wanted
	if err := cmd.Run(); err != nil {
		return err
	}

	// we wait for the output to finish (the lines following this one usually are reached after ctrl-c was pressed)
	cmd.Wait()

	// minecraft server will always return code 130 when
	// stop was successful, so we ignore the error here
	if cmd.ProcessState.ExitCode() == 130 {
		return nil
	}
	// and return the error otherwise
	return err
}

func (i *Instance) getMojangData() (*mojang.Profile, *mojang.AuthResponse, error) {
	var (
		profile *mojang.Profile
		creds   *mojang.AuthResponse
	)

	creds = i.MojangCredentials
	if creds == nil {
		return nil, nil, ErrNoCredentials
	}

	profile = creds.SelectedProfile
	// do not allow non paid accounts to start minecraft
	// unpaid accounts should not have a profile
	if profile == nil {
		return nil, creds, ErrNoPaidAccount
	}

	return profile, creds, nil
}

// BuildLaunchCmd returns a go cmd ready to start minecraft
func (i *Instance) BuildLaunchCmd(opts *LaunchOptions) (*exec.Cmd, error) {
	// this file tells us how to construct the start command
	launchManifest := opts.LaunchManifest
	var err error

	// get manifest if not passed as option
	if launchManifest == nil {
		launchManifest, err = i.GetLaunchManifest()
		if err != nil {
			return nil, err
		}
	}

	// fallback to local java if nothing was set
	if opts.Java == "" {
		opts.Java = "java"
	}

	// create tmp dir for instance
	tmpName := i.Manifest.Package.Name + fmt.Sprintf("%d", time.Now().Unix())
	tmpDir, err := ioutil.TempDir("", tmpName)
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll(tmpDir) // cleanup dir after minecraft is closed
	libDir := filepath.Join(i.LibrariesDir())

	// build that spooky -cp arg
	var cpArgs []string

	libs := launchManifest.Libraries.Required()
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "osx"
	}

	for _, lib := range libs {
		// copy natives. not sure if this implementation is complete
		if len(lib.Natives) != 0 {
			// extract native to temp dir
			nativeID := lib.Natives[osName]
			native := lib.Downloads.Classifiers[nativeID]

			p := filepath.Join(libDir, native.Path)

			err := extractNative(p, tmpDir)
			if err != nil {
				return nil, err
			}
			cpArgs = append(cpArgs, filepath.Join(libDir, native.Path))
		} else {
			// append this library to our doom -cp arg
			libPath := lib.Filepath()
			cpArgs = append(cpArgs, filepath.Join(libDir, libPath))
		}
	}

	// finally append the minecraft.jar
	mcJar := filepath.Join(i.VersionsDir(), launchManifest.MinecraftVersion(), launchManifest.JarName())
	cpArgs = append(cpArgs, mcJar)

	gameArgs, err := i.gameArgs(launchManifest, opts)
	if err != nil {
		return nil, err
	}

	javaCpSeperator := ":"
	// of course
	if runtime.GOOS == "windows" {
		javaCpSeperator = ";"
	}

	var maxRamMiB int

	if opts.RamMiB == 0 {
		sysMemMiB := float64(memory.TotalMemory()) / 1024 / 1024

		// 1GiB for base Minecraft + every dependency takes 25 MiB
		maxRamMiB = 1024 + len(i.Lockfile.Dependencies)*25

		// we take 1/4 of the system memory if that is more
		maxRamMiB = int(math.Max(float64(maxRamMiB), sysMemMiB/4))
		// but not more than 85% of the memory
		maxRamMiB = int(math.Min(float64(maxRamMiB), sysMemMiB*0.85))
	} else {
		maxRamMiB = opts.RamMiB
	}

	cmdArgs := []string{
		"-Djava.library.path=" + tmpDir,
		"-Dminecraft.launcher.brand=minepkg",
		// "-Dminecraft.launcher.version=" + "0.0.2", // TODO: implement!
		"-Dminecraft.client.jar=" + mcJar,
		"-cp",
		strings.Join(cpArgs, javaCpSeperator),
		fmt.Sprintf("-Xmx%dM", maxRamMiB),
		"-XX:+UnlockExperimentalVMOptions",
		"-XX:+UseG1GC",
		"-XX:G1NewSizePercent=20",
		"-XX:G1ReservePercent=20",
		"-XX:MaxGCPauseMillis=50",
		"-XX:G1HeapRegionSize=32M",
		"-XX:ErrorFile=./jvm-error.log",
		launchManifest.MainClass,
	}

	if opts.RamMiB != 0 {
		cmdArgs = append([]string{fmt.Sprintf("-Xms%dM", opts.RamMiB)}, cmdArgs...)
	}

	// HACK: prepend this so macos does not crash
	if runtime.GOOS == "darwin" {
		cmdArgs = append([]string{"-XstartOnFirstThread"}, cmdArgs...)
	}

	if !opts.Server {
		cmdArgs = append(cmdArgs, gameArgs...)
	} else {
		// maybe don't use client args for server …
		cmdArgs = append(cmdArgs, "nogui")
	}

	if opts.Debug {
		fmt.Println("cmd: ")
		fmt.Println(cmdArgs)
		fmt.Println("tmpdir: " + tmpDir)
		os.Exit(0)
	}

	if opts.Java == "" {
		opts.Java = "java"
	}
	cmd := exec.Command(opts.Java, cmdArgs...)
	i.launchCmd = opts.Java + " " + strings.Join(cmdArgs, " ")

	cmd.Env = os.Environ()
	if opts.JoinServer != "" {
		cmd.Env = append(cmd.Env, "MINEPKG_COMPANION_PLAY=server://"+opts.JoinServer)
	}

	if opts.StartSave != "" {
		cmd.Env = append(cmd.Env, "MINEPKG_COMPANION_PLAY=local://"+opts.StartSave)
	}

	if opts.Server {
		cmd.Stdin = os.Stdin
	}

	// we catch ctrl-c to handle this by ourself
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		// stops the minecraft server
		cmd.Process.Signal(syscall.SIGTERM)
	}()

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set the process directory to our minecraft dir
	cmd.Dir = i.McDir()
	// some things may rely on PWD
	cmd.Env = append(cmd.Env, "PWD="+i.McDir())

	return cmd, nil
}

func (i *Instance) gameArgs(launchManifest *minecraft.LaunchManifest, opts *LaunchOptions) ([]string, error) {
	gameArgs := map[string]string{
		// the minecraft version
		"version_name": launchManifest.MinecraftVersion(),
		// minecraft game dir that contains saves, worlds & mods
		"game_directory": i.McDir(),
		// asset dir contains some shared minecraft resources like sounds & some textures
		"assets_root": i.AssetsDir(),
		// asset index version tells the game which assets to load. this usually just is the same
		// as the minecraft version
		"assets_index_name": launchManifest.Assets,
		// release / snapshot … etc
		"version_type": launchManifest.Type,
	}

	// this is not a server, we need to set some more auth data
	if !opts.Server && !opts.Demo {
		profile, creds, err := i.getMojangData()
		if err != nil {
			return nil, err
		}
		gameArgs["auth_player_name"] = profile.Name
		gameArgs["auth_uuid"] = profile.ID
		gameArgs["auth_access_token"] = creds.AccessToken
		gameArgs["user_type"] = "mojang" // unsure about this one (legacy mc login flag?)
	}

	finalGameArgs := make([]string, 0, len(gameArgs)*2)
	launchArgsTemplate := launchManifest.LaunchArgs()
	for i := 0; i < len(launchArgsTemplate)-1; i += 2 {
		arg := launchArgsTemplate[i]
		// looks something like ${version_name}
		valueTemplate := launchArgsTemplate[i+1]
		// cut to just version_name
		valueName := valueTemplate[2 : len(valueTemplate)-1]

		// found in our args map
		if val, ok := gameArgs[valueName]; ok {
			// append to final args
			finalGameArgs = append(finalGameArgs, arg, val)
		}
	}
	return finalGameArgs, nil
}

func (i *Instance) launchManifest() (*minecraft.LaunchManifest, error) {
	lockfile := i.Lockfile
	if lockfile == nil {
		i.initLockfile()
	}
	buf, err := ioutil.ReadFile(filepath.Join(i.VersionsDir(), lockfile.McManifestName()))
	if err == nil {
		man := minecraft.LaunchManifest{}
		json.Unmarshal(buf, &man)
		return &man, nil
	}

	switch i.Platform() {
	case PlatformFabric:
		return i.fetchFabricManifest(lockfile.Fabric)
	case PlatformForge:
		// TODO: forge
		panic("Forge is not supported")
	default:
		return i.getVanillaManifest(i.Manifest.Requirements.Minecraft)
	}
}

func (i *Instance) getVanillaManifest(v string) (*minecraft.LaunchManifest, error) {
	buf, err := ioutil.ReadFile(filepath.Join(i.VersionsDir(), v, v+".json"))
	if err != nil {
		return i.fetchVanillaManifest(v)
		// return nil, err
	}
	instructions := minecraft.LaunchManifest{}
	json.Unmarshal(buf, &instructions)
	return &instructions, nil
}

func (i *Instance) fetchFabricManifest(lock *manifest.FabricLock) (*minecraft.LaunchManifest, error) {
	manifest := minecraft.LaunchManifest{}
	loader := lock.FabricLoader
	mappings := lock.Mapping
	minecraft := lock.Minecraft

	version := minecraft + "-fabric-" + loader
	dir := filepath.Join(i.VersionsDir(), minecraft+"-fabric-"+loader)
	file := filepath.Join(dir, version+".json")

	// cached
	if rawMan, err := ioutil.ReadFile(file); err == nil {
		err := json.Unmarshal(rawMan, &manifest)
		if err != nil {
			return nil, err
		}
		return &manifest, nil
	}

	res, err := http.Get("https://fabricmc.net/download/vanilla?format=profileJson&loader=" + url.QueryEscape(loader) + "&yarn=" + url.QueryEscape(mappings))
	if err != nil {
		return nil, err
	}

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, err
	}
	ioutil.WriteFile(filepath.Join(dir, version+".json"), buf, 0666)

	if err = json.Unmarshal(buf, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func (i *Instance) fetchVanillaManifest(version string) (*minecraft.LaunchManifest, error) {
	mcVersions, err := GetMinecraftReleases(context.TODO())
	if err != nil {
		return nil, err
	}

	manifestURL := ""
	for _, mc := range mcVersions.Versions {
		if mc.ID == version {
			manifestURL = mc.URL
		}
	}
	if manifestURL == "" {
		return nil, ErrNoVersion
	}

	manifest := minecraft.LaunchManifest{}
	res, err := http.Get(manifestURL)
	if err != nil {
		return nil, err
	}

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(i.VersionsDir(), version)
	os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, err
	}
	ioutil.WriteFile(filepath.Join(dir, version+".json"), buf, 0666)

	if err = json.Unmarshal(buf, &manifest); err != nil {
		return nil, err
	}

	// TODO: this is a side effect. it should not be here
	jarRes, err := http.Get(manifest.Downloads.Client.URL)
	if err != nil {
		return nil, err
	}
	jarDest, err := os.Create(filepath.Join(dir, version+".jar"))
	if err != nil {
		return nil, err
	}
	defer jarDest.Close()

	// copy the jar
	if _, err = io.Copy(jarDest, jarRes.Body); err != nil {
		return nil, err
	}

	return &manifest, nil
}
