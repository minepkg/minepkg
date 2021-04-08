package api

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/Masterminds/semver/v3"
)

// RequirementQuery is a query for a release describing contained requirements
type RequirementQuery struct {
	// Version to return. this can be any semver string
	Version string
	// Minecraft version for the project. This has to be either '*' or a valid version number.
	// any semver string is NOT allowed here
	Minecraft string
	// Platform can bei either fabric or forge
	Platform string
}

// ErrInvalidMinecraftRequirement is returned if an invalid minecraft requirement was passed
var ErrInvalidMinecraftRequirement = errors.New("minecraft requirement is invalid. Only * or a version number is allowed. No semver")

// ErrNoMatchingRelease is returned if a wanted releaseendency (package) could not be resolved given the requirements
type ErrNoMatchingRelease struct {
	// Package is the name of the package that can not be resolved
	Package string
	// Requirements are the requirements for this package to resolve (eg. minecraft version)
	Requirements *RequirementQuery
	// Err is the underlying error that describes why there is no matching release
	Err error
}

var (
	ErrNoReleasesForPlatform        = errors.New("project has no releases for this platform")
	ErrProjectDoesNotExist          = errors.New("project does not exist")
	ErrNoReleaseForMinecraftVersion = errors.New("no release exists for the wanted Minecraft version")
	ErrNoReleaseForVersion          = errors.New("no release exists for the wanted Version")
	ErrNoReleaseWithConstrains      = errors.New("no release exists for the given version/minecraft requirement")
)

func (e *ErrNoMatchingRelease) Error() string {
	return fmt.Sprintf("No release found for package \"%s\" with given requirements", e.Package)
}

// FindRelease gets the latest release matching the passed requirements via `RequirementQuery`
func (m *MinepkgAPI) FindRelease(ctx context.Context, project string, reqs *RequirementQuery) (*Release, error) {
	p := Project{client: m, Name: project}

	var wantedMCSemver *semver.Version
	if reqs.Minecraft != "*" {
		var err error
		wantedMCSemver, err = semver.NewVersion(reqs.Minecraft)
		if err != nil {
			return nil, ErrInvalidMinecraftRequirement
		}
	}

	wantedVersion := reqs.Version

	releases, err := p.GetReleases(ctx, reqs.Platform)
	if err != nil {
		if err == ErrNotFound {
			return nil, &ErrNoMatchingRelease{Package: project, Requirements: reqs, Err: ErrProjectDoesNotExist}
		}
		return nil, err
	}

	// found nothing
	if len(releases) == 0 {
		return nil, &ErrNoMatchingRelease{Package: project, Requirements: reqs, Err: ErrNoReleasesForPlatform}
	}

	testedReleases := make([]*Release, 0, len(releases))

	// find all tested & working releases
	for _, release := range releases {
		if release.testedFor(wantedMCSemver) {
			testedReleases = append(testedReleases, release)
		}
	}

	// TODO: handle prereleases
	if wantedVersion == "latest" || wantedVersion == "*" {
		// return the latest working version
		if len(testedReleases) != 0 {
			return testedReleases[0], nil
		}

		// just get the latest latest version
		if reqs.Minecraft == "*" {
			return releases.Latest(), nil
		}

		// get the latest version that matches the wanted minecraft version
		for _, release := range releases {
			if release.compatWith(wantedMCSemver) {
				return release, nil
			}
		}

		return nil, &ErrNoMatchingRelease{Package: project, Requirements: reqs, Err: ErrNoReleaseForMinecraftVersion}
	}

	versionConstraint, err := semver.NewConstraint(wantedVersion)
	if err != nil {
		return nil, err
	}

	// search for tested releases first
	for _, release := range testedReleases {
		if versionConstraint.Check(release.SemverVersion()) {
			return release, nil
		}
	}

	mcCompatCount := 0
	versionCompatCount := 0

	// fallback to search all releases
	for _, release := range releases {
		mcCompatible := release.compatWith(wantedMCSemver)
		versionCompatible := release.compatWith(wantedMCSemver)

		if mcCompatible && versionCompatible {
			return release, nil
		}

		// version is no compatible, but maybe parts of it?
		if mcCompatible {
			mcCompatCount += 1
		}
		if versionCompatible {
			versionCompatCount += 1
		}
	}

	err = ErrNoReleaseWithConstrains
	// no compatible mc releases, but compatible versions
	if mcCompatCount == 0 && versionCompatCount != 0 {
		// wanted mc version is the problem
		err = ErrNoReleaseForMinecraftVersion
	}

	// compatible mc releases, but version does not match
	if mcCompatCount != 0 && versionCompatCount == 0 {
		// wanted mc version is the problem
		err = ErrNoReleaseForVersion
	}

	// found nothing
	return nil, &ErrNoMatchingRelease{Package: project, Requirements: reqs, Err: err}
}

// testedFor returns true if this release was tested worked for the given minecraft version
func (r *Release) testedFor(mcVersion *semver.Version) bool {

	// precondition (release requirement is compatible) failed
	if !r.compatWith(mcVersion) {
		return false
	}

	for _, test := range r.Tests {
		if test.worksWithMCVersion(mcVersion) {
			return true
		}
	}
	return false
}

// compatWith returns true if this release requirement is compatible with the given minecraft version
func (r *Release) compatWith(mcVersion *semver.Version) bool {
	modMcConstraint, err := semver.NewConstraint(r.Requirements.Minecraft)
	if err != nil {
		// TODO: maybe this should be an error
		fmt.Printf(
			"%s has invalid minecraft requirement: %s\n",
			r.Identifier(),
			r.Requirements.Minecraft,
		)
		return false
	}

	return mcVersion == nil || modMcConstraint.Check(mcVersion)
}

// ReleaseList is a slice of releases with a helper function
type ReleaseList []*Release

// Latest returns the latest release from the release list based on the semver version number
func (r *ReleaseList) Latest() *Release {
	releases := *r
	// no releases yet
	if len(releases) == 0 {
		return nil
	}

	// sort version strings
	versions := make([]*semver.Version, 0, len(releases))
	for i, r := range releases {
		versions[i] = r.SemverVersion()
	}

	sort.Sort(semver.Collection(versions))

	latestV := versions[0]
	for _, r := range releases {
		if r.Package.Version == latestV.Original() {
			return r
		}
	}
	panic("FindLatestRelease internal error. Could not find release again")
}
