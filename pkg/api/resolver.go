package api

import (
	"context"

	"github.com/fiws/minepkg/pkg/manifest"
)

// Resolver resolves given the mods of given dependencies
type Resolver struct {
	Resolved map[string]*Release
	client   *MinepkgAPI
}

// NewResolver returns a new resolver
func NewResolver(client *MinepkgAPI) *Resolver {
	return &Resolver{Resolved: make(map[string]*Release), client: client}
}

// ResolveManifest resolves a manifest
func (r *Resolver) ResolveManifest(man *manifest.Manifest) error {

	for name, version := range man.Dependencies {
		release, err := r.client.FindRelease(context.TODO(), name, version)
		if err != nil {
			return err
		}
		r.ResolveSingle(release)
	}

	return nil
}

// Resolve find all dependencies from the given `id`
// and adds it to the `resolved` map. Nothing is returned
func (r *Resolver) Resolve(releases []*Release) error {
	for _, release := range releases {
		r.ResolveSingle(release)
	}

	return nil
}

// ResolveSingle resolves all dependencies of a single release
func (r *Resolver) ResolveSingle(release *Release) error {

	r.Resolved[release.Project] = release
	// TODO: parallelize
	for _, d := range release.Dependencies {
		_, ok := r.Resolved[d.Name]
		if ok == true {
			return nil
		}
		r.Resolved[d.Name] = nil
		release, err := d.Resolve(context.TODO())
		if err != nil {
			return err
		}
		r.ResolveSingle(release)
	}

	return nil
}
