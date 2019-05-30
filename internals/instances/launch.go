package instances

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fiws/minepkg/pkg/manifest"

	"github.com/Masterminds/semver"
	homedir "github.com/mitchellh/go-homedir"
)

var (
	// ErrorLaunchNotImplemented is returned if attemting to start a non vanilla instance
	ErrorLaunchNotImplemented = errors.New("Can only launch vanilla instances (for now)")
	// ErrorNoCredentials is returned when an instance is launched without `MojangProfile` beeing set
	ErrorNoCredentials = errors.New("Can not launch without mojang credentials")
	// ErrorNoFabricLoader is returned if the wanted fabric version was not found
	ErrorNoFabricLoader = errors.New("Could not find wanted fabric version")
)

// GetLaunchManifest returns the merged manifest for the instance
func (m *McInstance) GetLaunchManifest() (*LaunchManifest, error) {
	man, err := m.launchManifest()
	if err != nil {
		return nil, err
	}

	if man.InheritsFrom != "" {
		parent, err := m.getLaunchManifest(man.InheritsFrom)
		if err != nil {
			return nil, err
		}
		man.MergeWith(parent)
	}
	return man, nil
}

// LaunchOptions are options for launching
type LaunchOptions struct {
	LaunchManifest *LaunchManifest
	// SkipDownload will NOT download missing assets & libraries
	SkipDownload bool
	// Offline is not implemented
	Offline bool
	Java    string
}

// Launch starts the minecraft instance
func (m *McInstance) Launch(opts *LaunchOptions) error {
	home, _ := homedir.Dir()
	globalDir := filepath.Join(home, ".minepkg")
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	creds := m.MojangCredentials
	profile := creds.SelectedProfile
	if profile == nil {
		return ErrorNoCredentials
	}

	// this file tells us howto construct the start command
	launchManifest := opts.LaunchManifest

	// get manifest if not passed as option
	if launchManifest == nil {
		launchManifest, err = m.GetLaunchManifest()
		if err != nil {
			return err
		}
	}

	// Download assets if not skipped
	if opts.SkipDownload != true {
		m.ensureAssets(launchManifest)
	}

	// create tmp dir for instance
	tmpName := m.Manifest.Package.Name + fmt.Sprintf("%d", time.Now().Unix())
	tmpDir, err := ioutil.TempDir("", tmpName)
	if err != nil {
		return err
	}

	defer os.RemoveAll(tmpDir) // cleanup dir after minecraft is closed
	libDir := filepath.Join(globalDir, "libraries")

	// build that spooky -cp arg
	var cpArgs []string

	libs := launchManifest.Libraries.Required()

	for _, lib := range libs {

		if opts.SkipDownload != true {
			existOrDownload(lib)
		}

		// copy natives. not sure if this implementation is complete
		if len(lib.Natives) != 0 {
			// extract native to temp dir
			nativeID, _ := lib.Natives[runtime.GOOS]
			native := lib.Downloads.Classifiers[nativeID]

			p := filepath.Join(libDir, native.Path)

			err := extractNative(p, tmpDir)
			if err != nil {
				return err
			}
			cpArgs = append(cpArgs, filepath.Join(libDir, native.Path))
		} else {
			// append this library to our doom -cp arg
			libPath := lib.Filepath()
			cpArgs = append(cpArgs, filepath.Join(libDir, libPath))
		}
	}

	// finally append the minecraft.jar
	jarTarget := launchManifest.Jar
	if jarTarget == "" {
		jarTarget = launchManifest.Assets
	}
	mcJar := filepath.Join(globalDir, "versions", jarTarget, jarTarget+".jar")
	cpArgs = append(cpArgs, mcJar)

	replacer := strings.NewReplacer(
		v("auth_player_name"), profile.Name,
		v("version_name"), jarTarget,
		v("game_directory"), cwd,
		v("assets_root"), filepath.Join(m.Directory, "assets"),
		v("assets_index_name"), launchManifest.Assets, // asset index version
		v("auth_uuid"), profile.ID, // profile id
		v("auth_access_token"), creds.AccessToken,
		v("user_type"), "mojang", // unsure about this one (legacy mc login flag?)
		v("version_type"), launchManifest.Type, // release / snapshot … etc
	)

	args := replacer.Replace(launchManifest.LaunchArgs())

	javaCpSeperator := ":"
	// of course
	if runtime.GOOS == "windows" {
		javaCpSeperator = ";"
	}

	cmdArgs := []string{
		"-Xss1M",
		"-Djava.library.path=" + tmpDir,
		"-Dminecraft.launcher.brand=minepkg",
		// "-Dminecraft.launcher.version=" + "0.0.2", // TODO: implement!
		"-Dminecraft.client.jar=" + mcJar,
		"-cp",
		strings.Join(cpArgs, javaCpSeperator),
		// "-Xmx2G", // TODO: option!
		"-XX:+UnlockExperimentalVMOptions",
		"-XX:+UseG1GC",
		"-XX:G1NewSizePercent=20",
		"-XX:G1ReservePercent=20",
		"-XX:MaxGCPauseMillis=50",
		"-XX:G1HeapRegionSize=32M",
		launchManifest.MainClass,
	}
	cmdArgs = append(cmdArgs, strings.Split(args, " ")...)

	// fmt.Println("cmd: ")
	// fmt.Println(cmdArgs)
	// fmt.Println("tmpdir: + " + tmpDir)
	// os.Exit(0)

	if opts.Java == "" {
		opts.Java = "java"
	}
	cmd := exec.Command(opts.Java, cmdArgs...)

	// TODO: detatch from process
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	return err
}

