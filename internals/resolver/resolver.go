package resolver

import (
	"context"
	"errors"
	"fmt"

	"github.com/minepkg/minepkg/internals/api"
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
}

// New returns a new resolver
func New(client *api.MinepkgAPI, reqs manifest.PlatformLock) *Resolver {
	return &Resolver{
		Resolved:      make(map[string]*manifest.DependencyLock),
		client:        client,
		GlobalReqs:    reqs,
		IgnoreVersion: false,
		IncludeDev:    true,
	}
}

// ResolveManifest resolves a manifest
func (r *Resolver) ResolveManifest(man *manifest.Manifest) error {

	if r.GlobalReqs == nil {
		return ErrNoGlobalReqs
	}

	for _, dependency := range man.InterpretedDependencies() {
		release, err := r.resolveMinepkg(dependency)
		if err != nil {
			return err
		}
		err = r.Resolve(release, nil, false)
		if err != nil {
			return err
		}
	}

	if r.IncludeDev {
		for _, dependency := range man.InterpretedDevDependencies() {
			release, err := r.resolveMinepkg(dependency)
			if err != nil {
				return err
			}
			err = r.Resolve(release, nil, true)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Resolve resolves all dependencies of a single package
func (r *Resolver) Resolve(release *api.Release, dependend *manifest.Manifest, isDev bool) error {

	// add this release
	r.Resolved[release.Package.Name] = r.lockFromRelease(release, dependend)
	resolveNext := release.InterpretedDependencies()

	for len(resolveNext) != 0 {
		resolveNow := resolveNext
		resolveNext = []*manifest.InterpretedDependency{}

		for _, dep := range resolveNow {
			// already resolved
			_, ok := r.Resolved[dep.Name]
			if ok == true {
				continue
			}
			r.Resolved[dep.Name] = nil

			resolvedDep, err := r.resolveMinepkg(dep)
			if err != nil {
				return err
			}
			lock := r.lockFromRelease(resolvedDep, release.Manifest)
			if isDev {
				lock.IsDev = true
			}
			r.Resolved[resolvedDep.Package.Name] = lock
			resolveNext = append(resolveNext, resolvedDep.InterpretedDependencies()...)
		}
	}

	return nil
}

func (r *Resolver) lockFromRelease(release *api.Release, dependend *manifest.Manifest) *manifest.DependencyLock {
	lock := &manifest.DependencyLock{
		Name:     release.Package.Name,
		Version:  release.Package.Version,
		Type:     release.Package.Type,
		IPFSHash: release.Meta.IPFSHash,
		Sha256:   release.Meta.Sha256,
		URL:      release.DownloadURL(),
	}

	if dependend != nil {
		lock.Dependend = dependend.Package.Name
	} else {
		lock.Dependend = "_root"
	}

	return lock
}

func (r *Resolver) resolveMinepkg(dep *manifest.InterpretedDependency) (*api.Release, error) {
	reqs := &api.RequirementQuery{
		Minecraft: r.GlobalReqs.MinecraftVersion(),
		Version:   dep.Source,
		Platform:  r.GlobalReqs.PlatformName(),
	}

	if r.IgnoreVersion {
		reqs.Version = "*"
	}

	release, err := r.client.FindRelease(context.TODO(), dep.Name, reqs)
	if err != nil {
		return nil, err
	}
	return release, nil
}

func resolveHttp() {

}
