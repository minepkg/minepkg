package patch

import (
	"context"
	"fmt"

	"github.com/minepkg/minepkg/internals/minecraft"
)

type AddLibraries struct{}

type addLibrariesArgs struct {
	Libraries []minecraft.Library `json:"libraries"`
}

func (r *AddLibraries) Apply(ctx context.Context, operation *PatchOperation) error {
	if operation.instance == nil {
		return fmt.Errorf("operation instance is nil")
	}

	var args addLibrariesArgs
	if err := UnmarshalArgs(operation, &args); err != nil {
		return err
	}

	libraries := args.Libraries

	launchManifest, err := operation.instance.GetLaunchManifest()
	if err != nil {
		return err
	}

	launchManifest.Libraries = append(launchManifest.Libraries, libraries...)
	return nil
}
