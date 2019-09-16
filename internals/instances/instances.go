package instances

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/fiws/minepkg/pkg/api"
	"github.com/fiws/minepkg/pkg/manifest"
	"github.com/gookit/color"
	homedir "github.com/mitchellh/go-homedir"
	strcase "github.com/stoewer/go-strcase"
)

var (
	// PlatformVanilla is a vanilla minecraft instance
	PlatformVanilla uint8 = 1
	// PlatformFabric is a fabric minecraft instance
	PlatformFabric uint8 = 2
	// PlatformForge is forge minecraft instance
	PlatformForge uint8 = 3

	// ErrorNoInstance is returned if no mc instance was found
	ErrorNoInstance = errors.New("Could not find minecraft instance in this directory")
	// ErrorNoVersion is returned if no mc version was detected
	ErrorNoVersion = errors.New("Could not detect minecraft version")
)

// Instance describes a locally installed minecraft instance
type Instance struct {
	IsServer bool
	// GlobalDir is the directory containing everything required to run minecraft.
	// this includes the libraries, assets, versions & mod cache folder
	// it defaults to $HOME/.minepkg
	GlobalDir         string
	ModsDirectory     string
	Manifest          *manifest.Manifest
	Lockfile          *manifest.Lockfile
	MojangCredentials *api.MojangAuthResponse
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

// DetectInstance tries to detect a minecraft instance
// returning it, if succesfull
func DetectInstance() (*Instance, error) {
	entries, _ := ioutil.ReadDir("./")

	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	modsDir := filepath.Join(dir, "mods")

	isServer := false
	isInstance := false
	for _, entry := range entries {
		switch entry.Name() {
		case "server.properties":
			isServer = true
			isInstance = true
			break
		case "minepkg.toml":
			isInstance = true
			break
		}
	}

	if isInstance == false {
		return nil, ErrorNoInstance
	}

	home, _ := homedir.Dir()
	globalDir := filepath.Join(home, ".minepkg")

	instance := &Instance{
		IsServer:      isServer,
		ModsDirectory: modsDir,
		GlobalDir:     globalDir,
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
	minepkg, err := ioutil.ReadFile("./minepkg.toml")
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
		version := "1.14.2" // TODO: not static
		if version == "" {
			return ErrorNoVersion
		}
		// replace patch with placeholder
		version = version[:len(version)-1] + "x"

		manifest.Package.Name = strcase.KebabCase(name)
		manifest.Requirements.Minecraft = version
		i.Manifest = manifest
		return nil
	}

	manifest := manifest.Manifest{}
	_, err = toml.Decode(string(minepkg), &manifest)
	if err != nil {
		return err
	}

	i.Manifest = &manifest
	return nil
}

// initLockfile sets the lockfile or creates one
func (i *Instance) initLockfile() error {
	rawLockfile, err := ioutil.ReadFile("./minepkg-lock.toml")
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
	return ioutil.WriteFile("minepkg.toml", manifest.Bytes(), os.ModePerm)
}

// SaveLockfile saves the lockfile to the current directory
func (i *Instance) SaveLockfile() error {
	lockfile := i.Lockfile.Buffer()
	return ioutil.WriteFile("minepkg-lock.toml", lockfile.Bytes(), os.ModePerm)
}
