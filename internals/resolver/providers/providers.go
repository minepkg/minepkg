package providers

import (
	"context"

	"github.com/minepkg/minepkg/pkg/manifest"
)

type Provider interface {
	Resolve(ctx context.Context, request *Request) (Result, error)
	// Fetch(ctx context.Context, toFetch Result) (io.Reader, int, error)
}

type Request struct {
	Dependency   *manifest.InterpretedDependency
	Requirements manifest.PlatformLock
	Root         *manifest.DependencyLock

	ignoreVersionsFlag bool
}

type Result interface {
	Lock() *manifest.DependencyLock
	Dependencies() []*manifest.InterpretedDependency
}
