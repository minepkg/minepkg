package instances

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/minepkg/minepkg/internals/minecraft"
	"github.com/minepkg/minepkg/pkg/manifest"
	"github.com/pbnjay/memory"
	"github.com/shirou/gopsutil/process"
)

var (
	// ErrLaunchNotImplemented is returned if attempting to start a non vanilla instance
	ErrLaunchNotImplemented = errors.New("can only launch vanilla & fabric instances (for now)")
	// ErrNoCredentials is returned when an instance is launched without `MojangProfile` being set
	ErrNoCredentials = errors.New("can not launch without mojang credentials")
	// ErrNoPaidAccount is returned when an instance is launched without `MojangProfile` being set
	ErrNoPaidAccount = errors.New("you need to buy Minecraft to launch it")
	// ErrInvalidVersion is returned if no mc version was detected
	ErrInvalidVersion = errors.New("supplied version could not be found")
	// ErrNoJava is returned if no java runtime is available to launch
	ErrNoJava = errors.New("no java runtime set to launch instance")
)

// GetLaunchManifest returns the merged manifest for the instance
func (i *Instance) GetLaunchManifest() (*minecraft.LaunchManifest, error) {
	if i.launchManifest != nil {
		return i.launchManifest, nil
	}

	log.Println("Generating launch manifest")

	man, err := i.readLaunchManifest()
	if err != nil {
		return nil, err
	}

	if man.InheritsFrom != "" {
		parent, err := i.getVanillaManifest(man.InheritsFrom)
		if err != nil {
			return nil, err
		}
		minecraft.MergeManifests(man, parent)
	}

	i.launchManifest = man
	return man, nil
}

// SetLaunchManifest sets the launch manifest for the instance
func (i *Instance) SetLaunchManifest(m *minecraft.LaunchManifest) {
	i.launchManifest = m
}

// LaunchOptions are options for launching
type LaunchOptions struct {
	LaunchManifest *minecraft.LaunchManifest
	Stdout         io.Writer
	Stderr         io.Writer
	// Offline is not implemented
	Offline bool
	Java    string
	Server  bool
	// Demo launches the client in demo mode. should have no effect on a server
	Demo bool
	// JoinServer can be a server address to join after startup
	JoinServer string
	// StartSave can be a save game name to start after startup
	StartSave string
	Debug     bool
	// RamMiB can be set to the amount of ram in MiB to start Minecraft with
	// 0 determines the amount by mod count + available system ram
	RamMiB int
	// Environment variables to set
	Env []string
}

// Launch will launch the minecraft instance
// prefer BuildLaunchCmd if you need more control over the process
func (i *Instance) Launch(opts *LaunchOptions) error {
	cmd, err := i.BuildLaunchCmd(opts)
	if err != nil {
		return err
	}

	// TODO: detach from process if wanted
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

	libs := minecraft.RequiredLibraries(launchManifest.Libraries)
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

	classPath := strings.Join(cpArgs, cpSeparator())

	gameArgs, err := i.launchManifestArgs(launchManifest, opts, classPath, tmpDir)
	if err != nil {
		return nil, err
	}

	// filter out "-Xmx" args
	filteredArgs := make([]string, 0, len(gameArgs))
	for _, arg := range gameArgs {
		if !strings.HasPrefix(arg, "-Xmx") {
			filteredArgs = append(filteredArgs, arg)
		}
	}
	gameArgs = filteredArgs

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
		"-Dminecraft.client.jar=" + mcJar,
		fmt.Sprintf("-Xmx%dM", maxRamMiB),
		"-XX:+UnlockExperimentalVMOptions",
		"-XX:+UseG1GC",
		"-XX:G1NewSizePercent=20",
		"-XX:G1ReservePercent=20",
		"-XX:MaxGCPauseMillis=50",
		"-XX:G1HeapRegionSize=32M",
		"-XX:ErrorFile=./jvm-error.log",
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
		fmt.Println("tmp dir: " + tmpDir)
		os.Exit(0)
	}

	if opts.Java == "" {
		opts.Java = "java"
	}
	cmd := exec.Command(opts.Java, cmdArgs...)
	i.launchCmd = opts.Java + " " + strings.Join(cmdArgs, " ")

	cmd.Env = os.Environ()
	if opts.JoinServer != "" {
		log.Println("joinServer", opts.StartSave)
		cmd.Env = append(cmd.Env, "MINEPKG_COMPANION_PLAY=server://"+opts.JoinServer)
	}

	if opts.StartSave != "" {
		log.Println("startSave", opts.StartSave)
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
		fmt.Println("Caught interrupt, stopping minecraft")
		// stops the minecraft server
		cmd.Process.Signal(syscall.SIGTERM)
		signal.Stop(c)

		// send SIGTERM to own process
		p := &process.Process{Pid: int32(os.Getpid())}
		p.Terminate()
	}()

	if opts.Stdout != nil {
		cmd.Stdout = opts.Stdout
	} else {
		cmd.Stdout = os.Stdout
	}
	if opts.Stderr != nil {
		cmd.Stderr = opts.Stderr
	} else {
		cmd.Stderr = os.Stderr
	}

	cmd.Stderr = os.Stderr

	// Set the process directory to our minecraft dir
	cmd.Dir = i.McDir()
	// some things may rely on PWD
	cmd.Env = append(cmd.Env, opts.Env...)
	cmd.Env = append(cmd.Env, "PWD="+i.McDir())

	return cmd, nil
}

