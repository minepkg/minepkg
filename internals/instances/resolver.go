package instances

import (
	"context"
	"errors"

	"github.com/fiws/minepkg/pkg/api"
	"github.com/fiws/minepkg/pkg/manifest"
)

var (
	// ErrNoGlobalReqs is returned when GlobalReqs was not set
	ErrNoGlobalReqs = errors.New("No GlobalReqs set. They are required to resolve")
)

// Resolver resolves given the mods of given dependencies
type Resolver struct {
	Resolved   map[string]*api.Release
	client     *api.MinepkgAPI
	GlobalReqs manifest.PlatformLock
}

// NewResolver returns a new resolver
func NewResolver(client *api.MinepkgAPI, reqs manifest.PlatformLock) *Resolver {
	return &Resolver{
		Resolved:   make(map[string]*api.Release),
		client:     client,
		GlobalReqs: reqs,
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
			Plattform: "fabric", // TODO: not hardcoded!
		}

		release, err := r.client.FindRelease(context.TODO(), name, reqs)
		if err != nil {
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
	r.Resolved[release.Project] = release
	// TODO: parallelize
	for _, d := range release.Dependencies {
		_, ok := r.Resolved[d.Name]
		if ok == true {
			continue
		}
		r.Resolved[d.Name] = nil
		reqs := &api.RequirementQuery{
			Minecraft: r.GlobalReqs.MinecraftVersion(),
			Version:   d.VersionRequirement,
			Plattform: r.GlobalReqs.PlatformName(),
		}
		release, err := r.client.FindRelease(context.TODO(), d.Name, reqs)
		if err != nil {
			return err
		}
		return r.ResolveSingle(release)
	}

	return nil
}
