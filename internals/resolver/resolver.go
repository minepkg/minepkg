package resolver

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/resolver/providers"
	"github.com/minepkg/minepkg/pkg/manifest"
)

var (
	// ErrNoGlobalReqs is returned when GlobalReqs was not set
	ErrNoGlobalReqs = errors.New("no GlobalReqs set. They are required to resolve")
)

// ErrNoMatchingRelease is returned if a wanted releaseendency (package) could not be resolved given the requirements
type ErrNoMatchingRelease struct {
	// Package is the name of the package that can not be resolved
	Package string
	// Requirements are the requirements for this package to resolve (eg. minecraft version)
	Requirements *api.RequirementQuery
	// Parent is the release of the package that required this one (if any)
	Parent *api.Release
}

func (e *ErrNoMatchingRelease) Error() string {
	parent := "(root)"
	if e.Parent != nil {
		parent = e.Parent.Package.Name
	}
	return fmt.Sprintf(
		"No Release found for %s. Details: \n\tPlatform: %s\n\tPackage: %s\n\tVersion: %s\n\tMinecraft Version: %s\n\tDependency of: %s",
		e.Package,
		e.Requirements.Platform,
		e.Package,
		e.Requirements.Version,
		e.Requirements.Minecraft,
		parent,
	)
}

// Resolver resolves given the mods of given dependencies
type Resolver struct {
	Resolved   map[string]*manifest.DependencyLock
	client     *api.MinepkgAPI
	GlobalReqs manifest.PlatformLock
	// IgnoreVersion will make the resolver ignore all version requirements and just fetch the latest version for everything
	IgnoreVersion bool
	// IncludeDev includes dev.dependencies
	IncludeDev bool

	Providers map[string]providers.Provider
}

// New returns a new resolver
func New(client *api.MinepkgAPI, reqs manifest.PlatformLock) *Resolver {
	resolver := &Resolver{
		Resolved:      make(map[string]*manifest.DependencyLock),
		client:        client,
		GlobalReqs:    reqs,
		IgnoreVersion: false,
		IncludeDev:    true,
		Providers:     make(map[string]providers.Provider, 1),
	}

	resolver.Providers["minepkg"] = &providers.MinepkgProvider{
		Client: client,
	}

	resolver.Providers["https"] = &providers.HttpProvider{
		Client: http.DefaultClient,
	}

	return resolver
}

// ResolveManifest resolves a manifest
func (r *Resolver) ResolveManifest(man *manifest.Manifest) error {

	if r.GlobalReqs == nil {
		return ErrNoGlobalReqs
	}

	for _, dependency := range man.InterpretedDependencies() {
		if err := r.Resolve(dependency, false); err != nil {
			return err
		}
	}

	if r.IncludeDev {
		for _, dependency := range man.InterpretedDevDependencies() {
			if err := r.Resolve(dependency, true); err != nil {
				return err
			}
		}
	}

	return nil
}

// Resolve resolves all dependencies of a single package
func (r *Resolver) Resolve(dep *manifest.InterpretedDependency, isDev bool) error {

	provider, ok := r.Providers[dep.Provider]
	if !ok {
		return fmt.Errorf("%s needs %s as install provider which is not supported", dep.Name, dep.Provider)
	}
	// add this release
	result, err := provider.Resolve(r.providerRequest(dep))
	if err != nil {
		return err
	}

	lock := result.Lock()
	lock.Dependend = "_root"

	if isDev {
		lock.IsDev = true
	}
	r.Resolved[lock.Name] = lock

	parentPackage := result
	resolveNext := result.Dependencies()

	for len(resolveNext) != 0 {
		resolveNow := resolveNext
		resolveNext = []*manifest.InterpretedDependency{}

		for _, dep := range resolveNow {
			// already resolved
			_, ok := r.Resolved[dep.Name]
			if ok {
				continue
			}
			r.Resolved[dep.Name] = nil

			result, err := r.Providers[dep.Provider].Resolve(r.providerRequest(dep))
			if err != nil {
				return err
			}

			lock := result.Lock()
			lock.Dependend = parentPackage.Lock().Name

			if isDev {
				lock.IsDev = true
			}
			r.Resolved[lock.Name] = lock
			resolveNext = append(resolveNext, result.Dependencies()...)
			parentPackage = result
		}
	}

	return nil
}

func (r *Resolver) providerRequest(dep *manifest.InterpretedDependency) *providers.Request {
	return &providers.Request{
		Dependency:   dep,
		Requirements: r.GlobalReqs,
	}
}