func existOrDownload(lib lib) {
	home, _ := homedir.Dir()
	globalDir := filepath.Join(home, ".minepkg/libraries")
	path := filepath.Join(globalDir, lib.Filepath())
	url := lib.DownloadURL()
	if lib.Natives[runtime.GOOS] != "" {
		nativeID, _ := lib.Natives[runtime.GOOS]
		native := lib.Downloads.Classifiers[nativeID]
		url = native.URL
		path = filepath.Join(globalDir, native.Path)
	}
	if _, err := os.Stat(path); err == nil {
		return
	}

	res, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	if res.StatusCode != http.StatusOK {
		panic(url + " did not return status code 200")
	}
	fmt.Println("downloading: " + path)
	// create directory first
	os.MkdirAll(filepath.Dir(path), 0755)
	// file next
	target, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(target, res.Body)
	if err != nil {
		panic(err)
	}
}

func extractNative(jar string, target string) error {
	r, err := zip.OpenReader(jar)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		// skip META-INF dir
		if strings.HasPrefix(f.Name, "META-INF") {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		f, err := os.Create(filepath.Join(target, f.Name))
		if err != nil {
			return err
		}

		io.Copy(f, rc)
		rc.Close()
	}
	return nil
}

// v for variable
func v(s string) string {
	return "${" + s + "}"
}

func (m *McInstance) launchManifest() (*LaunchManifest, error) {
	lockfile := m.Lockfile
	if lockfile == nil {
		m.initLockfile()
	}
	buf, err := ioutil.ReadFile(filepath.Join(m.Directory, "versions", lockfile.McManifestName()))
	if err == nil {
		man := LaunchManifest{}
		json.Unmarshal(buf, &man)
		return &man, nil
	}

	switch m.Platform() {
	case PlatformFabric:
		return m.fetchFabricManifest(lockfile.Fabric.FabricLoader, lockfile.Fabric.Mapping)
	case PlatformForge:
		// TODO: forge
		panic("Forge is not supported")
	default:
		return m.getLaunchManifest(m.Manifest.Requirements.Minecraft)
	}
}

func (m *McInstance) getLaunchManifest(v string) (*LaunchManifest, error) {
	buf, err := ioutil.ReadFile(filepath.Join(m.Directory, "versions", v, v+".json"))
	if err != nil {
		return m.fetchVanillaManifest(v)
		// return nil, err
	}
	instructions := LaunchManifest{}
	json.Unmarshal(buf, &instructions)
	return &instructions, nil
}

func (m *McInstance) ResolveVanilaVersion(ctx context.Context) (*MinecraftRelease, error) {
	constraint, _ := semver.NewConstraint(m.Manifest.Requirements.Minecraft)
	res, err := GetMinecraftReleases(ctx)
	if err != nil {
		return nil, err
	}

	// find newest compatible version
	for _, v := range res.Versions {
		// TODO: some versions contain spaces
		semverVersion, err := semver.NewVersion(v.ID)

		// skip unparsable minecraft versions
		if err != nil {
			continue
		}

		if constraint.Check(semverVersion) {
			return &v, nil
		}
	}

	return nil, nil
}

func (m *McInstance) ResolveRequirements(ctx context.Context) error {
	m.Lockfile = manifest.NewLockfile()
	switch m.Platform() {
	case PlatformFabric:
		lock, err := m.ResolveFabricRequirement(ctx)
		if err != nil {
			return err
		}
		m.Lockfile.Fabric = lock
	case PlatformForge:
		fmt.Println("forge is not supported for now")
	case PlatformVanilla:
		version, err := m.ResolveVanilaVersion(ctx)
		if err != nil {
			return err
		}
		m.Lockfile.Vanilla = &manifest.VanillaLock{Minecraft: version.ID}
	}
	return nil
}

