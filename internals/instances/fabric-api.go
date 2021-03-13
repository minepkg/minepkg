package instances

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type fabricLoaderVersion struct {
	Separator string `json:"separator"`
	Build     int    `json:"build"`
	Maven     string `json:"maven"`
	Version   string `json:"version"`
	Stable    bool   `json:"stable"`
}

type fabricMappingVersion struct {
	GameVersion string `json:"gameVersion"`
	Separator   string `json:"separator"`
	Build       int    `json:"build"`
	Maven       string `json:"maven"`
	Version     string `json:"version"`
	Stable      bool   `json:"stable"`
}

func getFabricLoaderVersions(ctx context.Context) ([]fabricLoaderVersion, error) {
	loaders := make([]fabricLoaderVersion, 0)
	res, err := fabricGet(ctx, "https://meta.fabricmc.net/v1/versions/loader")
	if err != nil {
		return nil, err
	}
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(buf, &loaders); err != nil {
		return nil, err
	}

	return loaders, nil
}

func getFabricMappingVersions(ctx context.Context) ([]fabricMappingVersion, error) {
	loaders := make([]fabricMappingVersion, 0)
	res, err := fabricGet(ctx, "https://meta.fabricmc.net/v1/versions/mappings")
	if err != nil {
		return nil, err
	}
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(buf, &loaders); err != nil {
		return nil, err
	}

	return loaders, nil
}

func fabricGet(ctx context.Context, url string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "minepkg (https://github.com/minepkg/minepkg)")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err == nil && res.StatusCode != 200 {
		return res, fmt.Errorf("fabric meta API did respond with unexpected status %s", res.Status)
	}
	return res, err
}
