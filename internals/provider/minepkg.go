package provider

import (
	"context"
	"io"
	"net/http"

	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/pkg/manifest"
)

type MinepkgProvider struct {
	Client *api.MinepkgClient
}

type minepkgResult struct{ *api.Release }

func (m *minepkgResult) Lock() *manifest.DependencyLock {
	lock := &manifest.DependencyLock{
		Name:     m.Package.Name,
		Version:  m.Package.Version,
		Type:     m.Package.Type,
		IPFSHash: m.Meta.IPFSHash,
		Sha256:   m.Meta.Sha256,
		URL:      m.DownloadURL(),
		Provider: "minepkg",
	}

	return lock
}

func (m *minepkgResult) Dependencies() []*manifest.InterpretedDependency {
	return m.InterpretedDependencies()
}

func (m *MinepkgProvider) Name() string { return "minepkg" }

func (m *MinepkgProvider) Resolve(ctx context.Context, request *Request) (Result, error) {
	reqs := &api.ReleasesQuery{
		Name:         request.Dependency.Name,
		VersionRange: request.Dependency.Version,
		Minecraft:    request.Requirements.MinecraftVersion(),
		Platform:     request.Requirements.PlatformName(),
	}

	release, err := m.Client.ReleasesQuery(ctx, reqs)
	if err != nil {
		return nil, err
	}

	return &minepkgResult{release}, nil
}

func (m *MinepkgProvider) ResolveLatest(ctx context.Context, request *Request) (Result, error) {
	request.Dependency.Version = "*"
	return m.Resolve(ctx, request)
}

func (m *MinepkgProvider) Fetch(ctx context.Context, toFetch Result) (io.Reader, int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", toFetch.Lock().URL, nil)
	if err != nil {
		return nil, 0, err
	}

	fileRes, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}

	return fileRes.Body, int(fileRes.ContentLength), nil
}
