package manifest

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// TODO: get those comments to be in the initial toml written to disk
const newPackageTemplate = `
# WARNING: This package might not work with future versions
#   of minepkg and might need to be updated then.
# This is a preview version of the minepkg.toml format!
version = 0

[package]
type = "modpack"

# This is the name of your modpack (in case you want to publish it)
# Please choose a unique name without spaces or special characters (- and _ are allowed)
name = "examplepack"
description = ""
version = "0.0.1"

# These are global requirements
[requirements]
minecraft-version = "1.12.x"

# All packages included in your modpack
[dependencies]

`

// Dependency defines a dependency that can be saved and installed from
// the minepkg.toml
type Dependency interface {
	Identifier() string // Identifier is the (unique) name of the dependency
	fmt.Stringer        // used as `data` in `name = "[data]"` inside the minepkg.toml
}

type Dependencies map[string]string

type Manifest struct {
	Version int `toml:"version"`
	Package struct {
		Description string   `toml:"description"`
		Name        string   `toml:"name"`
		Type        string   `toml:"type"`
		Version     string   `toml:"version"`
		Extends     []string `toml:"extends"`
	} `toml:"package"`
	Requirements struct {
		MinecraftVersion string `toml:"minecraft-version"`
	} `toml:"requirements"`
	Dependencies `toml:"dependencies"`
}

type ParsedDependency struct {
	Provider string
	Target   string
	Meta     string
}

func (d *Dependencies) Parsed() []ParsedDependency {
	parsed := make([]ParsedDependency, len(*d))

	i := 0
	for _, dep := range *d {
		splited := strings.SplitN(dep, ":", 1)
		parsed[i] = ParsedDependency{Provider: splited[0], Target: splited[1]}
		i++
	}

	return parsed
}

// FullDependencies returns all dependencies including dependencies
// specified in external toml files (Package.Extends)
func (m *Manifest) FullDependencies() (*Dependencies, error) {

	deps := Dependencies{}
	// merge with local deps
	for k, v := range m.Dependencies {
		deps[k] = v
	}
	fetched := make(map[string]bool)

	var resolve func(urls []string) error
	resolve = func(urls []string) error {
		for _, url := range urls {
			_, ok := fetched[url]
			if ok == true {
				break
			}
			fetched[url] = true
			m, err := fetchExtends(url)
			if err != nil {
				return err
			}
			// merge with deps
			for k, v := range m.Dependencies {
				deps[k] = v
			}

			// recursion – resolve its extends
			resolve(m.Package.Extends)
		}
		return nil
	}

	err := resolve(m.Package.Extends)
	if err != nil {
		return nil, err
	}

	return &deps, nil
}

// AddDependency adds a new dependency to the manifest
func (m *Manifest) AddDependency(d Dependency) {
	if m.Dependencies == nil {
		fmt.Println("[dependencies] was not initialized. This should not happen. Please report this")
		m.Dependencies = make(map[string]string)
	}
	m.Dependencies[d.Identifier()] = d.String()
}

// Save saves the manifest to disk
func (m *Manifest) Save() error {
	file, err := os.Create("minepkg.toml")
	if err != nil {
		return err
	}
	_, err = io.Copy(file, m.Buffer())
	if err != nil {
		return err
	}
	return nil
}

// Buffer returns the manifest as toml in Buffer form
func (m *Manifest) Buffer() *bytes.Buffer {
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(m); err != nil {
		log.Fatal(err)
	}
	return buf
}

func (m *Manifest) String() string {
	return m.Buffer().String()
}

// New returns a new manifest
func New() *Manifest {
	manifest := Manifest{}
	toml.Decode(newPackageTemplate, &manifest)
	return &manifest
}

// ResolvedMod is a mod that can be downloaded
type ResolvedMod struct {
	Slug        string
	DownloadURL string
	FileName    string
}

// LocalName returns the name that should be used on disk
func (r *ResolvedMod) LocalName() string {
	if strings.HasSuffix(r.FileName, ".jar") {
		return r.FileName
	}

	return r.FileName + ".jar"
}

func fetchExtends(url string) (*Manifest, error) {
	res, err := http.Get(url)
	b, err := ioutil.ReadAll(res.Body)

	m := Manifest{}
	err = toml.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
