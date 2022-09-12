package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/minepkg/minepkg/pkg/manifest"
)

type DummyProvider struct {
	Client *http.Client
}

type dummyResult struct{ *Request }

func NewDummyProvider() *DummyProvider {
	return &DummyProvider{
		Client: http.DefaultClient,
	}
}

func (d *dummyResult) Lock() *manifest.DependencyLock {
	lock := &manifest.DependencyLock{
		Name:     d.Dependency.Name,
		Provider: d.Dependency.Provider,
		Type:     "generic",
		Version:  "none",
	}

	return lock
}

func (d *dummyResult) Dependencies() []*manifest.InterpretedDependency {
	return []*manifest.InterpretedDependency{}
}

func (d *DummyProvider) Name() string { return "dummy" }

func (d *DummyProvider) Resolve(ctx context.Context, request *Request) (Result, error) {

	return &dummyResult{request}, nil
}

func (d *DummyProvider) Fetch(ctx context.Context, toFetch Result) (io.Reader, int, error) {
	return nil, 0, fmt.Errorf("dummy provider can not fetch")
}
