/*
Package manifest defines the file format that describes a mod or modpack.
The "minepkg.toml" manifest is the way minepkg defines packages. Packages can be mods or modpacks.

Learn More

For more details visit https://minepkg.io/docs/manifest
*/
package manifest

import (
	"bytes"
	"log"
	"net/mail"

	"github.com/pelletier/go-toml"
)

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
	ManifestVersion int `toml:"manifestVersion" comment:"Preview of the minepkg.toml format! Could break anytime!" json:"manifestVersion"`
	Package         struct {
		// Type should be one of `TypeMod` ("mod") or `TypeModpack` ("modpack")
		// this field is REQUIRED
		Type string `toml:"type" json:"type"`
		// Name is the name of the package. It may NOT include spaces. It may ONLY consist of
		// alphanumeric chars but can also include `-` and `_`
		// Should be unique. (This will be enforced by the minepkg api)
		// This field is REQUIRED
		Name string `toml:"name" json:"name"`
		// Description can be any free text that describes this package. Should be short(ish)
		Description string `toml:"description" json:"description"`
		// Version is the version number of this package. A proceeding `v` (like `v2.1.1`) is NOT
		// allowed to preserve consistency
		// The version may include prerelease information like `1.2.2-beta.0` or build
		// related information `1.2.1+B7382-2018`.
		// The version can be omitted. Publishing will require a version number as flag in that case
		Version string `toml:"version,omitempty" json:"version,omitempty"`
		// Platform indicates the supported playform of this package. can be `fabric`, `forge` or `vanilla`
		Platform string `toml:"platform,omitempty" json:"platform,omitempty"`
		// Licence for this project. Should be a valid SPDX identifier if possible
		// see https://spdx.org/licenses/
		License string `toml:"license,omitempty" json:"license,omitempty"`
		// Source is an URL that should point to the source code repository of this package (if any)
		Source string `toml:"source,omitempty" json:"source,omitempty"`
		// Homepage is an URL that should point to the website of this package (if any)
		Homepage string `toml:"homepage,omitempty" json:"homepage,omitempty"`
		// Author in the form of "Full Name <email@example.com>". Email can be omitted and Full Name does not have to be a real name
		Author string `toml:"author,omitempty" json:"author,omitempty"`
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
		// FabricLoader is a semver version string describing the required FabricLoader version
		// Only one of `Forge` or `FabricLoader` may be used
		FabricLoader string `toml:"fabricLoader,omitempty" json:"fabricLoader,omitempty"`
		// ForgeLoader is the minimum forge version required
		// no semver here, because forge does not follow semver
		ForgeLoader string `toml:"forgeLoader,omitempty" json:"forgeLoader,omitempty"`
		// MinepkgCompanion is the version of the minepkg companion plugin that is going to be added to modpacks.
		// This has no effect on other types of packages
		// `latest` is assumed if this field is omitted. `none` can be used to exclude the companion
		// plugin from a modpack – but this is not recommended
		MinepkgCompanion string `toml:"minepkgCompanion,omitempty" json:"minepkgCompanion,omitempty"`
		// old names, are only here for migration
		Fabric string `toml:"fabric,omitempty" json:"fabric,omitempty"`
		Forge  string `toml:"forge,omitempty" json:"forge,omitempty"`
	} `toml:"requirements" comment:"These are global requirements" json:"requirements"`
	// Dependencies lists runtime dependencies of this package
	// this list can contain mods and modpacks
	Dependencies `toml:"dependencies" json:"dependencies,omitempty"`
	// Dev contains development & testing related options
	Dev struct {
		// BuildCommand is the command used for building this package (usually "./gradlew build")
		BuildCommand string `toml:"buildCommand,omitempty" json:"buildCommand,omitempty"`
		// Jar defines the target location glob to the built jar file for mods. Can be empty. Example: "lib/builds/my-mod-v*.jar"
		Jar string `toml:"jar,omitempty" json:"jar,omitempty"`
		// Dependencies inside the dev struct should only be installed if this package is defined locally.
		// They should never be installed for published packages
		Dependencies `toml:"dependencies,omitempty" json:"dependencies,omitempty"`
	} `toml:"dev" json:"dev"`
}

// Dependencies are the dependencies of a mod or modpack as a map
type Dependencies map[string]string

// PlatformString returns the required platform as a string (vanilla, fabric or forge)
func (m *Manifest) PlatformString() string {
	switch {
	case m.Requirements.FabricLoader != "":
		return "fabric"
	case m.Requirements.ForgeLoader != "":
		return "forge"
	default:
		return "vanilla"
	}
}

// PlatformVersion returns the required platform version
func (m *Manifest) PlatformVersion() string {
	switch {
	case m.Requirements.FabricLoader != "":
		return m.Requirements.FabricLoader
	case m.Requirements.ForgeLoader != "":
		return m.Requirements.ForgeLoader
	default:
		return ""
	}
}

// AuthorName returns the name of the author (excluding the email address if set)
func (m *Manifest) AuthorName() string {
	parsed, err := mail.ParseAddress(m.Package.Author)
	if err == nil {
		return parsed.Name
	}
	return m.Package.Author
}

// AuthorEmail returns the email of the author or empty string if none is set
func (m *Manifest) AuthorEmail() string {
	parsed, err := mail.ParseAddress(m.Package.Author)
	if err == nil {
		return parsed.Address
	}
	return ""
}

// AddDependency adds a new dependency to the manifest
func (m *Manifest) AddDependency(name string, version string) {
	// remove from dev dependencies
	m.RemoveDevDependency(name)
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

// AddDevDependency adds a new dev dependency to the manifest
func (m *Manifest) AddDevDependency(name string, version string) {
	// remove from normal dependencies
	m.RemoveDependency(name)
	if m.Dev.Dependencies == nil {
		m.Dev.Dependencies = make(map[string]string)
	}
	m.Dev.Dependencies[name] = version
}

// RemoveDevDependency removes a dev dependency from the manifest
func (m *Manifest) RemoveDevDependency(name string) {
	if m.Dev.Dependencies == nil {
		m.Dev.Dependencies = make(map[string]string)
	}
	delete(m.Dependencies, name)
}

// Buffer returns the manifest as toml in Buffer form
func (m *Manifest) Buffer() *bytes.Buffer {
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Order(toml.OrderPreserve).Encode(m); err != nil {
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
	manifest.Dependencies = make(Dependencies)
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
