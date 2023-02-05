package api

import (
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/minepkg/minepkg/pkg/manifest"
)

const (
	// TypeMod indicates a mod
	TypeMod = "mod"
	// TypeModpack indicates a modpack
	TypeModpack = "modpack"
)

// User describes a registered user
type User struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

// Project is a project
type Project struct {
	client      *MinepkgClient
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Readme      string `json:"readme,omitempty"`
	Links       struct {
		Source   string `json:"source,omitempty"`
		Homepage string `json:"homepage,omitempty"`
	} `json:"links,omitempty"`
	Stats *struct {
		TotalDownloads uint32 `json:"totalDownloads"`
	} `json:"stats,omitempty"`
	Unofficial bool `json:"unofficial,omitempty"`
}

type ProjectStats struct {
	Summary struct {
		TotalDownloads int `json:"totalDownloads"`
		Releases       int `json:"releases"`
		Likes          int `json:"likes"`
	} `json:"summary"`
	Downloads []struct {
		Downloads int       `json:"downloads"`
		Date      time.Time `json:"date"`
	} `json:"downloads"`
}

// ReleaseMeta is metadata for a release. found in the `meta` field
type ReleaseMeta struct {
	IPFSHash   string     `json:"ipfsHash,omitempty"`
	Sha256     string     `json:"sha256,omitempty"`
	Published  bool       `json:"published"`
	Unofficial *bool      `json:"unofficial,omitempty"`
	CreatedAt  *time.Time `json:"createdAt,omitempty"`
}

// Release is a released version of a project
type Release struct {
	*manifest.Manifest
	client *MinepkgClient
	Meta   *ReleaseMeta           `json:"meta,omitempty"`
	Tests  map[string]ReleaseTest `json:"tests,omitempty"`
}

// NewRelease returns a `Release` object. Only exists locally. Can be used to POST a new release to the API
func (a *MinepkgClient) NewRelease(m *manifest.Manifest) *Release {
	return &Release{
		Manifest: m,
		client:   a,
	}
}

// NewUnpublishedRelease returns a `Release` object that has `Meta.published` set to false.
// should be used if you want to upload an artifact after publishing this release
// Only exists locally. Can be used to POST a new release to the API
func (a *MinepkgClient) NewUnpublishedRelease(m *manifest.Manifest) *Release {
	return &Release{
		Manifest: m,
		client:   a,
		Meta:     &ReleaseMeta{Published: false},
	}
}

// ReleaseTest is a test of the package
type ReleaseTest struct {
	ID        string `json:"_id"`
	Minecraft string `json:"minecraft"`
	Works     bool   `json:"works"`
}

func (rt *ReleaseTest) worksWithMCVersion(mcVersion *semver.Version) bool {
	return rt.Works && (mcVersion == nil || mcVersion.Equal(mcVersion))
}

// Requirements contains the wanted Minecraft version
// and either the required Forge or Fabric version
type Requirements struct {
	Minecraft string `json:"minecraft"`
	Forge     string `json:"forge,omitempty"`
	Fabric    string `json:"fabric,omitempty"`
}

// Dependency in verbose form
type Dependency struct {
	// Provider is only minepkg for now. Kept for future extensions
	Provider string `json:"provider"`
	// Name is the name of the package (eg. storage-drawers)
	Name string `json:"name"`
	// VersionRequirement is a semver version Constraint
	// Example: `^2.9.22` or `5.x.x`
	VersionRequirement string `json:"versionRequirement"`
}

// ForgeVersion is a release of forge
type ForgeVersion struct {
	Branch      string     `json:"branch"`
	Build       int        `json:"build"`
	Files       [][]string `json:"files"`
	McVersion   string     `json:"mcversion"`
	Modified    int        `json:"modified"`
	Version     string     `json:"version"`
	Recommended bool       `json:"recommended"`
}

// MinepkgError is the json response if the response
// was not successful
type MinepkgError struct {
	StatusCode uint16 `json:"statusCode"`
	Status     string `json:"error"`
	Message    string `json:"message"`
	// ResolveError is only present for release queries
	ResolveError string `json:"resolveError"`
}

func (m MinepkgError) Error() string {
	return m.Status + ": " + m.Message
}

// CrashReportPackage is a package in a crash report
type CrashReportPackage struct {
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Version  string `json:"version"`
}

// CrashReportFabricDetail is a the fabric part of the crash report
type CrashReportFabricDetail struct {
	Loader  string `json:"loader"`
	Mapping string `json:"mapping"`
}

// CrashReportForgeDetail is a the forge part of the crash report
type CrashReportForgeDetail struct {
	Loader string `json:"loader"`
}

// CrashReport is a crash report
type CrashReport struct {
	Package          CrashReportPackage       `json:"package"`
	Fabric           *CrashReportFabricDetail `json:"fabric,omitempty"`
	Forge            *CrashReportForgeDetail  `json:"forge,omitempty"`
	MinecraftVersion string                   `json:"minecraftVersion"`
	Server           bool                     `json:"server"`
	Mods             map[string]string        `json:"mods"`
	Logs             string                   `json:"logs,omitempty"`
	OS               string                   `json:"os,omitempty"`
	Arch             string                   `json:"arch,omitempty"`
	JavaVersion      string                   `json:"javaVersion,omitempty"`
	ExitCode         int                      `json:"exitCode,omitempty"`
}
