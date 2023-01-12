package api

import (
	"context"
	"net/url"
)

// Project returns a Project object without fetching it from the API
func (m *MinepkgClient) Project(name string) *Project {
	return &Project{
		client: m,
		Name:   name,
	}
}

// GetProjectsQuery are the query parameters for the GetProjects function
type GetProjectsQuery struct {
	Type     string `json:"type"`
	Platform string `json:"platform"`
	Simple   bool   `json:"simple"`
}

// GetProjects gets all projects matching a query
func (m *MinepkgClient) GetProjects(ctx context.Context, opts *GetProjectsQuery) ([]Project, error) {

	uri, err := url.Parse(m.APIUrl + "/projects")
	if err != nil {
		return nil, err
	}

	query := uri.Query()

	query.Set("type", opts.Type)
	query.Set("platform", opts.Platform)
	if opts.Simple {
		query.Set("simple", "true")
	}

	uri.RawQuery = query.Encode()

	res, err := m.get(ctx, uri.String())
	if err != nil {
		return nil, err
	}

	projects := make([]Project, 0)
	if err := decode(res, &projects); err != nil {
		return nil, err
	}

	return projects, nil
}

// GetProject gets a single project
func (m *MinepkgClient) GetProject(ctx context.Context, name string) (*Project, error) {
	res, err := m.get(ctx, m.APIUrl+"/projects/"+name)
	if err != nil {
		return nil, err
	}

	project := Project{client: m}
	if err := decode(res, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

// CreateProject creates a new project
func (m *MinepkgClient) CreateProject(p *Project) (*Project, error) {
	res, err := m.postJSON(context.TODO(), m.APIUrl+"/projects", p)
	if err != nil {
		return nil, err
	}

	project := Project{client: m}
	if err := decode(res, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

// GetProjectStats gets the statistics for a project
func (m *MinepkgClient) GetProjectStats(ctx context.Context, name string) (*ProjectStats, error) {
	res, err := m.get(ctx, m.APIUrl+"/projects/"+name+"/stats")
	if err != nil {
		return nil, err
	}

	stats := ProjectStats{}
	if err := decode(res, &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// CreateRelease will create a new release
func (p *Project) CreateRelease(ctx context.Context, r *Release) (*Release, error) {
	res, err := p.client.postJSON(ctx, p.client.APIUrl+"/releases", r)
	if err != nil {
		return nil, err
	}

	release := Release{client: p.client}
	if err := decode(res, &release); err != nil {
		return nil, err
	}
	return &release, nil
}

// GetReleases gets a all available releases for this project
func (p *Project) GetReleases(ctx context.Context, platform string) (ReleaseList, error) {
	platformParam := "?platform=fabric"
	if platform != "" {
		platformParam = "?platform=" + platform
	}
	res, err := p.client.get(ctx, p.client.APIUrl+"/projects/"+p.Name+"/releases"+platformParam)
	if err != nil {
		return nil, err
	}

	releases := make([]*Release, 0, 20)
	if err := decode(res, &releases); err != nil {
		return nil, err
	}
	for _, r := range releases {
		r.decorate(p.client) // sets the private client field
	}

	return ReleaseList(releases), nil
}
