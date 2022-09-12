package provider

import (
	"context"

	"github.com/minepkg/minepkg/internals/pkgid"
	"github.com/minepkg/minepkg/pkg/manifest"
)

type Provider interface {
	Name() string
	Resolve(ctx context.Context, request *Request) (Result, error)
}

type LatestResolver interface {
	ResolveLatest(ctx context.Context, request *Request) (Result, error)
}

type URLConverter interface {
	CanConvertURL(url string) bool
	ConvertURL(ctx context.Context, url string) (string, error)
}

// Request is a request to resolve a dependency
type Request struct {
	// Dependency is the dependency to resolve
	Dependency *pkgid.ID
	// Requirements is the platform lock of the current instance (e.g. minecraft version)
	Requirements manifest.PlatformLock
	// DependencyLock might be set to the current lock of the dependency. Can also be nil.
	DependencyLock *manifest.DependencyLock
	// Root is the root dependency lock of the current instance
	Root *manifest.DependencyLock
}

// Result is a result of a dependency resolve
type Result interface {
	// Lock returns the dependency lock of the resolved dependency, can NOT be nil
	Lock() *manifest.DependencyLock
	// Dependencies returns the dependencies of the resolved dependency, can be nil
	Dependencies() []*manifest.InterpretedDependency
}
