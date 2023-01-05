package instances

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/minepkg/minepkg/internals/minecraft"
)

// FindMissingLibraries returns all missing assets
func (i *Instance) FindMissingLibraries(man *minecraft.LaunchManifest) ([]minecraft.Library, error) {
	missing := make([]minecraft.Library, 0)

	libs := minecraft.RequiredLibraries(man.Libraries)
	globalDir := i.LibrariesDir()

	for _, lib := range libs {
		path := filepath.Join(globalDir, lib.Filepath())
		if _, err := os.Stat(path); err == nil {
			continue
		}

		missing = append(missing, lib)
	}

	return missing, nil
}

// FindMissingAssets returns all missing assets
func (i *Instance) FindMissingAssets(man *minecraft.LaunchManifest) ([]minecraft.AssetObject, error) {
	assets := minecraft.AssetIndex{}

	assetJSONPath := filepath.Join(i.AssetsDir(), "indexes", man.Assets+".json")
	buf, err := ioutil.ReadFile(assetJSONPath)
	if err != nil {
		res, err := http.Get(man.AssetIndex.URL)
		if err != nil {
			return nil, err
		}

		buf, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		os.MkdirAll(filepath.Join(i.AssetsDir(), "indexes"), os.ModePerm)
		err = ioutil.WriteFile(assetJSONPath, buf, 0666)
		if err != nil {
			return nil, err
		}
	}
	json.Unmarshal(buf, &assets)

	missing := make([]minecraft.AssetObject, 0)

	for _, asset := range assets.Objects {
		file := filepath.Join(i.AssetsDir(), "objects", asset.UnixPath())
		if _, err := os.Stat(file); os.IsNotExist(err) {
			missing = append(missing, asset)
		}
	}

	return missing, nil
}
