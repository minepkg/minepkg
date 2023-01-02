package patch

import (
	"context"
	"fmt"

	"github.com/minepkg/minepkg/internals/minecraft"
)

type AddLibraries struct {
	args struct {
		Libraries []minecraft.Lib `json:"libraries"`
	}
}

func (r *AddLibraries) Args() any {
	return &r.args
}

func (r *AddLibraries) Apply(ctx context.Context, operation *PatchOperation) error {
	if operation.instance == nil {
		return fmt.Errorf("operation instance is nil")
	}

	libraries := r.args.Libraries

	launchManifest, err := operation.instance.GetLaunchManifest()
	if err != nil {
		return err
	}

	launchManifest.Libraries = append(launchManifest.Libraries, libraries...)
	return nil
}
