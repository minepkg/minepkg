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
	"github.com/logrusorgru/aurora"
	homedir "github.com/mitchellh/go-homedir"
	strcase "github.com/stoewer/go-strcase"
)

const compatMMCFormat = 1

var (
	// FlavourVanilla is a vanilla minecraft instance
	FlavourVanilla uint8 = 1
	// FlavourServer is a server side instance
	FlavourServer uint8 = 3

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
	Flavour           uint8
	Directory         string
	ModsDirectory     string
	Manifest          *manifest.Manifest
	Lockfile          *manifest.Lockfile
	MojangCredentials *api.MojangAuthResponse
	MinepkgAPI        *api.MinepkgAPI
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
		aurora.BgBlue(name).String(),
		aurora.BgGray(depCount).Black().String(),
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

	var flavour uint8
	for _, entry := range entries {
		switch entry.Name() {
		case "versions":
			flavour = FlavourVanilla
			break
		case "server.properties":
			flavour = FlavourServer
			break
		case "minepkg.toml":
			flavour = FlavourServer
			home, _ := homedir.Dir()
			dir = filepath.Join(home, ".minepkg")
			break
		}
	}

	if flavour == 0 {
		return nil, ErrorNoInstance
	}

	instance := &Instance{
		Flavour:       flavour,
		ModsDirectory: modsDir,
		Directory:     dir,
	}

	err = instance.initManifest()
	if err != nil {
		return nil, err
	}

	err = instance.initLockfile()
	if err != nil {
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

// RefreshToken refreshed the mojang token
func (i *Instance) RefreshToken() error {
	newCreds, err := i.MinepkgAPI.MojangEnsureToken(
		i.MojangCredentials.AccessToken,
		i.MojangCredentials.ClientToken,
	)
	if err != nil {
		return err
	}
	fmt.Println(newCreds)
	i.MojangCredentials.AccessToken = newCreds.AccessToken
	i.MojangCredentials.ClientToken = newCreds.ClientToken
	return nil
}
