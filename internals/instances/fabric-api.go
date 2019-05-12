package instances

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

const fabricBaseAPI string = "https://launchermeta.mojang.com/mc/game/version_manifest.json"

type fabricLoaderEntry struct {
	Loader struct {
		Separator string `json:"separator"`
		Build     int    `json:"build"`
		Maven     string `json:"maven"`
		Version   string `json:"version"`
		Stable    bool   `json:"stable"`
	} `json:"loader"`
	Mappings struct {
		GameVersion string `json:"gameVersion"`
		Separator   string `json:"separator"`
		Build       int    `json:"build"`
		Maven       string `json:"maven"`
		Version     string `json:"version"`
		Stable      bool   `json:"stable"`
	} `json:"mappings"`
}

func getFabricLoaderForGameVersion(mcVersion string) (*fabricLoaderEntry, error) {
	loaders := make([]fabricLoaderEntry, 0)
	res, err := http.Get("https://meta.fabricmc.net/v1/versions/loader/" + mcVersion)
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

	// TODO: version matching
	if len(loaders) == 0 {
		return nil, ErrorNoFabricLoader
	}
	matched := loaders[0]

	return &matched, nil
}
