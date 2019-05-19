package manifest

import (
	"bytes"
	"log"

	"github.com/BurntSushi/toml"
)

// LockfileVersion is the current version of the lockfile template
const LockfileVersion = 1

// Lockfile includes resolved dependencies and requirements
type Lockfile struct {
	LockfileVersion int          `toml:"lockfileVersion" json:"lockfileVersion"`
	Fabric          *FabricLock  `toml:"fabric,omitempty" json:"fabric,omitempty"`
	Forge           *ForgeLock   `toml:"forge,omitempty" json:"forge,omitempty"`
	Vanilla         *VanillaLock `toml:"vanilla,omitempty" json:"vanilla,omitempty"`
	Dependencies    `toml:"dependencies,omitempty" json:"dependencies,omitempty"`
}

// FabricLock describes resolved fabric requirements
type FabricLock struct {
	Minecraft    string `toml:"minecraft" json:"minecraft"`
	FabricLoader string `toml:"fabricLoader" json:"fabricLoader"`
	Mapping      string `toml:"mapping" json:"mapping"`
}

// VanillaLock describes resolved vanilla requirements
type VanillaLock struct {
	Minecraft string `toml:"minecraft" json:"minecraft"`
}

// ForgeLock describes resolved forge requirements
// this is not used for now, because forge does not provide a API
// to resolve this
type ForgeLock struct {
	Minecraft   string `toml:"minecraft" json:"minecraft"`
	ForgeLoader string `toml:"forgeLoader" json:"forgeLoader"`
}

// DependencyLock is a single resolved dependency
type DependencyLock struct {
	Project  string `toml:"project" json:"project"`
	Version  string `toml:"version" json:"version"`
	IPFSHash string `toml:"ipfsHash" json:"ipfsHash"`
	Sha1     string `toml:"sha1" json:"sha1"`
}

// McManifestName returns the Minecraft Launcher Manifest name
func (l *Lockfile) McManifestName() string {
	switch {
	case l.Fabric != nil:
		return l.Fabric.Minecraft + "-fabric-" + l.Fabric.FabricLoader
	case l.Forge != nil:
		return l.Forge.Minecraft + "-forge-" + l.Forge.ForgeLoader
	default:
		return l.Vanilla.Minecraft
	}
}

// Buffer returns the manifest as toml in Buffer form
func (l *Lockfile) Buffer() *bytes.Buffer {
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(l); err != nil {
		log.Fatal(err)
	}
	return buf
}

func (l *Lockfile) String() string {
	return l.Buffer().String()
}

// NewLockfile returns a new lockfile
func NewLockfile() *Lockfile {
	manifest := Lockfile{LockfileVersion: LockfileVersion}
	return &manifest
}
