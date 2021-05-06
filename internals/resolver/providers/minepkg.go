package providers

import (
	"context"

	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/pkg/manifest"
)

type MinepkgProvider struct {
	Client *api.MinepkgAPI
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

func (m *MinepkgProvider) Resolve(request *Request) (Result, error) {
	reqs := &api.RequirementQuery{
		Version:   request.Dependency.Source,
		Minecraft: request.Requirements.MinecraftVersion(),
		Platform:  request.Requirements.PlatformName(),
	}

	if request.ignoreVersionsFlag {
		reqs.Version = "*"
	}

	release, err := m.Client.FindRelease(context.TODO(), request.Dependency.Name, reqs)
	if err != nil {
		return nil, err
	}

	return &minepkgResult{release}, nil
}
