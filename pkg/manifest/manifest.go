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

const (
	// TypeMod indicates a package containing a single mod
	TypeMod = "mod"
	// TypeModpack indicates a package containing a list of mods (modpack)
	TypeModpack = "modpack"
)

// Manifest is a collection of data that describes a mod a modpack
type Manifest struct {
	// ManifestVersion specifies the format version
	// This field is REQUIRED
	ManifestVersion int `toml:"manifestVersion"`
	Package         struct {
		Description string `toml:"description"`
		// Name is the name of the package. It may NOT include spaces. It may ONLY consist of
		// alphanumeric chars but can also include `-` and `_`
		// Should be unique. (This will be enforced by the minepkg api)
		// This field is REQUIRED
		Name string `toml:"name"`
		// Type should be one of `TypeMod` ("mod") or `TypeModpack` ("modpack")
		Type string `toml:"type"`
		// Version is the version number of this package. A preceeding `v` (like `v2.1.1`) should NOT
		// be allowed for consistency
		// The version may include prerelease information like `1.2.2-beta.0` or build
		// related information `1.2.1+B7382-2018`.
		Version  string   `toml:"version"`
		Provides []string `toml:"provides"`
		Extends  []string `toml:"extends"`
	} `toml:"package"`
	Requirements struct {
		// Minecraft is a semver version string describing the required Minecraft version
		// The Minecraft version is binding and implementers should not install
		// Mods for non-matching Minecraft versions.
		// Modpack & Mod Authors are encuraged to use semver to allow a broader install range.
		// Plain version numbers just default to the `~` semver operator here. Allowing patches but not minor or major versions.
		// So `1.12.0` and `~1.12.0` are equal
		// This field is REQUIRED
		Minecraft string `toml:"minecraft"`
		// Forge is the minimum forge version required
		// no semver here, because forge does not follow semver
		Forge string `toml:"forge,omitempty"`
		// Fabric  is a semver version string describing the required Fabric version
		// Only one of `Forge` or `Fabric` may be used
		Fabric string `toml:"fabric,omitempty"`
	} `toml:"requirements"`
	// Dependencies lists runtime dependencies of this package
	Dependencies `toml:"dependencies"`
	// Hooks should help mod developers to ease publishing
	Hooks struct {
		Build string `toml:"build,omitempty"`
	} `toml:"hooks"`
}

// Dependency defines a dependency that can be saved and installed from
// the minepkg.toml
type Dependency interface {
	Identifier() string // Identifier is the (unique) name of the dependency
	fmt.Stringer        // used as `data` in `name = "[data]"` inside the minepkg.toml
}

// Dependencies are the dependencies of a mod or modpack
type Dependencies map[string]string

// ParsedDependency is returned by `Parsed` to make installing
// Dependencies easier
type ParsedDependency struct {
	Provider string
	Target   string
	Meta     string
}

// Parsed returns a `ParsedDependency` slice including all dependencies
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
