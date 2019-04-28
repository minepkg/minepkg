package api

import (
	"context"
	"github.com/fiws/minepkg/pkg/manifest"
)

// Project returns a Project object without fetching it from the API
func (m *MinepkgAPI) Project(name string) *Project {
	return &Project{
		c:    m,
		Name: name,
	}
}

// GetProject gets a single project
func (m *MinepkgAPI) GetProject(ctx context.Context, name string) (*Project, error) {
	res, err := m.get(ctx, baseAPI+"/projects/"+name)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(res); err != nil {
		return nil, err
	}

	project := Project{c: m}
	if err := parseJSON(res, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

// CreateProject creates a new project
func (m *MinepkgAPI) CreateProject(p *Project) (*Project, error) {
	res, err := m.postJSON(context.TODO(), baseAPI+"/projects", p)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(res); err != nil {
		return nil, err
	}

	project := Project{c: m}
	if err := parseJSON(res, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

// CreateRelease will create a new release
func (p *Project) CreateRelease(ctx context.Context, m *manifest.Manifest) (*Release, error) {
	wrap := struct {
		Manifest  *manifest.Manifest `json:"manifest"`
		Published bool               `json:"published"`
	}{
		Manifest:  m,
		Published: m.Package.Type == manifest.TypeModpack,
	}
	res, err := p.c.postJSON(ctx, baseAPI+"/projects/"+m.Package.Name+"/releases", wrap)
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
	release.decorate(p.c)
	return &release, nil
}

// GetReleases gets a all available releases for this project
func (p *Project) GetReleases(ctx context.Context) ([]*Release, error) {
	res, err := p.c.get(ctx, baseAPI+"/projects/"+p.Name+"/releases")
	if err != nil {
		return nil, err
	}
	if err := checkResponse(res); err != nil {
		return nil, err
	}

	releases := make([]*Release, 0, 20)
	if err := parseJSON(res, &releases); err != nil {
		return nil, err
	}
	for _, r := range releases {
		r.decorate(p.c) // sets the private client field
	}

	return releases, nil
}
