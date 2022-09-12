package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/pelletier/go-toml"
)

// LockfileVersion is the current version of the lockfile template
const LockfileVersion = 1

var (
	// ErrDependencyConflicts is returned when trying to add a dependency that is already present
	ErrDependencyConflicts = errors.New("a dependency with that name is already present")

	// DependencyLockTypeMod describes a mod dependency
	DependencyLockTypeMod = "mod"
	// DependencyLockTypeModpack describes a modpack dependency
	DependencyLockTypeModpack = "modpack"

	PlatformFabric  = "fabric"
	PlatformForge   = "forge"
	PlatformVanilla = "vanilla"
)

// PlatformLock describes a queryable platform (fabric, forge)
type PlatformLock interface {
	PlatformName() string
	MinecraftVersion() string
	PlatformVersion() string
}

type PlatformRequirement struct {
	Minecraft     string `toml:"minecraft"`
	LoaderName    string `toml:"loader"`
	LoaderVersion string `toml:"loader_version"`
}

func (p *PlatformRequirement) MinecraftVersion() string {
	return p.Minecraft
}

func (p *PlatformRequirement) PlatformName() string {
	return p.LoaderName
}

func (p *PlatformRequirement) PlatformVersion() string {
	return p.LoaderVersion
}

// Lockfile includes resolved dependencies and requirements
type Lockfile struct {
	LockfileVersion int                        `toml:"lockfileVersion" json:"lockfileVersion"`
	Fabric          *FabricLock                `toml:"fabric,omitempty" json:"fabric,omitempty"`
	Forge           *ForgeLock                 `toml:"forge,omitempty" json:"forge,omitempty"`
	Vanilla         *VanillaLock               `toml:"vanilla,omitempty" json:"vanilla,omitempty"`
	Dependencies    map[string]*DependencyLock `toml:"dependencies,omitempty" json:"dependencies,omitempty"`
}

// FabricLock describes resolved fabric requirements
type FabricLock struct {
	Minecraft    string `toml:"minecraft" json:"minecraft"`
	FabricLoader string `toml:"fabricLoader" json:"fabricLoader"`
	Mapping      string `toml:"mapping" json:"mapping"`
}

// PlatformName returns the string fabric
func (f *FabricLock) PlatformName() string { return "fabric" }

// MinecraftVersion returns the minecraft version
func (f *FabricLock) MinecraftVersion() string { return f.Minecraft }

// PlatformVersion returns the fabric mod loader version
func (f *FabricLock) PlatformVersion() string { return f.FabricLoader }

// VanillaLock describes resolved vanilla requirements
type VanillaLock struct {
	Minecraft string `toml:"minecraft" json:"minecraft"`
}

// PlatformName returns the string vanilla
func (v *VanillaLock) PlatformName() string { return "vanilla" }

// MinecraftVersion returns the minecraft version
func (v *VanillaLock) MinecraftVersion() string { return v.Minecraft }

// PlatformVersion returns ""
func (f *VanillaLock) PlatformVersion() string { return "" }

// ForgeLock describes resolved forge requirements
// this is not used for now, because forge does not provide a API
// to resolve this
type ForgeLock struct {
	Minecraft   string `toml:"minecraft" json:"minecraft"`
	ForgeLoader string `toml:"forgeLoader" json:"forgeLoader"`
}

// PlatformName returns the string vanilla
func (f *ForgeLock) PlatformName() string { return "forge" }

// MinecraftVersion returns the minecraft version
func (f *ForgeLock) MinecraftVersion() string { return f.Minecraft }

// MinecraftVersion returns the forge loader version
func (f *ForgeLock) PlatformVersion() string { return f.ForgeLoader }

