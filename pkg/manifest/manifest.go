package manifest

import (
	"bytes"
	"log"

	"github.com/BurntSushi/toml"
)

// TODO: get those comments to be in the initial toml written to disk
const newPackageTemplate = `
# WARNING: This package might not work with future versions
#   of minepkg and might need to be updated then.
# This is a preview version of the minepkg.toml format!
manifestVersion = 0

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
	ManifestVersion int `toml:"manifestVersion" json:"manifestVersion"`
	Package         struct {
		// Type should be one of `TypeMod` ("mod") or `TypeModpack` ("modpack")
		Type string `toml:"type" json:"type"`
		// Name is the name of the package. It may NOT include spaces. It may ONLY consist of
		// alphanumeric chars but can also include `-` and `_`
		// Should be unique. (This will be enforced by the minepkg api)
		// This field is REQUIRED
		Name        string `toml:"name" json:"name"`
		Description string `toml:"description" json:"description"`
		// Version is the version number of this package. A preceeding `v` (like `v2.1.1`) should NOT
		// be allowed for consistency
		// The version may include prerelease information like `1.2.2-beta.0` or build
		// related information `1.2.1+B7382-2018`.
		// The version can be omited. In that case minepkg will try to use git tags
		Version string `toml:"version,omitempty" json:"version,omitempty"`
		// Platform incidates the supported playform of this package. can be `fabric`, `forge` or `vanilla`
		Platform string   `toml:"platform,omitempty" json:"platform,omitempty"`
		License  string   `toml:"license,omitempty" json:"license,omitempty"`
		Provides []string `toml:"provides,omitempty" json:"provides,omitempty"`
	} `toml:"package" json:"package"`
	Requirements struct {
		// Minecraft is a semver version string describing the required Minecraft version
		// The Minecraft version is binding and implementers should not install
		// Mods for non-matching Minecraft versions.
		// Modpack & Mod Authors are encuraged to use semver to allow a broader install range.
		// Plain version numbers just default to the `~` semver operator here. Allowing patches but not minor or major versions.
		// So `1.12.0` and `~1.12.0` are equal
		// This field is REQUIRED
		Minecraft string `toml:"minecraft" json:"minecraft"`
		// Fabric is a semver version string describing the required Fabric version
		// Only one of `Forge` or `Fabric` may be used
		Fabric string `toml:"fabric,omitempty" json:"fabric,omitempty"`
		// Forge is the minimum forge version required
		// no semver here, because forge does not follow semver
		Forge string `toml:"forge,omitempty" json:"forge,omitempty"`
		// MinepkgCompanion is the version of the minepkg companion plugin that is going to be added to modpacks.
		// This has no effect on other types of packages
		// `latest` is assumed if this field is omited. `none` can be used to exclude the companion
		// plugin from a modpack – but this is not recommended
		MinepkgCompanion string `toml:"minepkgCompanion,omitempty" json:"minepkgCompanion,omitempty"`
	} `toml:"requirements" json:"requirements"`
	// Dependencies lists runtime dependencies of this package
	// this list can contain mods and modpacks
	Dependencies `toml:"dependencies" json:"dependencies,omitempty"`
	// Hooks should help mod developers to ease publishing
	Hooks struct {
		Build string `toml:"build,omitempty" json:"build,omitempty"`
	} `toml:"hooks" json:"hooks"`
}

// Dependencies are the dependencies of a mod or modpack
type Dependencies map[string]string

// PlatformString returns the required platform as a string (vanilla, fabric or forge)
func (m *Manifest) PlatformString() string {
	switch {
	case m.Requirements.Fabric != "":
		return "fabric"
	case m.Requirements.Forge != "":
		return "forge"
	default:
		return "vanilla"
	}
}

// PlatformVersion returns the required platform version
func (m *Manifest) PlatformVersion() string {
	switch {
	case m.Requirements.Fabric != "":
		return m.Requirements.Fabric
	case m.Requirements.Forge != "":
		return m.Requirements.Forge
	default:
		return ""
	}
}

// AddDependency adds a new dependency to the manifest
func (m *Manifest) AddDependency(name string, version string) {
	if m.Dependencies == nil {
		m.Dependencies = make(map[string]string)
	}
	m.Dependencies[name] = version
}

// RemoveDependency removes a dependecy from the manifest
func (m *Manifest) RemoveDependency(name string) {
	if m.Dependencies == nil {
		m.Dependencies = make(map[string]string)
	}
	delete(m.Dependencies, name)
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

// NewInstanceLike takes an existing manifest and copies most package
func NewInstanceLike(from *Manifest) *Manifest {
	manifest := New()
	// TODO: this feels like a hack
	// maybe introduce a Package.Type instance ?
	manifest.Package.Name = "_instance-" + from.Package.Name
	manifest.Package.Description = from.Package.Description
	manifest.Package.Type = from.Package.Type
	manifest.Package.Platform = from.Package.Platform

	manifest.Requirements = from.Requirements

	// set this instance as first dependency
	manifest.Dependencies[from.Package.Name] = from.Package.Version
	return manifest
}
