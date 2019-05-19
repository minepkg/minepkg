package instances

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
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
	// usually installed with the official minecraft launcher
	FlavourVanilla uint8 = 1
	// FlavourMMC is a minecraft instance initiated with MultiMC
	FlavourMMC uint8 = 2
	// FlavourServer is a server side instance
	FlavourServer uint8 = 3
	// FlavourMinepkg is the native minepkg instance
	FlavourMinepkg uint8 = 4

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

// McInstance describes a locally installed minecraft instance
type McInstance struct {
	Flavour           uint8
	Directory         string
	ModsDirectory     string
	Manifest          *manifest.Manifest
	Lockfile          *manifest.Lockfile
	MojangCredentials *api.MojangAuthResponse
}

// Platform returns the type of loader required to start this instance
func (m *McInstance) Platform() uint8 {
	switch {
	case m.Manifest.Requirements.Fabric != "":
		return PlatformFabric
	case m.Manifest.Requirements.Forge != "":
		return PlatformForge
	default:
		return PlatformVanilla
	}
}

// Desc returns a one-liner summary of this instance
func (m *McInstance) Desc() string {
	manifest := m.Manifest

	depCount := fmt.Sprintf(" %d deps ", len(manifest.Dependencies))
	name := fmt.Sprintf(" 📦 %s ", manifest.Package.Name)
	build := []string{
		aurora.BgBlue(name).String(),
		aurora.BgGray(depCount).Black().String(),
	}
	return strings.Join(build, "")
}

// Download downloads a mod into the mod directory
func (m *McInstance) Download(name string, url string) error {
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("Unexpected status code %d for %s", res.StatusCode, url)
	}
	dest, err := os.Create(path.Join(m.ModsDirectory, name))
	if err != nil {
		return err
	}
	io.Copy(dest, res.Body)
	return nil
}

// Add a new mod using a reader
func (m *McInstance) Add(name string, r io.Reader) error {
	dest, err := os.Create(path.Join(m.ModsDirectory, name))
	if err != nil {
		return err
	}
	_, err = io.Copy(dest, r)
	return err
}

// DetectInstance tries to detect a minecraft instance
// returning it, if succesfull
func DetectInstance() (*McInstance, error) {
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

	instance := &McInstance{
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
func (m *McInstance) initManifest() error {
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
		version := m.Version().String()
		if version == "" {
			return ErrorNoVersion
		}
		// replace patch with placeholder
		version = version[:len(version)-1] + "x"

		manifest.Package.Name = strcase.KebabCase(name)
		manifest.Requirements.Minecraft = version
		m.Manifest = manifest
		return nil
	}

	manifest := manifest.Manifest{}
	_, err = toml.Decode(string(minepkg), &manifest)
	if err != nil {
		return err
	}

	m.Manifest = &manifest
	return nil
}

// initLockfile sets the lockfile or creates one
func (m *McInstance) initLockfile() error {
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

	m.Lockfile = &lockfile
	return nil
}
