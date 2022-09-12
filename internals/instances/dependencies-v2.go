package instances

import (
	"sort"

	"github.com/Masterminds/semver/v3"
	"github.com/minepkg/minepkg/internals/pkgid"
	"github.com/minepkg/minepkg/internals/provider"
	"github.com/minepkg/minepkg/pkg/manifest"
)

type Dependency struct {
	name string
	Lock *manifest.DependencyLock
	ID   *pkgid.ID
}

type DependencyList []Dependency

// Sorted returns a list of all dependencies sorted by the name
func (d DependencyList) Sorted() DependencyList {
	// sort by name
	sort.Slice(d, func(i, j int) bool {
		return d[i].name < d[j].name
	})
	return d
}

func (d Dependency) Name() string {
	return d.name
}

func (d Dependency) NeedsUpdating() bool {
	if d.ID.Provider == "dummy" {
		return false
	}
	// contains non minepkg package, unsure if update is needed, better be safe
	if d.ID.Provider != "minepkg" {
		return true
	}

	// missing dependency
	if d.Lock == nil {
		return true
	}

	// might not even be semver, but versions match, next!
	if d.ID.Version == d.Lock.Version {
		return true
	}

	packageDep, err := semver.NewConstraint(d.ID.Version)
	if err != nil {
		return false
	}

	sVersion, err := semver.NewVersion(d.Lock.Version)
	// not semver and not equal? we check
	if err != nil {
		return true
	}
	// Version does not match
	if !packageDep.Check(sVersion) {
		return true
	}

	// all checks passed, no update needed
	return false
}

func ProviderRequest(dependency *Dependency, requirements manifest.PlatformLock) *provider.Request {
	return &provider.Request{
		Dependency:   dependency.ID,
		Requirements: requirements,
		Root:         nil,
	}
}

// GetDependencyList returns a list of all dependencies of the instance
func (i *Instance) GetDependencyList() DependencyList {
	dependencies := make(DependencyList, 0, len(i.Manifest.Dependencies))

	for name, requirement := range i.Manifest.Dependencies {
		lock := i.Lockfile.Dependencies[name]
		pkgID := pkgid.Parse(requirement)
		if pkgID.Name == "" {
			pkgID.Name = name
		}
		pkgID.Platform = i.Manifest.PlatformString()

		dependencies = append(dependencies, Dependency{
			name: name,
			Lock: lock,
			ID:   pkgID,
		})
	}

	return dependencies
}
