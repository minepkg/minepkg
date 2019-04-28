package api

import (
	"context"
	"errors"

	"github.com/Masterminds/semver"
)

// ErrNotMatchingRelease gets returned if no matching release was found
var ErrNotMatchingRelease = errors.New("No matching release found for this dependency")

// Dependency in verbose form
// type Dependency struct {
// 	// Provider is only minepkg for now. Kept for future extensions
// 	Provider string `json:"provider"`
// 	// Name is the name of the package (eg. storage-drawers)
// 	Name string `json:"name"`
// 	// VersionRequirement is a semver version Constraint
// 	// Example: `^2.9.22` or `5.x.x`
// 	VersionRequirement string `json:"versionRequirement"`
// }

// Resolve tries to fetch a matching release to this dependency requirement
func (d *Dependency) Resolve(ctx context.Context) (*Release, error) {
	releases, err := d.client.GetReleaseList(ctx, d.Name)
	if err != nil {
		return nil, err
	}

	if d.VersionRequirement == "latest" || d.VersionRequirement == "*" {
		return releases[0], nil
	}

	semverReq, err := semver.NewConstraint(d.VersionRequirement)
	if err != nil {
		return nil, err
	}

	for _, r := range releases {
		if semverReq.Check(r.SemverVersion()) == true {
			return r, nil
		}
	}

	// found nothing
	return nil, ErrNotMatchingRelease
}
