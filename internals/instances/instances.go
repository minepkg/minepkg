package instances

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwalton/gchalk"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/internals/minecraft"
	"github.com/minepkg/minepkg/internals/provider"

	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/pkg/manifest"
	"github.com/pelletier/go-toml"
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
			"Add the field as documented here https://minepkg.io/docs/manifest#requirements",
		},
	}
)

// Instance describes a locally installed minecraft instance
type Instance struct {
	// GlobalDir contains persistent instance data
	// on linux this usually is $HOME/.config/minepkg
	GlobalDir string
	// CacheDir is similar to cache dir but only contains data that can easily be re-downloaded
	// like java binaries, libraries, assets, versions & mod cache
	// on linux this usually is $HOME/.cache/minepkg
	CacheDir string
	// Directory is the path of this instance. defaults to current working directory
	Directory       string
	Manifest        *manifest.Manifest
	Lockfile        *manifest.Lockfile
	MinepkgAPI      *api.MinepkgClient
	AuthCredentials *LaunchCredentials
	ProviderStore   *provider.Store

	isFromWd                     bool
	launchManifest               *minecraft.LaunchManifest
	launchCmd                    string
	lockfileNeedsRenameMigration bool
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

// LockfilePath is the path to the `.minepkg-lock.toml`. The file does not necessarily exist
func (i *Instance) LockfilePath() string {
	return filepath.Join(i.Directory, ".minepkg-lock.toml")
}

// legacyLockfilePath is the old path `minepkg-lock.toml` (no dot)
func (i *Instance) legacyLockfilePath() string {
	return filepath.Join(i.Directory, "minepkg-lock.toml")
}

// Platform returns the type of loader required to start this instance
func (i *Instance) Platform() uint8 {
	switch {
	case i.Manifest.Requirements.FabricLoader != "":
		return PlatformFabric
	case i.Manifest.Requirements.ForgeLoader != "":
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

// New returns a new instance with the default settings
// panics if user config or cache directory can not be determined
func New() *Instance {
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

// NewFromDir tries to detect a instance in the given directory
func NewFromDir(dir string) (*Instance, error) {
	manifestToml, err := ioutil.ReadFile("./minepkg.toml")
	if err != nil {
		// TODO only for not found errors
		return nil, ErrNoInstance
	}
	manifest := manifest.Manifest{}
	if err = toml.Unmarshal(manifestToml, &manifest); err != nil {
		return nil, err
	}

	if manifest.Requirements.Minecraft == "" {
		return nil, ErrMissingRequirementMinecraft
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

	// initialize lockfile
	if err := instance.initLockfile(); err != nil {
		return nil, err
	}

	// run migrations
	if err := instance.migrate(); err != nil {
		return nil, err
	}

	return instance, nil
}

// NewFromWd tries to detect a instance in the current working directory
func NewFromWd() (*Instance, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	return NewFromDir(dir)
}

// initLockfile sets the lockfile or creates one
func (i *Instance) initLockfile() error {
	lockfile, err := LockfileFromPath(i.LockfilePath())
	if err != nil {
		// this is bad
		if !os.IsNotExist(err) {
			return err
		}

		// non existing lockfile is not bad

		// try old name
		if lockfile, err = LockfileFromPath(i.legacyLockfilePath()); err != nil {
			if os.IsNotExist(err) {
				return nil
			} else {
				return err
			}
		}
		i.lockfileNeedsRenameMigration = true
	}

	i.Lockfile = lockfile
	return nil
}

// SaveManifest saves the manifest to the current directory
func (i *Instance) SaveManifest() error {
	// always remove the companion if it has been there (kinda hacky)
	i.Manifest.RemoveDependency("minepkg-companion")
	manifest := i.Manifest.Buffer()
	return ioutil.WriteFile(i.ManifestPath(), manifest.Bytes(), 0644)
}

// SaveLockfile saves the lockfile to the current directory
func (i *Instance) SaveLockfile() error {
	lockfile := i.Lockfile.Buffer()
	return ioutil.WriteFile(i.LockfilePath(), lockfile.Bytes(), 0644)
}
