package instances

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/fiws/minepkg/internals/merrors"

	"github.com/BurntSushi/toml"
	"github.com/fiws/minepkg/pkg/api"
	"github.com/fiws/minepkg/pkg/manifest"
	"github.com/fiws/minepkg/pkg/mojang"
	"github.com/gookit/color"
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
	ErrNoInstance = &merrors.CliError{
		Err:  "No minepkg.toml file was found in this directory",
		Help: "Create a new modpack with \"minepkg init\" or move into a folder containing a minepkg.toml file",
	}
	// ErrMissingRequirementMinecraft is returned if requirements.minecraft is not set
	ErrMissingRequirementMinecraft = &merrors.CliError{
		Err:  "The manifest is missing the required requirements.minecraft field",
		Help: "Add the field as documented on https://test-www.minepkg.io/docs/manifest#requirements",
	}
)

// Instance describes a locally installed minecraft instance
type Instance struct {
	IsServer bool
	// GlobalDir is the directory containing everything required to run minecraft.
	// this includes the libraries, assets, versions & mod cache folder
	// it defaults to $HOME/.minepkg
	GlobalDir string
	// Directory is the path of the instance. defaults to current working directory
	Directory         string
	Manifest          *manifest.Manifest
	Lockfile          *manifest.Lockfile
	MojangCredentials *mojang.AuthResponse
	MinepkgAPI        *api.MinepkgAPI

	launchCmd  string
	javaBinary string
}

// LaunchCmd returns the cmd used to launch minecraft (if started)
func (i *Instance) LaunchCmd() string {
	return i.launchCmd
}

// VersionsDir returns the path to the versions directory
func (i *Instance) VersionsDir() string {
	return filepath.Join(i.GlobalDir, "versions")
}

// AssetsDir returns the path to the assets directory
func (i *Instance) AssetsDir() string {
	return filepath.Join(i.GlobalDir, "assets")
}

// LibrariesDir returns the path to the libraries directory
func (i *Instance) LibrariesDir() string {
	return filepath.Join(i.GlobalDir, "libraries")
}

// InstancesDir returns the path to the instances directory
func (i *Instance) InstancesDir() string {
	return filepath.Join(i.GlobalDir, "instances")
}

// CacheDir returns the path to the cache directory. contains downloaded packages (mods & modpacks)
func (i *Instance) CacheDir() string {
	return filepath.Join(i.GlobalDir, "cache")
}

// McDir is the path where the actual minecraft instance is living. This is the `minecraft` subfolder
func (i *Instance) McDir() string {
	return filepath.Join(i.Directory, "minecraft")
}

// ModsDir is the path where the mods get linked to. This is the `minecraft/mods` subfolder
func (i *Instance) ModsDir() string {
	return filepath.Join(i.Directory, "minecraft/mods")
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
		color.BgBlue.Sprint(name),
		color.BgGray.Sprint(depCount),
	}
	return strings.Join(build, "")
}

// NewEmptyInstance returns a new instance with the default settings
// panics if user homedir can not be determined
func NewEmptyInstance() *Instance {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	globalDir := filepath.Join(home, ".minepkg")

	return &Instance{
		GlobalDir: globalDir,
	}
}

// NewInstanceFromWd tries to detect a minecraft instance in the current working directory
// returning it, if succesfull
func NewInstanceFromWd() (*Instance, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	hasManifest := func() bool {
		info, err := os.Stat("./minepkg.toml")
		if err != nil {
			return false
		}
		return info.IsDir() != true
	}()

	// TODO: maybe add this to the minepkg toml.
	// detection is not very meaningful as this file is here even if the server was only started once
	// property should reflect if LAST launch was server
	isServer := func() bool {
		info, err := os.Stat("./minecraft/server.properties")
		if err != nil {
			return false
		}
		return info.IsDir() != true
	}()

	if hasManifest == false {
		return nil, ErrNoInstance
	}

	home, _ := os.UserHomeDir()
	globalDir := filepath.Join(home, ".minepkg")

	instance := &Instance{
		IsServer:  isServer,
		Directory: dir,
		GlobalDir: globalDir,
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
		if os.IsNotExist(err) != true {
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
	_, err = toml.Decode(string(minepkg), &manifest)
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
	_, err = toml.Decode(string(rawLockfile), &lockfile)
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
	return ioutil.WriteFile(i.ManifestPath(), manifest.Bytes(), os.ModePerm)
}

// SaveLockfile saves the lockfile to the current directory
func (i *Instance) SaveLockfile() error {
	lockfile := i.Lockfile.Buffer()
	return ioutil.WriteFile(i.LockfilePath(), lockfile.Bytes(), os.ModePerm)
}
