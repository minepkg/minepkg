package api

import (
	"context"
)

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
	return d.client.FindRelease(ctx, d.Name, d.VersionRequirement)
}
