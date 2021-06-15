package providers

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

func (h *dummyResult) Lock() *manifest.DependencyLock {
	lock := &manifest.DependencyLock{
		Name:     h.Dependency.Name,
		Provider: h.Dependency.Provider,
		Type:     "generic",
		Version:  "none",
	}

	return lock
}

func (h *dummyResult) Dependencies() []*manifest.InterpretedDependency {
	return []*manifest.InterpretedDependency{}
}

func (h *DummyProvider) Resolve(ctx context.Context, request *Request) (Result, error) {

	return &dummyResult{request}, nil
}

func (h *DummyProvider) Fetch(ctx context.Context, toFetch Result) (io.Reader, int, error) {
	return nil, 0, fmt.Errorf("dummy provider can not fetch")
}
