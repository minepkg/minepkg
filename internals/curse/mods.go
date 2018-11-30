package curse

import (
	"sort"
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
	PackageType   uint32    `json:packageType`
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
	for _, mod := range m {
		for _, v := range mod.GameVersion {
			if v == version {
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

type Release struct {
	FileType    string `json:"releaseType"`
	ID          uint32 `json:"projectFileID"`
	GameVersion string `json:"gameVersion"`
}

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

type ModDependency struct {
	AddOnID uint32 `json:"addonId"`
	Type    uint8  `json:"type"`
}