// launchManifestArgs returns a slice of args came from the launch manifest
// it replaces known variables in those args (like "game_directory") with the actual value
func (i *Instance) launchManifestArgs(launchManifest *minecraft.LaunchManifest, opts *LaunchOptions, classPaths string, nativesDir string) ([]string, error) {

	if launchManifest.MainClass == "" {
		log.Println("[WARN] launchManifest.MainClass is empty")
	}

	if launchManifest.Type == "" {
		log.Println("[WARN] launchManifest.type is empty")
	}

	if launchManifest.MinecraftArguments != "" {
		log.Println("[INFO] launchManifest is using (the old style) minecraftArguments")
	}

	version := launchManifest.MinecraftVersion()
	if version == "" {
		return nil, errors.New("launchManifest does not contain a minecraft version (missing ID and inheritsFrom)")
	}

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
		"version_type":        launchManifest.Type,
		"launcher_name":       "minepkg",
		"launcher_version":    "0.0.0",
		"classpath":           classPaths,
		"classpath_separator": cpSeparator(),
		"natives_directory":   nativesDir,
		"library_directory":   i.LibrariesDir(),
	}

	// this is not a server, we need to set some more auth data
	if !opts.Server && !opts.Demo {
		creds, err := i.getLaunchCredentials()
		if err != nil {
			return nil, err
		}
		gameArgs["auth_player_name"] = creds.PlayerName
		gameArgs["auth_uuid"] = creds.UUID
		gameArgs["auth_access_token"] = creds.AccessToken
		gameArgs["user_type"] = creds.UserType
		gameArgs["clientid"] = creds.ClientID

		if creds.UserType == "msa" {
			gameArgs["auth_xuid"] = creds.XUID
		}
	}

	finalGameArgs := make([]string, 0, len(gameArgs))
	launchArgsTemplate := launchManifest.FullArgs()

	variableRegex := regexp.MustCompile(`\$\{[a-zA-Z0-9_]+\}`)

	// build string replacer out of gameArgs
	replacerArgs := make([]string, 0, len(gameArgs)*2)
	for k, v := range gameArgs {
		replacerArgs = append(replacerArgs, "${"+k+"}", v)
	}
	replacer := strings.NewReplacer(replacerArgs...)

	for _, template := range launchArgsTemplate {
		// replace all ${var} with their value
		replaced := replacer.Replace(template)

		// check for any remaining ${var} and replace them with empty string
		if variableRegex.MatchString(replaced) {
			log.Println("[WARN] found unresolvable variable in launch args: " + replaced + "")
			replaced = variableRegex.ReplaceAllString(replaced, "")
			// TODO: filter out pairs not only the value!
		}

		// append replaced arg to finalGameArgs
		finalGameArgs = append(finalGameArgs, replaced)
	}

	return finalGameArgs, nil
}

func (i *Instance) readLaunchManifest() (*minecraft.LaunchManifest, error) {
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
		return i.getVanillaManifest(lockfile.MinecraftVersion())
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
	minecraft := lock.Minecraft

	version := minecraft + "-fabric-" + loader
	dir := filepath.Join(i.VersionsDir(), minecraft+"-fabric-"+loader)
	file := filepath.Join(dir, version+".json")

	// cached
	if rawMan, err := ioutil.ReadFile(file); err == nil {
		err := json.Unmarshal(rawMan, &manifest)
		if err == nil {
			return &manifest, nil
		}
		fmt.Printf("WARNING: Failed to parse cached manifest %s (this is a bug pls report)\n", file)
		// corrupted manifest, try downloading
	}

	profileURL := fmt.Sprintf(
		"https://meta.fabricmc.net/v2/versions/loader/%s/%s/profile/json",
		url.QueryEscape(minecraft),
		url.QueryEscape(loader),
	)
	res, err := http.Get(profileURL)
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
		return nil, ErrInvalidVersion
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

func cpSeparator() string {
	if runtime.GOOS == "windows" {
		return ";"
	}
	return ":"
}
