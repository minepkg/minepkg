package api

import (
	"context"
	"io"
	"net/http"
)

func (r *Release) decorate(c *MinepkgAPI) {
	r.client = c
	for _, d := range r.Dependencies {
		d.client = c
	}
}

// DownloadURL returns the download url for this release
func (r *Release) DownloadURL() string {
	return baseAPI + "/projects/" + r.Project + "@" + r.Version + "/download"
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
	return p.GetReleases(ctx)
}
