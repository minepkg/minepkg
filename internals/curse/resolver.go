package curse

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/fiws/minepkg/internals/manifest"
)

// Resolver resolves given the mods of given dependencies
type Resolver struct {
	Resolved map[uint32]manifest.ResolvedMod
}

// ResolveMultiple resolved multiple mods
func (r *Resolver) ResolveMultiple(ids []uint32, version string) error {
	for _, id := range ids {
		err := r.Resolve(id, version)
		if err != nil {
			return err
		}
	}
	return nil
}

// ResolvePackage accepts mods or modepacks to resolve
func (r *Resolver) ResolvePackage(mod *Mod, version string) error {
	// resolve mods the usuall way
	if mod.PackageType != PackageTypeModpack {
		return r.Resolve(mod.ID, version)
	}

	// Modpacks require a bit of extra logic
	// dependencies are in the zip file
	modFiles, _ := FetchModFiles(mod.ID)
	matchingRelease := FindRelease(modFiles, version)

	res, err := http.Get(matchingRelease.DownloadURL)
	if err != nil {
		panic(err)
	}
	tmpfile, err := ioutil.TempFile("", matchingRelease.FileNameOnDisk)
	if err != nil {
		panic(err)
	}
	io.Copy(tmpfile, res.Body)
	man := extractManifestDeps(tmpfile.Name())

	depIds := make([]uint32, len(man.Files))
	for i, dep := range man.Files {
		depIds[i] = dep.ProjectID
	}

	return r.ResolveMultiple(depIds, version)
}

// Resolve find all dependencies from the given `id`
// and adds it to the `resolved` map. Nothing is returned
func (r *Resolver) Resolve(id uint32, version string) error {
	var resolve func(id uint32) error
	resolve = func(id uint32) error {
		_, ok := r.Resolved[id]
		if ok == true {
			return nil
		}

		modFiles, _ := FetchModFiles(id)
		matchingRelease := FindRelease(modFiles, version)
		if matchingRelease == nil {
			return fmt.Errorf("Mod with id %d does not support mc version %s", id, version)
		}

		r.Resolved[id] = manifest.ResolvedMod{
			DownloadURL: matchingRelease.DownloadURL,
			FileName:    matchingRelease.FileNameOnDisk,
		}
		var wg sync.WaitGroup
		for _, dependency := range matchingRelease.Dependencies {
			if dependency.Type == DependencyTypeRequired {
				wg.Add(1)
				go func(id uint32) {
					defer wg.Done()
					resolve(id)
				}(dependency.AddOnID)
			}
		}
		wg.Wait()
		return nil
	}

	return resolve(id)
}

// NewResolver returns a new resolver
func NewResolver() *Resolver {
	return &Resolver{Resolved: make(map[uint32]manifest.ResolvedMod)}
}

type curseManifest struct {
	Files []struct {
		ProjectID uint32 `json:"projectID"`
		FileID    uint64 `json:"fileID"`
		Required  bool   `json:"required"`
	} `json:"files"`
}

func extractManifestDeps(filename string) *curseManifest {
	r, err := zip.OpenReader(filename)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	manifest := curseManifest{}
	for _, f := range r.File {
		if f.Name == "manifest.json" {

			rc, err := f.Open()
			if err != nil {
				panic(err)
			}
			raw, err := ioutil.ReadAll(rc)
			if err != nil {
				panic(err)
			}
			json.Unmarshal(raw, &manifest)
			rc.Close()
		}
	}
	return &manifest
}
