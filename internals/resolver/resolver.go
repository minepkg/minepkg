package resolver

import (
	"context"
	"errors"
	"fmt"

	"github.com/fiws/minepkg/internals/api"
	"github.com/fiws/minepkg/pkg/manifest"
)

var (
	// ErrNoGlobalReqs is returned when GlobalReqs was not set
	ErrNoGlobalReqs = errors.New("no GlobalReqs set. They are required to resolve")
)

// ErrNoMatchingRelease is returned if a wanted releaseendency (package) could not be resolved given the requirements
type ErrNoMatchingRelease struct {
	// Package is the name of the pacakge that can not be resolved
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
		"No Release found for %s. Details: \n\tPlattform: %s\n\tPackage: %s\n\tVersion: %s\n\tMinecraft Version: %s\n\tDependency of: %s",
		e.Package,
		e.Requirements.Plattform,
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
}

// New returns a new resolver
func New(client *api.MinepkgAPI, reqs manifest.PlatformLock) *Resolver {
	return &Resolver{
		Resolved:      make(map[string]*manifest.DependencyLock),
		client:        client,
		GlobalReqs:    reqs,
		IgnoreVersion: false,
	}
}

// ResolveManifest resolves a manifest
func (r *Resolver) ResolveManifest(man *manifest.Manifest) error {

	if r.GlobalReqs == nil {
		return ErrNoGlobalReqs
	}

	for name, version := range man.Dependencies {

		reqs := &api.RequirementQuery{
			Version:   version,
			Minecraft: r.GlobalReqs.MinecraftVersion(),
			Plattform: man.PlatformString(),
		}

		if r.IgnoreVersion {
			reqs.Version = "*"
		}

		release, err := r.client.FindRelease(context.TODO(), name, reqs)
		if err != nil {
			return err
		}
		err = r.Resolve(release)
		if err != nil {
			return err
		}
	}

	return nil
}

// Resolve resolves all dependencies of a single package
func (r *Resolver) Resolve(release *api.Release) error {

	r.addResolvedRelease(release)
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
			r.addResolvedRelease(resolvedDep)
			resolveNext = append(resolveNext, resolvedDep.InterpretedDependencies()...)
		}
	}

	return nil
}

func (r *Resolver) addResolvedRelease(release *api.Release) {
	r.Resolved[release.Package.Name] = &manifest.DependencyLock{
		Name:     release.Package.Name,
		Version:  release.Package.Version,
		Type:     release.Package.Type,
		IPFSHash: release.Meta.IPFSHash,
		Sha256:   release.Meta.Sha256,
		URL:      release.DownloadURL(),
	}
}

func (r *Resolver) resolveMinepkg(dep *manifest.InterpretedDependency) (*api.Release, error) {
	reqs := &api.RequirementQuery{
		Minecraft: r.GlobalReqs.MinecraftVersion(),
		Version:   dep.Source,
		Plattform: r.GlobalReqs.PlatformName(),
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
