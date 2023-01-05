package patch

import (
	"context"
	"fmt"

	"github.com/minepkg/minepkg/pkg/manifest"
)

type MergeMinepkgManifest struct{}

type changePackagesArgs struct {
	Manifest *manifest.Manifest `json:"manifest,omitempty"`
}

func (c *MergeMinepkgManifest) Apply(ctx context.Context, operation *PatchOperation) error {
	if operation.instance == nil {
		return fmt.Errorf("operation instance is nil")
	}

	var args changePackagesArgs
	if err := UnmarshalArgs(operation, &args); err != nil {
		return err
	}

	manifest := operation.instance.Manifest
	if manifest == nil {
		return fmt.Errorf("instance manifest is nil")
	}

	if args.Manifest == nil {
		return fmt.Errorf("supplied manifest is nil")
	}

	for lib, version := range args.Manifest.Dependencies {
		manifest.AddDependency(lib, version)
	}

	for lib, version := range args.Manifest.Dev.Dependencies {
		manifest.AddDevDependency(lib, version)
	}

	return nil
}
