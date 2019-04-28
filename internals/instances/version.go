package instances

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
)

var (
// ErrorNoVersion is returned if no mc version was detected
// ErrorNoVersion = errors.New("Could not detect minecraft version")
)

// AvailableVersions returns all available versions sorted highest first
// TODO: return error
func (m *McInstance) AvailableVersions() semver.Collection {
	path := filepath.Join(m.Directory, "versions")
	entries, _ := ioutil.ReadDir(path)

	versions := make(semver.Collection, len(entries))
	for i, entry := range entries {
		cleanedUp := strings.Split(entry.Name(), "-forge")[0] // just cut away forge version for now
		v, err := semver.NewVersion(cleanedUp)
		if err != nil {
			panic("versions folder contains invalid version: " + entry.Name())
		}
		versions[i] = v
	}
	sort.Sort(sort.Reverse(versions))
	return versions
}

// Version returns the minecraft version of the instance
func (m *McInstance) Version() *semver.Version {
	switch m.Flavour {
	case FlavourVanilla:
		versions := m.AvailableVersions()

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

type vanillaVersion struct {
	mc    *semver.Version
	forge string
}
