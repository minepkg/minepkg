package instances

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver"
	"github.com/fiws/minepkg/internals/manifest"
	"github.com/logrusorgru/aurora"
	"github.com/stoewer/go-strcase"
)

const compatMMCFormat = 1

var (
	// FlavourVanilla is a vanilla minecraft instance
	// usually installed with the official minecraft launcher
	FlavourVanilla uint8 = 1
	// FlavourMMC is a minecraft instance initiated with MultiMC
	FlavourMMC uint8 = 2
	// FlavourServer is a server side instance
	FlavourServer uint8 = 1

	// ErrorNoInstance is returned if no mc instance was found
	ErrorNoInstance = errors.New("Could not find minecraft instance in this directory")
	// ErrorNoVersion is returned if no mc version was detected
	ErrorNoVersion = errors.New("Could not detect minecraft version")
)

// McInstance describes a locally installed minecraft instance
type McInstance struct {
	Flavour       uint8
	Directory     string
	ModsDirectory string
	Manifest      *manifest.Manifest
}

// Desc returns a one-liner summary of this instance
func (m *McInstance) Desc() string {
	var flavourText string
	switch m.Flavour {
	case FlavourMMC:
		flavourText = "MMC"
	default:
		flavourText = "Vanilla"
	}
	manifest := m.Manifest

	flavourText = fmt.Sprintf(" ⌂ %s ", flavourText)
	version := fmt.Sprintf(" MC %s ", manifest.Requirements.MinecraftVersion)
	depCount := fmt.Sprintf(" %d deps ", len(manifest.Dependencies))
	name := fmt.Sprintf(" 📦 %s ", manifest.Package.Name)
	build := []string{
		aurora.BgBrown(flavourText).String(),
		aurora.BgGray(version).Black().String(),
		" ",
		aurora.BgBlue(name).String(),
		aurora.BgGray(depCount).Black().String(),
	}
	return strings.Join(build, "")
}

// Download downloads a mod into the mod directory
func (m *McInstance) Download(mod *manifest.ResolvedMod) error {
	res, err := http.Get(mod.DownloadURL)
	if err != nil {
		return err
	}
	dest, err := os.Create(path.Join(m.ModsDirectory, mod.LocalName()))
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

	modsDir := "mods"

	var flavour uint8
	for _, entry := range entries {
		switch entry.Name() {
		case "mmc-pack.json":
			flavour = FlavourMMC
			modsDir = detectMmcModsDir(entries)
			break
		case "versions":
			flavour = FlavourVanilla
			break
		case "server.properties":
			flavour = FlavourServer
			break
		}
	}

	if flavour == 0 {
		return nil, ErrorNoInstance
	}

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	instance := &McInstance{
		Flavour:       flavour,
		ModsDirectory: modsDir,
		Directory:     wd,
	}

	err = instance.initManifest()
	if err != nil {
		return nil, err
	}

	return instance, nil
}

// Version returns the minecraft version of the instance
func (m *McInstance) Version() *semver.Version {
	switch m.Flavour {
	case FlavourVanilla:
		entries, _ := ioutil.ReadDir("./version")
		versions := make(semver.Collection, len(entries))
		for i, version := range entries {
			versions[i] = semver.MustParse(version.Name())
		}
		// sort by highest version first
		sort.Sort(sort.Reverse(versions))
		return versions[0] // assume this is the version wanted
	case FlavourMMC:
		pack := mmcPack{}
		raw, _ := ioutil.ReadFile("./mmc-pack.json")
		json.Unmarshal(raw, &pack)
		if pack.FormatVersion != compatMMCFormat {
			panic("incompatible MMC version. Open a bug for minepkg")
		}
		for _, comp := range pack.Components {
			if comp.UID == "net.minecraft" {
				return semver.MustParse(comp.Version)
			}
		}
		fallthrough
	default:
		// fallback to 1.12.2 (?!)
		return semver.MustParse("1.12.2")
	}
}

// initManifest sets the manifest file or creates one
func (m *McInstance) initManifest() error {
	minepkg, err := ioutil.ReadFile("./minepkg.toml")
	if err != nil {
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
		manifest.Requirements.MinecraftVersion = version
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

func detectMmcModsDir(e []os.FileInfo) string {
	for _, entry := range e {
		name := entry.Name()
		if name == "minecraft" || name == ".minecraft" {
			return name + "/mods"
		}
	}

	return ""
}

type mmcPack struct {
	FormatVersion uint32         `json:"formatVersion"`
	Components    []mmcComponent `json:"components"`
}

type mmcComponent struct {
	UID     string `json:"uid"`
	Version string `json:"version"`
}
