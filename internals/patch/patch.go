package patch

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/minepkg/minepkg/internals/instances"
	"gopkg.in/yaml.v3"
)

type Patch struct {
	// Name is the name of the patch
	Name string `json:"name"`
	// Description is a description of the patch
	Description string `json:"description"`

	// Patches is the list of patches
	Patches []PatchOperation `json:"patches"`
}

type PatchOperation struct {
	// Action is the action to perform
	Action string `json:"action"`
	// With are the arguments for the action
	With yaml.Node `json:"with"`

	// instance is the instance the patch is applied to, this is set by the patcher
	instance *instances.Instance
}

type Operator interface {
	// Apply applies the patch to the given instance
	Apply(ctx context.Context, operation *PatchOperation) error
}

var (
	Operations = map[string]Operator{
		"removeLibraries":      &RemoveLibraries{},
		"addLibraries":         &AddLibraries{},
		"mergeLaunchManifest":  &MergeLaunchManifest{},
		"mergeMinepkgManifest": &MergeMinepkgManifest{},
	}
)

func PatchInstance(ctx context.Context, patch *Patch, instance *instances.Instance) error {
	for _, operation := range patch.Patches {
		operation.instance = instance
		patcher := Operations[operation.Action]
		if patcher == nil {
			return fmt.Errorf("unknown patcher %q", operation.Action)
		}

		if err := patcher.Apply(ctx, &operation); err != nil {
			return err
		}
	}
	return nil
}

// FetchPatch fetches a patch from the given location (can be a URL or a local path)
func FetchPatch(ctx context.Context, location string) (*Patch, error) {
	if strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://") {
		return FetchPatchFromURL(ctx, location)
	}

	return FetchPatchFromFile(ctx, location)
}

// FetchPatchFromURL fetches a patch from a URL
func FetchPatchFromURL(ctx context.Context, url string) (*Patch, error) {
	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch patch: %s", response.Status)
	}

	var patch Patch
	if err := yaml.NewDecoder(response.Body).Decode(&patch); err != nil {
		return nil, err
	}

	return &patch, nil
}

// FetchPatchFromFile fetches a patch from a local file
func FetchPatchFromFile(ctx context.Context, path string) (*Patch, error) {
	// try to open the file
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var patch Patch
	if err := yaml.NewDecoder(file).Decode(&patch); err != nil {
		return nil, err
	}

	return &patch, nil
}

func UnmarshalArgs(op *PatchOperation, args any) error {
	if err := op.With.Decode(args); err != nil {
		return fmt.Errorf("failed to unmarshal patcher arguments: %w", err)
	}
	return nil
}
