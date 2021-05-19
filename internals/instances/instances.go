package instances

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwalton/gchalk"
	"github.com/minepkg/minepkg/internals/commands"

	"github.com/BurntSushi/toml"
	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/mojang"
	"github.com/minepkg/minepkg/pkg/manifest"
	strcase "github.com/stoewer/go-strcase"
)

var (
	// PlatformVanilla is a vanilla minecraft instance
	PlatformVanilla uint8 = 1
	// PlatformFabric is a fabric minecraft instance
	PlatformFabric uint8 = 2
	// PlatformForge is forge minecraft instance
	PlatformForge uint8 = 3

	// ErrNoInstance is returned if no mc instance was found
	ErrNoInstance = &commands.CliError{
		Text: "no minepkg.toml file was found in this directory",
		Suggestions: []string{
			fmt.Sprintf("Create it with %s", gchalk.Bold("minepkg init")),
			"Move into a folder containing a minepkg.toml file",
		},
	}
	// ErrMissingRequirementMinecraft is returned if requirements.minecraft is not set
	ErrMissingRequirementMinecraft = &commands.CliError{
		Text: "the manifest is missing the required requirements.minecraft field",
		Suggestions: []string{
			"Add the field as documented here https://preview.minepkg.io/docs/manifest#requirements",
		},
	}
)

// Instance describes a locally installed minecraft instance
type Instance struct {
	// GlobalDir contains persistent instance data
	// on linux this usually is $HOME/.config/minepkg
	GlobalDir string
	// CacheDir is similar to cache dir but only contains data that can easily be redownloaded
	// like java binaries, libraries, assets, versions & mod cache
	// on linux this usually is $HOME/.cache/minepkg
	CacheDir string
	// Directory is the path of the instance. defaults to current working directory
	Directory         string
	Manifest          *manifest.Manifest
	Lockfile          *manifest.Lockfile
	MojangCredentials *mojang.AuthResponse
	MinepkgAPI        *api.MinepkgAPI

	isFromWd   bool
	launchCmd  string
	javaBinary string
}

// LaunchCmd returns the cmd used to launch minecraft (if started)
func (i *Instance) LaunchCmd() string {
	return i.launchCmd
}

// VersionsDir returns the path to the versions directory
func (i *Instance) VersionsDir() string {
	return filepath.Join(i.CacheDir, "versions")
}

// AssetsDir returns the path to the assets directory
// it contains some shared Minecraft resources like sounds & some textures
func (i *Instance) AssetsDir() string {
	return filepath.Join(i.CacheDir, "assets")
}

// LibrariesDir returns the path to the libraries directory
// contains libraries needed to load minecraft
func (i *Instance) LibrariesDir() string {
	return filepath.Join(i.CacheDir, "libraries")
}

// InstancesDir returns the path to the "global" instances directory
func (i *Instance) InstancesDir() string {
	return filepath.Join(i.GlobalDir, "instances")
}

// PackageCacheDir returns the path to the cache directory. contains downloaded packages (mods & modpacks)
func (i *Instance) PackageCacheDir() string {
	return filepath.Join(i.CacheDir, "cache")
}

// JavaDir returns the path for local java binaries
func (i *Instance) JavaDir() string {
	return filepath.Join(i.CacheDir, "java")
}

// McDir is the path where the actual Minecraft instance is living. This is the `minecraft` subfolder
// this folder contains saves, configs & mods that should be loaded
func (i *Instance) McDir() string {
	return filepath.Join(i.Directory, "minecraft")
}

// ModsDir is the path where the mods get linked to. This is the `minecraft/mods` subfolder
func (i *Instance) ModsDir() string {
	return filepath.Join(i.Directory, "minecraft/mods")
}

// OverwritesDir is the path where overwrite files reside in. They get copied to `McDir` on launch. This is the `overwrites` subfolder
func (i *Instance) OverwritesDir() string {
	return filepath.Join(i.Directory, "overwrites")
}

// ManifestPath is the path to the `minepkg.toml`. The file does not necessarily exist
func (i *Instance) ManifestPath() string {
	return filepath.Join(i.Directory, "minepkg.toml")
}

