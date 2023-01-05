package patch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/minepkg/minepkg/internals/minecraft"
)

// MergeLaunchManifest is a patch operation that merges a manifest into the instance manifest.
type MergeLaunchManifest struct{}

type mergeLaunchArgs struct {
	Manifest *minecraft.LaunchManifest `json:"manifest,omitempty"`
	URL      string                    `json:"url,omitempty"`
}

func (r *MergeLaunchManifest) Apply(ctx context.Context, operation *PatchOperation) error {
	if operation.instance == nil {
		return fmt.Errorf("operation instance is nil")
	}

	var args mergeLaunchArgs
	if err := UnmarshalArgs(operation, &args); err != nil {
		return err
	}

	manifest := args.Manifest
	if args.URL != "" {

		if manifest != nil {
			return fmt.Errorf("cannot specify both manifest and url")
		}

		// fetch manifest
		res, err := http.Get(args.URL)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		if err := json.NewDecoder(res.Body).Decode(&manifest); err != nil {
			return err
		}
	}

	launchManifest, err := operation.instance.GetLaunchManifest()
	if err != nil {
		return err
	}

	minecraft.MergeManifests(launchManifest, manifest)
	operation.instance.SetLaunchManifest(launchManifest)
	return nil
}
