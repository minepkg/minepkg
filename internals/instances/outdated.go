package instances

import (
	"github.com/Masterminds/semver/v3"
	"github.com/minepkg/minepkg/pkg/manifest"
)

// AreRequirementsOutdated returns true if the requirements of this instance do not
// match what is currently set in the lockfile. Requirements should be updated with
// "UpdateLockfileRequirements" in most cases if this is true
func (i *Instance) AreRequirementsOutdated() (bool, error) {
	lock := i.Lockfile
	mani := i.Manifest

	// no lockfile yet or requirements are missing
	if lock == nil || !lock.HasRequirements() {
		return true, nil
	}

	mcVersionReq, err := semver.NewConstraint(mani.Requirements.Minecraft)
	if err != nil {
		return false, err
	}

	if !mcVersionReq.Check(semver.MustParse(lock.MinecraftVersion())) {
		return true, nil
	}

	platform := mani.PlatformString()
	// vanilla instance with up to date minecraft version, has no loader
	if platform == manifest.PlatformVanilla {
		return false, nil
	}

	platformVersionReq, err := semver.NewConstraint(mani.PlatformVersion())
	if err != nil {
		return false, err
	}

	// check if the platform version is up to date (mod loader version)
	if !platformVersionReq.Check(semver.MustParse(lock.PlatformLock().PlatformVersion())) {
		return true, nil
	}

	// all check are fine, this instance is up to date
	return false, nil
}

// AreDependenciesOutdated returns true if the dependencies of this instance do not
// match what is currently set in the lockfile. Dependencies should be updated with
// "UpdateLockfileDependencies" in most cases if this is true
func (i *Instance) AreDependenciesOutdated() (bool, error) {
	lock := i.Lockfile
	mani := i.Manifest

	// no lockfile yet or dependencies are missing
	if lock == nil || lock.Dependencies == nil {
		return true, nil
	}

	deps := mani.InterpretedDependencies()
	for _, dep := range deps {
		if dep.Provider == "dummy" {
			continue
		}
		// contains non minepkg package, unsure if update is needed, better be safe
		if dep.Provider != "minepkg" {
			return true, nil
		}

		packageDep, err := semver.NewConstraint(dep.Source)
		if err != nil {
			return false, err
		}
		lockEntry, ok := lock.Dependencies[dep.Name]
		// missing dependency
		if !ok {
			return true, nil
		}
		// Version does not match
		if !packageDep.Check(semver.MustParse(lockEntry.Version)) {
			return true, nil
		}
	}

	// check for removed dependencies
	for _, lock := range lock.Dependencies {
		if lock.Dependend == "" || lock.Dependend == i.Manifest.Package.Name {
			if lock.Name == "minepkg-companion" {
				continue
			}
			if _, ok := mani.Dependencies[lock.Name]; !ok {
				return true, nil
			}
		}
	}

	return false, nil
}
