package instances

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/minepkg/minepkg/internals/downloadmgr"
	"github.com/minepkg/minepkg/internals/pack"
	"github.com/minepkg/minepkg/internals/resolver"
	"github.com/minepkg/minepkg/pkg/manifest"
)

func (i *Instance) GetResolver(ctx context.Context) (*resolver.Resolver, error) {
	if i.Lockfile == nil {
		i.Lockfile = manifest.NewLockfile()
		if err := i.UpdateLockfileRequirements(ctx); err != nil {
			return nil, err
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

	res := resolver.New(i.Manifest, i.Lockfile.PlatformLock())
	res.ProviderStore = i.ProviderStore
	// only include dev dependencies if this instance was created from a working directory
	// (eg. typing "minepkg launch" in a directory with a minepkg.toml)
	res.IncludeDev = i.isFromWd
	// res.AlsoDownload = true

	return res, nil
}

// UpdateLockfileDependencies resolves all dependencies
func (i *Instance) UpdateLockfileDependencies(ctx context.Context) error {
	resolver, err := i.GetResolver(ctx)
	if err != nil {
		return err
	}
	if err := resolver.Resolve(ctx); err != nil {
		return err
	}

	for _, lock := range resolver.Resolved {
		i.Lockfile.AddDependency(lock)
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
		p := filepath.Join(dep.Name, dep.Version+dep.FileExt())
		if _, err := os.Stat(filepath.Join(i.PackageCacheDir(), p)); os.IsNotExist(err) {
			missing = append(missing, dep)
		}
	}

	return missing, nil
}

// LinkDependencies links or copies all missing dependencies into the mods folder
func (i *Instance) LinkDependencies() error {
	files, err := ioutil.ReadDir(i.ModsDir())
	if err != nil {
		if os.IsNotExist(err) {
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
		from := filepath.Join(i.PackageCacheDir(), dep.Name, dep.Version+dep.FileExt())
		to := filepath.Join(i.ModsDir(), dep.Filename())

		// extract modpack content and stuff, don't symlink them into the mods folder
		if dep.Type == manifest.DependencyLockTypeModpack {
			if err := i.handleModpackDependencyCopy(dep); err != nil {
				return err
			}
			continue
		}

		copyFallback := false
		// windows required admin permissions for symlinks (yea …)
		if runtime.GOOS == "windows" {
			if !copyFallback {
				err = os.Link(from, to)
				// first time link fails, we just fallback to copying
				if err != nil {
					copyFallback = true
				}
			}
			if copyFallback {
				err = copyFileContents(from, to)
			}
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

	modpackPath := filepath.Join(i.PackageCacheDir(), dep.Name, dep.Version+".zip")
	pkg, err := pack.Open(modpackPath)
	if err != nil {
		return err
	}
	defer pkg.Close()
	return pkg.ExtractModpack(i.McDir())
}

// EnsureDependencies downloads missing dependencies
func (i *Instance) EnsureDependencies(ctx context.Context) error {
	missingFiles, err := i.FindMissingDependencies()
	if err != nil {
		return err
	}

	mgr := downloadmgr.New()
	for _, m := range missingFiles {
		p := filepath.Join(i.PackageCacheDir(), m.Name, m.Version+m.FileExt())
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
