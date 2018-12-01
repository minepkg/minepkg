package curse

import (
	"sort"
	"unicode"

	"github.com/Masterminds/semver"
)

var (
	// ReleaseTypeStable indicates a stable release
	ReleaseTypeStable uint8 = 1
	// ReleaseTypeBeta indicates a beta release
	ReleaseTypeBeta uint8 = 2
	// ReleaseTypeAlpha indicates a alpha release
	ReleaseTypeAlpha uint8 = 3

	// PackageTypeModpack indicates a modpack
	PackageTypeModpack uint8 = 5
	// PackageTypeMod indicates a single mod
	PackageTypeMod uint8 = 6

	// DependencyTypeRequired indicates a required dependency
	DependencyTypeRequired uint8 = 1
	// DependencyTypeOptional indicates a optional dependency
	// In the ui this sometimes appears as a "tool" dependency.
	// this can usually be ignored
	DependencyTypeOptional uint8 = 2
	// DependencyTypeEmbedded indicates a embedded dependency
	// this can usually be ignored
	DependencyTypeEmbedded uint8 = 3
)

// Mod todo
type Mod struct {
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	ID            uint32    `json:"id"`
	WebsiteURL    string    `json:"webSiteURL"`
	DownloadCount float32   `json:"downloadCount"`
	LastReleases  []Release `json:"gameVersionLatestFiles"`
	PackageType   uint8     `json:"packageType"`
}

// Identifier returns the Slug, for the minepkg.toml
func (m *Mod) Identifier() string {
	return m.Slug
}

// String returns a string that can be used to install this mod again
func (m *Mod) String() string {
	return "curse:" + idToString(m.ID)
}

// FindRelease returns the latest downloadable version for the given version
func FindRelease(m []ModFile, version string) *ModFile {
	// prepend default ~ to plain version numbers
	// TODO: do not do this here
	if unicode.IsDigit(rune(version[0])) {
		version = "~" + version
	}
	constraint, err := semver.NewConstraint(version)
	if err != nil {
		panic("invalid minecraft version requirement in minepkg.toml: " + version)
	}
	// sort by newest id (hack to sort by date)
	sort.Slice(m, func(i, j int) bool {
		return m[i].ID > m[j].ID
	})

	for _, mod := range m {
		for _, v := range mod.GameVersion {
			version := semver.MustParse(v)
			if constraint.Check(version) {
				return &mod
			}
		}
	}
	return nil
}

// Filter returns filtered mods
func Filter(m []Mod, f func(Mod) bool) (mods []Mod) {
	for _, v := range m {
		if f(v) {
			mods = append(mods, v)
		}
	}
	return mods
}

// SortByDownloadCount sorty the given slice by download count
func SortByDownloadCount(m []Mod) {
	less := func(i, j int) bool { return m[i].DownloadCount > m[j].DownloadCount }
	sort.Slice(m, less)
}

// Release is a curseforge Release of a package (not used for the most part)
type Release struct {
	FileType    string `json:"releaseType"`
	ID          uint32 `json:"projectFileID"`
	GameVersion string `json:"gameVersion"`
}

// ModFile is a released version of a given package
type ModFile struct {
	ID             uint32          `json:"id"`
	FileName       string          `json:"fileName"`
	FileNameOnDisk string          `json:"fileNameOnDisk"`
	ReleaseType    uint8           `json:"releaseType"`
	DownloadURL    string          `json:"downloadUrl"`
	IsAlternate    bool            `json:"isAlternate"`
	Dependencies   []ModDependency `json:"dependencies"`
	GameVersion    []string        `json:"gameVersion"`
}

// ModDependency is a dependency required for a given ModFile
type ModDependency struct {
	AddOnID uint32 `json:"addonId"`
	Type    uint8  `json:"type"`
}
