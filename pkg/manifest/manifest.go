/*
Package manifest defines the file format that describes a mod or modpack.
The `minepkg.toml` manifest is the way minepkg defines packages. Packages can be mods or modpacks.

Structure

Example manifest.toml:

  manifestVersion = 0
  [package]
    type = "modpack"
    name = "examplepack"
    version = "0.1.0"
    authors = ["John Doe <example@example.com>"]

  [requirements]
    minecraft = "1.14.x"
    # Only fabric OR forge can be set. never both
    fabric = "0.1.0"
    # forge = "0.1.0"

  [dependencies]
    # semver version range
    rftools = "~1.4.2"

    # exactly this version
    ender-io = "=1.0.2"

    # any version (you usually define the version instead)
    # * is equal to "latest" for minepkg. So it will try to fetch the latest
    # version that works
    some-modpack = "*"

  [dev]
    buildCommand = "gradle build"

Dependencies

The dependencies map (map[string]string) contains all dependencies of a package.

The "key" always is the package name. For example `test-utils`

The "value" usually is a semver version number, like this: `^1.4.2`
This will allow any update except major versions.

The following semver formats are allowed:

  test-utils = "^1.0.0" # caret operator (default)
  test-utils = "~1.0.0" # tilde operator
  test-utils = "2.0.1-beta.2" # prerelease
  test-utils = "1.0.0 - 3.0.0" # range
  test-utils = "1.x.x" # range
https://github.com/npm/node-semver#ranges provides a good explanation of the operators mentioned above

Other sources may be specified by using the `source:` syntax in the "value" like this: `raw:https://example.com/package.zip`.
The "key" will still be the package name when using the source syntax.

For more details visit https://minepkg.io/docs/manifest
*/
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
name = "example-pack"
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
		// Version is the version number of this package. A proceeding `v` (like `v2.1.1`) is NOT
		// allowed to preserve consistency
		// The version may include prerelease information like `1.2.2-beta.0` or build
		// related information `1.2.1+B7382-2018`.
		// The version can be omitted. Publishing will require a version number as flag in that case
		Version string `toml:"version,omitempty" json:"version,omitempty"`
		// Platform indicates the supported playform of this package. can be `fabric`, `forge` or `vanilla`
		Platform string   `toml:"platform,omitempty" json:"platform,omitempty"`
		License  string   `toml:"license,omitempty" json:"license,omitempty"`
		Provides []string `toml:"provides,omitempty" json:"provides,omitempty"`
		// BasedOn can be a another package that this one is based on.
		// Most notably, this field is used for instances to determine what modpack is actually running
		// This field is striped when publishing the package to the minepkg api
		BasedOn string `toml:"basedOn,omitempty" json:"basedOn,omitempty"`
		// Savegame can be the name of the primary savegame on this modpack. Not applicable for other package types.
		// This savegame will be used when launching this package via `minepkg try`.
		// This should be the folder name of the savegame
		Savegame string `toml:"build,omitempty" json:"build,omitempty"`
	} `toml:"package" json:"package"`
	Requirements struct {
		// Minecraft is a semver version string describing the required Minecraft version
		// The Minecraft version is binding and implementers should not install
		// Mods for non-matching Minecraft versions.
		// Modpack & Mod Authors are encouraged to use semver to allow a broader install range.
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
		// `latest` is assumed if this field is omitted. `none` can be used to exclude the companion
		// plugin from a modpack – but this is not recommended
		MinepkgCompanion string `toml:"minepkgCompanion,omitempty" json:"minepkgCompanion,omitempty"`
	} `toml:"requirements" json:"requirements"`
	// Dependencies lists runtime dependencies of this package
	// this list can contain mods and modpacks
	Dependencies `toml:"dependencies" json:"dependencies,omitempty"`
	// Dev contains development & testing related options
	Dev struct {
		// BuildCommand is the command used for building this package (usually "./gradlew build")
		BuildCommand string `toml:"buildCommand,omitempty" json:"buildCommand,omitempty"`
	} `toml:"dev" json:"dev"`
}

// Dependencies are the dependencies of a mod or modpack as a map
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

// RemoveDependency removes a dependency from the manifest
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

// NewInstanceLike takes an existing manifest and copies most fields
func NewInstanceLike(from *Manifest) *Manifest {
	manifest := New()
	// TODO: this feels like a hack
	// maybe introduce a Package.Type instance ?
	manifest.Package.Name = "_instance-" + from.Package.Name
	manifest.Package.Description = from.Package.Description
	manifest.Package.Type = from.Package.Type
	manifest.Package.Platform = from.Package.Platform
	manifest.Package.Version = from.Package.Version

	// this is a reference to the original manifest
	manifest.Package.BasedOn = from.Package.Name

	manifest.Requirements = from.Requirements

	// set this instance as first dependency
	manifest.Dependencies[from.Package.Name] = from.Package.Version
	return manifest
}