// LockfilePath is the path to the `minepkg-lock.toml`. The file does not necessarily exist
func (i *Instance) LockfilePath() string {
	return filepath.Join(i.Directory, "minepkg-lock.toml")
}

// Platform returns the type of loader required to start this instance
func (i *Instance) Platform() uint8 {
	switch {
	case i.Manifest.Requirements.Fabric != "":
		return PlatformFabric
	case i.Manifest.Requirements.Forge != "":
		return PlatformForge
	default:
		return PlatformVanilla
	}
}

// Desc returns a one-liner summary of this instance
func (i *Instance) Desc() string {
	manifest := i.Manifest

	depCount := fmt.Sprintf(" %d deps ", len(manifest.Dependencies))
	name := fmt.Sprintf(" 📦 %s ", manifest.Package.Name)
	build := []string{
		gchalk.BgBlue(name),
		gchalk.BgGray(depCount),
	}
	return strings.Join(build, "")
}

// NewEmptyInstance returns a new instance with the default settings
// panics if user config or cache directory can not be determined
func NewEmptyInstance() *Instance {
	userConfig, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}
	userCache, err := os.UserCacheDir()
	if err != nil {
		panic(err)
	}

	return &Instance{
		GlobalDir: filepath.Join(userConfig, "minepkg"),
		CacheDir:  filepath.Join(userCache, "minepkg"),
	}
}

// NewInstanceFromWd tries to detect a minecraft instance in the current working directory
// returning it, if successful
func NewInstanceFromWd() (*Instance, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	manifestToml, err := ioutil.ReadFile("./minepkg.toml")
	if err != nil {
		// TODO only for not found errors
		return nil, ErrNoInstance
	}
	manifest := manifest.Manifest{}
	if err = toml.Unmarshal(manifestToml, &manifest); err != nil {
		return nil, err
	}

	userConfig, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	userCache, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}

	instance := &Instance{
		Manifest:  &manifest,
		Directory: dir,
		GlobalDir: filepath.Join(userConfig, "minepkg"),
		CacheDir:  filepath.Join(userCache, "minepkg"),
		isFromWd:  true,
	}

	// initialize manifest
	if err := instance.initManifest(); err != nil {
		return nil, err
	}

	// initialize lockfile
	if err := instance.initLockfile(); err != nil {
		return nil, err
	}

	return instance, nil
}

// initManifest sets the manifest file or creates one
func (i *Instance) initManifest() error {
	minepkg, err := ioutil.ReadFile(i.ManifestPath())
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		manifest := manifest.New()
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		name := filepath.Base(wd)

		manifest.Package.Name = strcase.KebabCase(name)
		i.Manifest = manifest

		return nil
	}

	manifest := manifest.Manifest{}
	err = toml.Unmarshal(minepkg, &manifest)
	if err != nil {
		return err
	}

	i.Manifest = &manifest

	if i.Manifest.Requirements.Minecraft == "" {
		return ErrMissingRequirementMinecraft
	}
	return nil
}

// initLockfile sets the lockfile or creates one
func (i *Instance) initLockfile() error {
	rawLockfile, err := ioutil.ReadFile(i.LockfilePath())
	if err != nil {
		// non existing lockfile is not bad
		if os.IsNotExist(err) {
			return nil
		}
		// this is bad
		return err
	}

	lockfile := manifest.Lockfile{}
	err = toml.Unmarshal(rawLockfile, &lockfile)
	if err != nil {
		return err
	}
	if lockfile.Dependencies == nil {
		lockfile.Dependencies = make(map[string]*manifest.DependencyLock)
	}

	i.Lockfile = &lockfile
	return nil
}

// SaveManifest saves the manifest to the current directory
func (i *Instance) SaveManifest() error {
	manifest := i.Manifest.Buffer()
	return ioutil.WriteFile(i.ManifestPath(), manifest.Bytes(), 0644)
}

// SaveLockfile saves the lockfile to the current directory
func (i *Instance) SaveLockfile() error {
	lockfile := i.Lockfile.Buffer()
	return ioutil.WriteFile(i.LockfilePath(), lockfile.Bytes(), 0644)
}
