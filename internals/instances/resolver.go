package instances

import (
	"context"
	"errors"
	"fmt"

	"github.com/fiws/minepkg/pkg/api"
	"github.com/fiws/minepkg/pkg/manifest"
)

var (
	// ErrNoGlobalReqs is returned when GlobalReqs was not set
	ErrNoGlobalReqs = errors.New("No GlobalReqs set. They are required to resolve")
)

// ErrNoMatchingRelease is returned if a wanted dependency (package) could not be resolved given the requirements
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
	Resolved   map[string]*api.Release
	client     *api.MinepkgAPI
	GlobalReqs manifest.PlatformLock
	// IgnoreVersion will make the resolver ignore all version requirements and just fetch the latest version for everything
	IgnoreVersion bool
}

// NewResolver returns a new resolver
func NewResolver(client *api.MinepkgAPI, reqs manifest.PlatformLock) *Resolver {
	return &Resolver{
		Resolved:      make(map[string]*api.Release),
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
			Minecraft: man.Requirements.Minecraft,
			Plattform: man.PlatformString(),
		}

		if r.IgnoreVersion {
			reqs.Version = "*"
		}

		release, err := r.client.FindRelease(context.TODO(), name, reqs)
		if err != nil {
			if err == api.ErrNotMatchingRelease {
				return &ErrNoMatchingRelease{Package: name, Requirements: reqs}
			}
			return err
		}
		err = r.ResolveSingle(release)
		if err != nil {
			return err
		}
	}

	return nil
}

// Resolve find all dependencies from the given `id`
// and adds it to the `resolved` map. Nothing is returned
func (r *Resolver) Resolve(releases []*api.Release) error {
	for _, release := range releases {
		r.ResolveSingle(release)
	}

	return nil
}

// ResolveSingle resolves all dependencies of a single release
func (r *Resolver) ResolveSingle(release *api.Release) error {
	r.Resolved[release.Package.Name] = release
	original := release
	// TODO: parallelize
	for name, versionRequirement := range release.Dependencies {
		_, ok := r.Resolved[name]
		if ok == true {
			continue
		}
		r.Resolved[name] = nil
		reqs := &api.RequirementQuery{
			Minecraft: r.GlobalReqs.MinecraftVersion(),
			Version:   versionRequirement,
			Plattform: r.GlobalReqs.PlatformName(),
		}

		if r.IgnoreVersion {
			reqs.Version = "*"
		}

		release, err := r.client.FindRelease(context.TODO(), name, reqs)
		if err != nil {
			if err == api.ErrNotMatchingRelease {
				return &ErrNoMatchingRelease{Package: name, Requirements: reqs, Parent: original}
			}
			return err
		}
		return r.ResolveSingle(release)
	}

	return nil
}
