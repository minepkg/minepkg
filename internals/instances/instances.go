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
	"github.com/fiws/minepkg/internals/manifest"
	"github.com/logrusorgru/aurora"
	"github.com/stoewer/go-strcase"
)

var (
	// FlavourVanilla is a vanilla minecraft instance
	// usually installed with the official minecraft launcher
	FlavourVanilla uint8 = 1
	// FlavourMMC is a minecraft instance initiated with MultiMC
	FlavourMMC uint8 = 2

	// ErrorNoInstance is returned if no mc instance was found
	ErrorNoInstance = errors.New("Could not find minecraft instance in this directory")
)

// McInstance describes a locally installed minecraft instance
type McInstance struct {
	Flavour       uint8
	Version       string
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
	flavourText = fmt.Sprintf(" ⌂ %s ", flavourText)
	version := fmt.Sprintf(" MC %s ", m.Manifest.Requirements.MinecraftVersion)
	depCount := fmt.Sprintf(" %d deps ", len(m.Manifest.Dependencies))
	name := fmt.Sprintf(" 📦 %s ", m.Manifest.Package.Name)
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

	// TODO: wow, this is one ugly .includes()
	var flavour uint8
	for _, entry := range entries {
		switch entry.Name() {
		case "mmc-pack.json":
			flavour = FlavourMMC
			break
		case "versions":
			flavour = FlavourVanilla
			break
		}
	}

	if flavour == 0 {
		return nil, ErrorNoInstance
	}

	manifest, err := getManifest()
	if err != nil {
		return nil, err
	}

	return &McInstance{
		Flavour:       flavour,
		ModsDirectory: detectMmcModsDir(entries),
		Manifest:      manifest,
	}, nil
}

func getManifest() (*manifest.Manifest, error) {
	minepkg, err := ioutil.ReadFile("./minepkg.toml")
	if err != nil {
		m := manifest.New()
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		name := filepath.Base(wd)

		m.Package.Name = strcase.KebabCase(name)
		return m, nil
	}

	manifest := manifest.Manifest{}
	_, err = toml.Decode(string(minepkg), &manifest)
	if err != nil {
		return nil, err
	}
	return &manifest, nil
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