func (m *McInstance) ResolveFabricRequirement(ctx context.Context) (*manifest.FabricLock, error) {
	// TODO: check for invalid semver
	MCconstraint, _ := semver.NewConstraint(m.Manifest.Requirements.Minecraft)
	FabricLoaderConstraint, _ := semver.NewConstraint(m.Manifest.Requirements.Fabric)
	// mcVersions, err := GetMinecraftReleases(ctx)

	fabricMappings, err := getFabricMappingVersions(ctx)
	if err != nil {
		return nil, err
	}
	fabricLoaders, err := getFabricLoaderVersions(ctx)
	if err != nil {
		return nil, err
	}

	var foundMapping *fabricMappingVersion
	// find newest compatible version
	for _, v := range fabricMappings {
		// TODO: some versions contain spaces
		semverVersion, err := semver.NewVersion(v.GameVersion)

		// skip unparsable minecraft versions
		if err != nil {
			continue
		}

		if MCconstraint.Check(semverVersion) {
			foundMapping = &v
			break
		}
	}

	var foundLoader *fabricLoaderVersion
	// find newest compatible version
	for _, v := range fabricLoaders {
		// TODO: some versions contain spaces
		semverVersion, err := semver.NewVersion(v.Version)

		// skip unparsable minecraft versions
		if err != nil {
			continue
		}

		if FabricLoaderConstraint.Check(semverVersion) {
			foundLoader = &v
			break
		}
	}

	if foundLoader == nil {
		return nil, ErrorNoFabricLoader
	}

	return &manifest.FabricLock{
		Minecraft:    foundMapping.GameVersion,
		Mapping:      foundMapping.Version,
		FabricLoader: foundLoader.Version,
	}, nil
}

func (m *McInstance) fetchFabricManifest(loader string, mappings string) (*LaunchManifest, error) {
	manifest := LaunchManifest{}
	res, err := http.Get("https://fabricmc.net/download/vanilla?format=profileJson&loader=" + url.QueryEscape(loader) + "&yarn=" + url.QueryEscape(mappings))
	if err != nil {
		return nil, err
	}

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	version := m.Manifest.Requirements.Minecraft + "-fabric-" + loader
	dir := filepath.Join(m.Directory, "versions", m.Manifest.Requirements.Minecraft+"-fabric-"+loader)
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

func (m *McInstance) fetchVanillaManifest(version string) (*LaunchManifest, error) {
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
		return nil, ErrorNoVersion
	}

	manifest := LaunchManifest{}
	res, err := http.Get(manifestURL)
	if err != nil {
		return nil, err
	}

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(m.Directory, "versions", version)
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

	// copy the jar
	if _, err = io.Copy(jarDest, jarRes.Body); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// FindMissingLibraries returns all missing assets
func (m *McInstance) FindMissingLibraries(man *LaunchManifest) (Libraries, error) {
	missing := Libraries{}

	libs := man.Libraries.Required()
	globalDir := filepath.Join(m.Directory, "libraries")

	for _, lib := range libs {
		path := filepath.Join(globalDir, lib.Filepath())
		if _, err := os.Stat(path); err == nil {
			continue
		}

		missing = append(missing, lib)
	}

	return missing, nil
}

// FindMissingAssets returns all missing assets
func (m *McInstance) FindMissingAssets(man *LaunchManifest) ([]McAssetObject, error) {
	assets := mcAssetsIndex{}

	assetJSONPath := filepath.Join(m.Directory, "assets/indexes", man.Assets+".json")
	buf, err := ioutil.ReadFile(assetJSONPath)
	if err != nil {
		res, err := http.Get(man.AssetIndex.URL)
		if err != nil {
			return nil, err
		}

		buf, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		os.MkdirAll(filepath.Join(m.Directory, "assets/indexes"), os.ModePerm)
		err = ioutil.WriteFile(assetJSONPath, buf, 0666)
		if err != nil {
			return nil, err
		}
	}
	json.Unmarshal(buf, &assets)

	missing := make([]McAssetObject, 0)

	for _, asset := range assets.Objects {
		file := filepath.Join(m.Directory, "assets/objects", asset.UnixPath())
		if _, err := os.Stat(file); os.IsNotExist(err) {
			missing = append(missing, asset)
		}
	}

	return missing, nil
}

func (m *McInstance) ensureAssets(man *LaunchManifest) error {

	missing, err := m.FindMissingAssets(man)
	if err != nil {
		return err
	}

	for _, asset := range missing {
		fileRes, err := http.Get(asset.DownloadURL())
		// TODO: check status code and all the things!
		os.MkdirAll(filepath.Join(m.Directory, "assets/objects", asset.Hash[:2]), os.ModePerm)
		dest, err := os.Create(filepath.Join(m.Directory, "assets/objects", asset.UnixPath()))
		if err != nil {
			return err
		}
		_, err = io.Copy(dest, fileRes.Body)
		if err != nil {
			return err
		}
	}

	return nil
}
