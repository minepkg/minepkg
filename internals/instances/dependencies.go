package instances

import (
	"archive/zip"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fiws/minepkg/internals/downloadmgr"
	"github.com/fiws/minepkg/pkg/manifest"
)

// UpdateLockfileDependencies resolves all dependencies
func (i *Instance) UpdateLockfileDependencies(ctx context.Context) error {
	if i.Lockfile == nil {
		i.Lockfile = manifest.NewLockfile()
		if err := i.UpdateLockfileRequirements(ctx); err != nil {
			return err
		}
	} else {
		i.Lockfile.ClearDependencies()
	}

	// add our companion mod if not disabled by user or non fabric
	if i.Manifest.Requirements.MinepkgCompanion != "none" && i.Manifest.PlatformString() == "fabric" {
		// just add it to the manifest. this is pretty hacky
		v := "latest"
		if i.Manifest.Requirements.MinepkgCompanion != "" {
			v = i.Manifest.Requirements.MinepkgCompanion
		}
		i.Manifest.AddDependency("minepkg-companion", v)
	}

	res := NewResolver(i.MinepkgAPI, i.Lockfile.PlatformLock())
	err := res.ResolveManifest(i.Manifest)

	if err != nil {
		return err
	}
	for _, release := range res.Resolved {
		i.Lockfile.AddDependency(&manifest.DependencyLock{
			Project:  release.Package.Name,
			Version:  release.Package.Version,
			Type:     release.Package.Type,
			IPFSHash: release.Meta.IPFSHash,
			Sha256:   release.Meta.Sha256,
			URL:      release.DownloadURL(),
		})
	}

	// This is kind of a hack
	// remove minepkg-companion if it was there
	i.Manifest.RemoveDependency("minepkg-companion")

	return nil
}

// FindMissingDependencies returns all dependencies that are not present
func (i *Instance) FindMissingDependencies() ([]*manifest.DependencyLock, error) {
	missing := make([]*manifest.DependencyLock, 0)

	deps := i.Lockfile.Dependencies

	for _, dep := range deps {
		if dep.URL == "" {
			continue // skip dependencies without download url
		}
		p := filepath.Join(dep.Project, dep.Version+dep.FileExt())
		if _, err := os.Stat(filepath.Join(i.CacheDir(), p)); os.IsNotExist(err) {
			missing = append(missing, dep)
		}
	}

	return missing, nil
}

// LinkDependencies links or copies all missing dependencies into the mods folder
func (i *Instance) LinkDependencies() error {
	files, err := ioutil.ReadDir(i.ModsDir())
	if err != nil {
		if os.IsNotExist(err) == true {
			if err := os.MkdirAll(i.ModsDir(), os.ModePerm); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	for _, f := range files {
		os.Remove(filepath.Join(i.ModsDir(), f.Name()))
	}

	for _, dep := range i.Lockfile.Dependencies {
		// skip packages with no binary
		if dep.URL == "" {
			continue
		}
		from := filepath.Join(i.CacheDir(), dep.Project, dep.Version+dep.FileExt())
		to := filepath.Join(i.ModsDir(), dep.Filename())

		// extract modpack content and stuff, don't symlink them into the mods folder
		if dep.Type == manifest.DependencyLockTypeModpack {
			i.handleModpackDependencyCopy(dep)
			continue
		}

		// windows required admin permissions for symlinks (yea …)
		if runtime.GOOS == "windows" {
			// TODO: fallback to copy
			err = os.Link(from, to)
		} else {
			err = os.Symlink(from, to)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *Instance) handleModpackDependencyCopy(dep *manifest.DependencyLock) error {

	modpackPath := filepath.Join(i.CacheDir(), dep.Project, dep.Version+".zip")
	zipReader, err := zip.OpenReader(modpackPath)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	skipPrefixes := []string{}
	createdDirs := make(map[string]interface{})

outer:
	for _, f := range zipReader.File {

		// make sure zip only contains valid paths
		if err := sanitizeExtractPath(f.Name, i.McDir()); err != nil {
			return err
		}

		// get a relative path – used for name matching and stuff
		relative, err := filepath.Rel(i.McDir(), filepath.Join(i.McDir(), f.Name))
		if err != nil {
			return err
		}

		// skipping already created save directories
		for _, skip := range skipPrefixes {
			if strings.HasPrefix(relative, skip) {
				continue outer
			}
		}

		// not sure if this is optimal...
		if f.FileInfo().IsDir() {
			continue outer
		}

		relativeDir := filepath.Dir(relative)
		// TODO: is this also / on windows?
		dirs := strings.Split(relativeDir, "/")

		for n := range dirs {
			// this gets us `saves`, `saves/test-world`, `saves/test-world/DIM1` etc.
			dir := strings.Join(dirs[0:n+1], "/")

			// see if we already created that dir. skip creating in that case
			if _, ok := createdDirs[dir]; ok == true {
				continue
			}

			err := os.Mkdir(filepath.Join(i.McDir(), dir), os.ModePerm)
			createdDirs[dir] = nil

			switch {
			case err != nil && !os.IsExist(err):
				// unknown error, return it
				return err
			case err != nil && strings.HasPrefix(dir, "saves") && dir != "saves":
				// we tried to create a save dir (eg, `saves/test-world`) and it already exists, exclude it
				skipPrefixes = append(skipPrefixes, dir)
				continue outer
			}
		}

		// all directories for this file are here, we can finally copy the file
		rc, err := f.Open()
		if err != nil {
			return err
		}
		target, err := os.Create(filepath.Join(i.McDir(), f.Name))
		if err != nil {
			return err
		}

		_, err = io.Copy(target, rc)
		if err != nil {
			return err
		}

		rc.Close()
	}
	return nil
}

// EnsureDependencies downloads missing dependencies
func (i *Instance) EnsureDependencies(ctx context.Context) error {
	missingFiles, err := i.FindMissingDependencies()
	if err != nil {
		return err
	}

	mgr := downloadmgr.New()
	for _, m := range missingFiles {
		p := filepath.Join(i.CacheDir(), m.Project, m.Version+m.FileExt())
		mgr.Add(downloadmgr.NewHTTPItem(m.URL, p))
	}

	if err := mgr.Start(ctx); err != nil {
		return err
	}
	if err := i.LinkDependencies(); err != nil {
		return err
	}
	return nil
}
