package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/Masterminds/semver/v3"
	"github.com/minepkg/minepkg/pkg/manifest"
)

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
	return fmt.Sprintf("%s/releases/%s/%s/download", r.client.APIUrl, r.Package.Platform, r.Identifier())
}

// SemverVersion returns the Version as a `semver.Version` struct
func (r *Release) SemverVersion() *semver.Version {
	return semver.MustParse(r.Package.Version)
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
	// oh well ...
	// TODO: maybe not static
	if len(workingMcVersion) == 0 {
		return "1.16.5"
	}
	return workingMcVersion[len(workingMcVersion)-1].String()
}

// WorksWithManifest returns if this release was tested to the manifest requirements
// (currently only checks mc version)
func (r *Release) WorksWithManifest(man *manifest.Manifest) bool {
	mcConstraint, err := semver.NewConstraint(man.Requirements.Minecraft)
	if err != nil {
		return false
	}
	for _, test := range r.Tests {
		mcVersion := semver.MustParse(test.Minecraft)
		if mcConstraint.Check(mcVersion) && test.Works {
			return true
		}
	}
	return false
}

// Upload uploads the jar or zipfile for a release
func (r *Release) Upload(reader io.Reader, size int64) (*Release, error) {
	// prepare request
	client := r.client

	url := fmt.Sprintf("%s/releases/%s/%s/upload", r.client.APIUrl, r.Package.Platform, r.Identifier())
	req, err := http.NewRequest("POST", url, reader)
	if err != nil {
		return nil, err
	}

	client.decorate(req)
	req.Header.Set("Content-Type", "application/java-archive")
	if size != 0 {
		req.ContentLength = size
	}

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
	release.decorate(r.client)

	return &release, nil
}

// GetRelease gets a single release from a project
// `identifier` is a project@version string
func (m *MinepkgAPI) GetRelease(ctx context.Context, platform string, identifier string) (*Release, error) {
	res, err := m.get(ctx, m.APIUrl+"/releases/"+platform+"/"+identifier)
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

// DeleteRelease gets a single release from a project
// `identifier` is a project@version string
func (m *MinepkgAPI) DeleteRelease(ctx context.Context, platform string, identifier string) (*Release, error) {
	res, err := m.delete(ctx, m.APIUrl+"/releases/"+platform+"/"+identifier)
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

	return &release, nil
}

// GetReleaseList gets a all available releases for a project
func (m *MinepkgAPI) GetReleaseList(ctx context.Context, project string) ([]*Release, error) {
	p := Project{client: m, Name: project}
	return p.GetReleases(ctx, "")
}
