package instances

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/minepkg/minepkg/pkg/manifest"
)

func areRequirementsInLockfileOutdated(lock *manifest.Lockfile, mani *manifest.Manifest) (bool, error) {
	if mani == nil {
		return false, fmt.Errorf("manifest is nil")
	}

	// no lockfile or requirements are missing
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

func areDependenciesInLockfileOutdated(lock *manifest.Lockfile, mani *manifest.Manifest) (bool, error) {
	if mani == nil {
		return false, fmt.Errorf("manifest is nil")
	}

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

		lockEntry, ok := lock.Dependencies[dep.Name]
		// missing dependency
		if !ok {
			return true, nil
		}

		// might not even be semver, but versions match, next!
		if dep.Source == lockEntry.Version {
			continue
		}

		packageDep, err := semver.NewConstraint(dep.Source)
		if err != nil {
			return false, err
		}

		sVersion, err := semver.NewVersion(lockEntry.Version)
		// not semver and not equal? we check
		if err != nil {
			return true, nil
		}
		// Version does not match
		if !packageDep.Check(sVersion) {
			return true, nil
		}
	}

	// check for removed dependencies
	for _, lock := range lock.Dependencies {
		// ignore dev dependencies for now
		if lock.IsDev {
			continue
		}
		if lock.Dependend == "" || lock.Dependend == mani.Package.Name {
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

// AreDependenciesOutdated returns true if the dependencies of this instance do not
// match what is currently set in the lockfile. Dependencies should be updated with
// "UpdateLockfileDependencies" in most cases if this is true
func (i *Instance) AreDependenciesOutdated() (bool, error) {
	return areDependenciesInLockfileOutdated(i.Lockfile, i.Manifest)
}

// AreRequirementsOutdated returns true if the requirements of this instance do not
// match what is currently set in the lockfile. Requirements should be updated with
// "UpdateLockfileRequirements" in most cases if this is true
func (i *Instance) AreRequirementsOutdated() (bool, error) {
	return areRequirementsInLockfileOutdated(i.Lockfile, i.Manifest)
}
