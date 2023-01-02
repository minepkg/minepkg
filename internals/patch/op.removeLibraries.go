package patch

import (
	"context"
	"fmt"
	"strings"

	"github.com/minepkg/minepkg/internals/minecraft"
)

type RemoveLibraries struct {
	args struct {
		Prefix string `json:"prefix"`
	}
}

func (r *RemoveLibraries) Args() any {
	return &r.args
}

func (r *RemoveLibraries) Apply(ctx context.Context, operation *PatchOperation) error {
	if operation.instance == nil {
		return fmt.Errorf("operation instance is nil")
	}

	prefix := r.args.Prefix
	if prefix == "" {
		return fmt.Errorf("prefix is empty")
	}

	launchManifest, err := operation.instance.GetLaunchManifest()
	if err != nil {
		return err
	}
	filtered := make([]minecraft.Lib, 0, len(launchManifest.Libraries))
	for _, lib := range launchManifest.Libraries {
		if !strings.HasPrefix(lib.Name, prefix) {
			filtered = append(filtered, lib)
		}
	}
	launchManifest.Libraries = filtered
	return nil
}
