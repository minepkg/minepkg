package java

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

type Factory struct {
	baseDir string
	http    *http.Client
}

func NewFactory(baseDir string) *Factory {
	return &Factory{
		baseDir,
		http.DefaultClient,
	}
}

// SetHTTPClient replaces the default http client with the given one
func (j *Factory) SetHTTPClient(c *http.Client) {
	j.http = c
}

func (j *Factory) Version(ctx context.Context, featureRelease uint8) (*Java, error) {
	fullName := fmt.Sprintf("%d-jre-openj9", featureRelease)

	entries, err := os.ReadDir(j.baseDir)
	if err != nil {
		return nil, err
	}

	p, err := filepath.Abs(filepath.Join(j.baseDir, fullName))
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.Name() == fullName {
			asset, err := readAssetFile(filepath.Join(p, "asset.json"))
			if err != nil {
				break
			}

			return &Java{
				dir:              p,
				asset:            asset,
				needsDownloading: false,
			}, nil
		}
	}

	// no cached version, downloading
	assets, err := j.getAssets(ctx, &AdoptAssetRequest{version: featureRelease})
	if err != nil {
		return nil, err
	}
	if len(assets) == 0 {
		return nil, fmt.Errorf("no java version found")
	}

	return &Java{dir: p, asset: &assets[0], needsDownloading: true}, nil
}

func readAssetFile(file string) (*AdoptAsset, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	asset := &AdoptAsset{}
	if err := json.NewDecoder(f).Decode(asset); err != nil {
		return nil, err
	}
	return asset, nil
}
