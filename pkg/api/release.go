package api

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/Masterminds/semver"
)

// ErrNotMatchingRelease gets returned if no matching release was found
var ErrNotMatchingRelease = errors.New("No matching release found for this dependency")

func (r *Release) decorate(c *MinepkgAPI) {
	r.client = c
	for _, d := range r.Dependencies {
		d.client = c
	}
}

// Identifier returns this release in a "project@version" format. eg: `fabric@0.2.0`
func (r *Release) Identifier() string {
	return r.Project + "@" + r.Version
}

// Filename returns this release in a "project@version.jar" format. eg: `fabric@0.2.0.jar`
func (r *Release) Filename() string {
	return r.Identifier() + ".jar"
}

// DownloadURL returns the download url for this release
func (r *Release) DownloadURL() string {
	if r.FileLocation == "" {
		return ""
	}
	return baseAPI + "/projects/" + r.Identifier() + "/download?platform=" + r.Platform
}

// Upload uploads the jar or zipfile for a release
func (r *Release) Upload(reader io.Reader) (*Release, error) {
	// prepare request
	client := r.client
	req, err := http.NewRequest("POST", baseAPI+"/projects/"+r.Project+"@"+r.Version+"/upload", reader)
	if err != nil {
		return nil, err
	}

	client.decorate(req)
	req.Header.Set("Content-Type", "application/java-archive")

	// execute request and handle response
	res, err := client.http.Do(req)
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
func (m *MinepkgAPI) GetRelease(ctx context.Context, project string, version string) (*Release, error) {
	res, err := m.get(ctx, baseAPI+"/projects/"+project+"@"+version)
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

// FindRelease gets the latest release matching the versionRequirement (can be "latest" or a semver requirement)
func (m *MinepkgAPI) FindRelease(ctx context.Context, project string, reqs *RequirementQuery) (*Release, error) {
	p := Project{client: m, Name: project}

	versionRequirement := reqs.Version
	releases, err := p.GetReleases(ctx, reqs.Plattform)
	if err != nil {
		return nil, err
	}

	// found nothing
	if len(releases) == 0 {
		return nil, ErrNotMatchingRelease
	}

	// TODO: handle prereleases
	if versionRequirement == "latest" || versionRequirement == "*" {
		return releases[0], nil
	}

	semverReq, err := semver.NewConstraint(versionRequirement)
	if err != nil {
		return nil, err
	}

	for _, r := range releases {
		if semverReq.Check(r.SemverVersion()) == true {
			return r, nil
		}
	}

	// found nothing
	return nil, ErrNotMatchingRelease
}
