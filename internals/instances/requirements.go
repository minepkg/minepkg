package instances

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/pkg/manifest"
)

var (
	// ErrNoFabricLoader is returned if the wanted fabric version was not found
	ErrNoFabricLoader = &commands.CliError{
		Text: "Could not find fabric loader for wanted Minecraft version",
		Suggestions: []string{
			"Check if fabric is compatible with the Minecraft version in your minepkg.toml",
			"Check your requirements.fabric field",
			"Try again later",
		},
	}
	// ErrNoFabricMapping is returned if the wanted fabric mapping was not found
	ErrNoFabricMapping = &commands.CliError{
		Text: "Could not find fabric mapping for wanted Minecraft version",
		Suggestions: []string{
			"Check if fabric is compatible with the Minecraft version in your minepkg.toml",
			"Check your requirements.fabric field",
			"Try again later",
		},
	}
)

// UpdateLockfileRequirements updates the internal lockfile manifest with `VanillaLock`, `FabricLock` or `ForgeLock`
// containing the resolved requirements (semver requirement to actual version)
func (i *Instance) UpdateLockfileRequirements(ctx context.Context) error {
	if i.Lockfile == nil {
		i.Lockfile = manifest.NewLockfile()
	}
	switch i.Platform() {
	case PlatformFabric:
		lock, err := i.resolveFabricRequirement(ctx)
		if err != nil {
			return err
		}
		i.Lockfile.Fabric = lock
	case PlatformForge:
		fmt.Println("forge is not supported for now")
	case PlatformVanilla:
		version, err := i.resolveVanillaRequirement(ctx)
		if err != nil {
			return err
		}
		i.Lockfile.Vanilla = &manifest.VanillaLock{Minecraft: version.ID}
	}
	return nil
}

func (i *Instance) resolveVanillaRequirement(ctx context.Context) (*MinecraftRelease, error) {
	constraint, _ := semver.NewConstraint(i.Manifest.Requirements.Minecraft)
	res, err := GetMinecraftReleases(ctx)
	if err != nil {
		return nil, err
	}

	// find newest compatible version
	for _, v := range res.Versions {
		// TODO: some versions contain spaces
		semverVersion, err := semver.NewVersion(v.ID)

		// skip non-parsable minecraft versions
		if err != nil {
			// fallback to string compare
			if v.ID == i.Manifest.Requirements.Minecraft {
				return &v, nil
			}
			continue
		}

		if constraint.Check(semverVersion) {
			return &v, nil
		}
	}

	return nil, nil
}

func (i *Instance) resolveFabricRequirement(ctx context.Context) (*manifest.FabricLock, error) {

	reqMc := i.Manifest.Requirements.Minecraft
	// latest is the same as '*' for this logic (it will return the latest version)
	if reqMc == "latest" {
		reqMc = "*"
	}

	reqFabric := i.Manifest.Requirements.FabricLoader
	// latest is the same as '*' for this logic (it will return the latest version)
	if reqFabric == "latest" {
		reqFabric = "*"
	}

	// TODO: check for invalid semver
	MCconstraint, _ := semver.NewConstraint(reqMc)
	FabricLoaderConstraint, _ := semver.NewConstraint(reqFabric)
	// mcVersions, err := GetMinecraftReleases(ctx)

	fabricMappings, err := getFabricMappingVersions(ctx)
	if err != nil {
		return nil, err
	}
	fabricLoaders, err := getFabricLoaderVersions(ctx)
	if err != nil {
		return nil, err
	}

	var foundMapping *fabricMappingVersion

	// find newest compatible version
	for _, v := range fabricMappings {
		// TODO: some versions contain spaces
		semverVersion, err := semver.NewVersion(v.GameVersion)

		// skip unparsable minecraft versions
		if err != nil {
			continue
		}

		if MCconstraint.Check(semverVersion) {
			foundMapping = &v
			break
		}
	}

	if foundMapping == nil {
		return nil, ErrNoFabricMapping
	}

	var foundLoader *fabricLoaderVersion
	// find newest compatible version
	for _, v := range fabricLoaders {
		// TODO: some versions contain spaces
		semverVersion, err := semver.NewVersion(v.Version)

		// skip unparsable minecraft versions
		if err != nil {
			continue
		}

		if FabricLoaderConstraint.Check(semverVersion) {
			foundLoader = &v
			break
		}
	}

	if foundLoader == nil {
		return nil, ErrNoFabricLoader
	}

	return &manifest.FabricLock{
		Minecraft:    foundMapping.GameVersion,
		Mapping:      foundMapping.Version,
		FabricLoader: foundLoader.Version,
	}, nil
}