// DependencyLock is a single resolved dependency
type DependencyLock struct {
	Name        string `toml:"name" json:"name"`
	Version     string `toml:"version" json:"version"`
	VersionName string `toml:"versionName" json:"versionName"`
	Type        string `toml:"type" json:"type"`
	IPFSHash    string `toml:"ipfsHash" json:"ipfsHash"`
	Sha1        string `toml:"Sha1,omitempty" json:"Sha1,omitempty"`
	Sha256      string `toml:"Sha256,omitempty" json:"Sha256,omitempty"`
	Sha512      string `toml:"Sha512,omitempty" json:"Sha512,omitempty"`
	URL         string `toml:"url" json:"url"`
	// Provider usually is minepkg but can also be https
	Provider string `toml:"provider" json:"provider"`
	// Dependent is the package that requires this mod. can be _root if top package
	Dependent string `toml:"dependent" json:"dependent"`
	// IsDev is true if this is a dev dependency
	IsDev bool `toml:"isDev,omitempty" json:"isDev,omitempty"`
}

// FileExt returns ".jar" for mods and ".zip" for modpacks
func (d *DependencyLock) FileExt() string {
	ending := ".jar"
	if d.Type == DependencyLockTypeModpack {
		ending = ".zip"
	}
	return ending
}

// Filename returns the dependency in the "[name]-[version].jar" format
func (d *DependencyLock) Filename() string {
	return fmt.Sprintf("%s-%s%s", d.Name, d.Version, d.FileExt())
}

// MinecraftVersion returns the Minecraft version
func (l *Lockfile) MinecraftVersion() string {
	switch {
	case l.Fabric != nil:
		return l.Fabric.Minecraft
	case l.Forge != nil:
		return l.Forge.Minecraft
	case l.Vanilla != nil:
		return l.Vanilla.Minecraft
	default:
		panic("lockfile has no fabric, forge or vanila requirement")
	}
}

// PlatformLock returns the platform lock object (fabric, forge or vanilla lock)
func (l *Lockfile) PlatformLock() PlatformLock {
	switch {
	case l.Fabric != nil:
		return l.Fabric
	case l.Forge != nil:
		return l.Forge
	case l.Vanilla != nil:
		return l.Vanilla
	default:
		panic("lockfile has no fabric, forge or vanila requirement")
	}
}

// McManifestName returns the Minecraft Launcher Manifest name
func (l *Lockfile) McManifestName() string {
	switch {
	case l.Fabric != nil:
		return l.Fabric.Minecraft + "-fabric-" + l.Fabric.FabricLoader
	case l.Forge != nil:
		return l.Forge.Minecraft + "-forge-" + l.Forge.ForgeLoader
	case l.Vanilla != nil:
		return l.Vanilla.Minecraft
	default:
		panic("lockfile has no fabric, forge or vanila requirement")
	}
}

// HasRequirements returns true if lockfile has some requirements
func (l *Lockfile) HasRequirements() bool {
	return l.Fabric != nil || l.Forge != nil || l.Vanilla != nil
}

// Buffer returns the manifest as toml in Buffer form
func (l *Lockfile) Buffer() *bytes.Buffer {
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Order(toml.OrderPreserve).Encode(l); err != nil {
		log.Fatal(err)
	}

	bbuf := new(bytes.Buffer)
	bbuf.Write([]byte("# You should not edit this file.\n# It was generated by minepkg.\n\n"))
	bbuf.Write(buf.Bytes())

	return bbuf
}

func (l *Lockfile) String() string {
	return l.Buffer().String()
}

// AddDependency adds a new dependency to the lockfile
func (l *Lockfile) AddDependency(dep *DependencyLock) {
	l.Dependencies[dep.Name] = dep
}

// ClearDependencies removes all dependencies
func (l *Lockfile) ClearDependencies() {
	l.Dependencies = make(map[string]*DependencyLock)
}

// NewLockfile returns a new lockfile
func NewLockfile() *Lockfile {
	manifest := Lockfile{LockfileVersion: LockfileVersion, Dependencies: make(map[string]*DependencyLock)}
	return &manifest
}

// NewFromFile tries to parse a local lockfile file
func NewLockfileFromFile(filename string) (*Lockfile, error) {
	rawLockfile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	lockfile := Lockfile{}
	if err = toml.Unmarshal(rawLockfile, &lockfile); err != nil {
		return nil, err
	}

	return &lockfile, nil
}
