package api

import (
	"context"
	"net/url"

	"github.com/fiws/minepkg/pkg/manifest"
)

// Project returns a Project object without fetching it from the API
func (m *MinepkgAPI) Project(name string) *Project {
	return &Project{
		client: m,
		Name:   name,
	}
}

// GetProjectsQuery are the query paramters for the GetProjects function
type GetProjectsQuery struct {
	Type     string `json:"type"`
	Platform string `json:"platform"`
	Simple   bool   `json:"simple"`
}

// GetProjects gets all projects matching a query
func (m *MinepkgAPI) GetProjects(ctx context.Context, opts *GetProjectsQuery) ([]Project, error) {

	uri, err := url.Parse(baseAPI + "/projects")
	if err != nil {
		return nil, err
	}

	uri.Query().Set("type", opts.Type)
	uri.Query().Set("platform", opts.Platform)
	if opts.Simple == true {
		uri.Query().Set("simple", "true")
	}

	res, err := m.get(ctx, uri.String())
	if err != nil {
		return nil, err
	}
	if err := checkResponse(res); err != nil {
		return nil, err
	}

	projects := make([]Project, 0)
	if err := parseJSON(res, &projects); err != nil {
		return nil, err
	}

	return projects, nil
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

	project := Project{client: m}
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

	project := Project{client: m}
	if err := parseJSON(res, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

// CreateRelease will create a new release
func (p *Project) CreateRelease(ctx context.Context, man *manifest.Manifest) (*Release, error) {
	res, err := p.client.postJSON(ctx, baseAPI+"/releases", man)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(res); err != nil {
		return nil, err
	}

	release := Release{client: p.client}
	if err := parseJSON(res, &release); err != nil {
		return nil, err
	}
	return &release, nil
}

// GetReleases gets a all available releases for this project
func (p *Project) GetReleases(ctx context.Context, platform string) ([]*Release, error) {
	platformParam := "?platform=fabric"
	if platform != "" {
		platformParam = "?platform=" + platform
	}
	res, err := p.client.get(ctx, baseAPI+"/projects/"+p.Name+"/releases"+platformParam)
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
		r.decorate(p.client) // sets the private client field
	}

	return releases, nil
}
