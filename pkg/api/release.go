package api

import (
	"context"
)

func (r *Release) decorate(c *MinepkgAPI) {
	r.c = c
	for _, d := range r.Dependencies {
		d.c = c
	}
}

// DownloadURL returns the download url for this release
func (r *Release) DownloadURL() string {
	return baseAPI + "/projects/" + r.Project + "@" + r.Version.String() + "/download"
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
	p := Project{c: m, Name: project}
	return p.GetReleases(ctx)
}
