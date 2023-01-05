package patch

import (
	"context"
	"fmt"
	"strings"

	"github.com/minepkg/minepkg/internals/minecraft"
)

type RemoveLibraries struct{}

type removeLibrariesArgs struct {
	Prefix string `json:"prefix"`
}

func (r *RemoveLibraries) Apply(ctx context.Context, operation *PatchOperation) error {
	if operation.instance == nil {
		return fmt.Errorf("operation instance is nil")
	}

	var args removeLibrariesArgs
	if err := UnmarshalArgs(operation, &args); err != nil {
		return err
	}

	prefix := args.Prefix
	if prefix == "" {
		return fmt.Errorf("prefix is empty")
	}

	launchManifest, err := operation.instance.GetLaunchManifest()
	if err != nil {
		return err
	}
	filtered := make([]minecraft.Library, 0, len(launchManifest.Libraries))
	for _, lib := range launchManifest.Libraries {
		if !strings.HasPrefix(lib.Name, prefix) {
			filtered = append(filtered, lib)
		}
	}
	launchManifest.Libraries = filtered
	return nil
}
