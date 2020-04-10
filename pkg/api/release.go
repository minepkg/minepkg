package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/Masterminds/semver"
)

// ErrNotMatchingRelease gets returned if no matching release was found
var ErrNotMatchingRelease = errors.New("No matching release found for this dependency")

func (r *Release) decorate(c *MinepkgAPI) {
	r.client = c
}

// Identifier returns this release in a "project@version" format. eg: `fabric@0.2.0`
func (r *Release) Identifier() string {
	return r.Package.Name + "@" + r.Package.Version
}

// Filename returns this release in a "project@version.jar" format. eg: `fabric@0.2.0.jar`
func (r *Release) Filename() string {
	return r.Identifier() + ".jar"
}

// DownloadURL returns the download url for this release
func (r *Release) DownloadURL() string {
	// TODO: works but is kind of wonky
	if r.Meta.Sha256 == "" {
		return ""
	}
	return fmt.Sprintf("%s/releases/%s/%s/download", baseAPI, r.Package.Platform, r.Identifier())
}

// LatestTestedMinecraftVersion returns the last (highest) tested Minecraft version for this release
func (r *Release) LatestTestedMinecraftVersion() string {

	workingMcVersion := semver.Collection{}
	// check all tests of this release for matching mc version that works
	for _, test := range r.Tests {
		if test.Works {
			workingMcVersion = append(workingMcVersion, semver.MustParse(test.Minecraft))
		}
	}

	sort.Sort(workingMcVersion)
	return workingMcVersion[len(workingMcVersion)-1].String()
}

// Upload uploads the jar or zipfile for a release
func (r *Release) Upload(reader io.Reader) (*Release, error) {
	// prepare request
	client := r.client

	url := fmt.Sprintf("%s/releases/%s/%s/upload", baseAPI, r.Package.Platform, r.Identifier())
	req, err := http.NewRequest("POST", url, reader)
	if err != nil {
		return nil, err
	}

	client.decorate(req)
	req.Header.Set("Content-Type", "application/java-archive")

	// execute request and handle response
	res, err := client.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(res); err != nil {
		return nil, err
	}

	// parse body
	release := Release{}
	if err := parseJSON(res, &release); err != nil {
		return nil, err
	}

	return &release, nil
}

// GetRelease gets a single release from a project
// `identifier` is a project@version string
func (m *MinepkgAPI) GetRelease(ctx context.Context, platform string, identifier string) (*Release, error) {
	res, err := m.get(ctx, baseAPI+"/releases/"+platform+"/"+identifier)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(res); err != nil {
		return nil, err
	}

	release := Release{}
	if err := parseJSON(res, &release); err != nil {
		return nil, err
	}
	release.decorate(m)

	return &release, nil
}

// GetReleaseList gets a all available releases for a project
func (m *MinepkgAPI) GetReleaseList(ctx context.Context, project string) ([]*Release, error) {
	p := Project{client: m, Name: project}
	return p.GetReleases(ctx, "")
}

// RequirementQuery is a query for a release describing contained requirements
type RequirementQuery struct {
	Version   string
	Minecraft string
	Plattform string
}

// FindRelease gets the latest release matching the passed requirements via `RequirementQuery`
func (m *MinepkgAPI) FindRelease(ctx context.Context, project string, reqs *RequirementQuery) (*Release, error) {
	p := Project{client: m, Name: project}

	wantedVersion := reqs.Version
	mcConstraint, err := semver.NewConstraint(reqs.Minecraft)
	releases, err := p.GetReleases(ctx, reqs.Plattform)
	if err != nil {
		return nil, err
	}

	// found nothing
	if len(releases) == 0 {
		return nil, ErrNotMatchingRelease
	}

	testedReleases := make([]*Release, 0, len(releases))

	// find all tested & working releases
	for _, release := range releases {
		// check all tests of this release for matching mc version that works
		for _, test := range release.Tests {
			mcVersion := semver.MustParse(test.Minecraft)
			if mcConstraint.Check(mcVersion) == true && test.Works {
				testedReleases = append(testedReleases, release)
			}
		}
	}

	// TODO: handle prereleases
	if wantedVersion == "latest" || wantedVersion == "*" {
		// return the latest working version
		if len(testedReleases) != 0 {
			return testedReleases[0], nil
		}
		return releases[0], nil
	}

	versionConstraint, err := semver.NewConstraint(wantedVersion)
	if err != nil {
		return nil, err
	}

	// seach for tested releases first
	for _, release := range testedReleases {
		if versionConstraint.Check(release.SemverVersion()) == true {
			return release, nil
		}
	}

	// fallback to search all releases
	for _, release := range releases {
		if versionConstraint.Check(release.SemverVersion()) == true {
			return release, nil
		}
	}

	// found nothing
	return nil, ErrNotMatchingRelease
}
