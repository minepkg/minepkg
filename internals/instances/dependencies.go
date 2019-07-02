package instances

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fiws/minepkg/internals/downloadmgr"

	"github.com/fiws/minepkg/pkg/manifest"
)

// UpdateLockfileDependencies resolves all dependencies
func (i *Instance) UpdateLockfileDependencies() error {
	if i.Lockfile == nil {
		i.Lockfile = manifest.NewLockfile()
		if err := i.UpdateLockfileRequirements(context.TODO()); err != nil {
			return err
		}
	} else {
		i.Lockfile.ClearDependencies()
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
			IPFSHash: release.Meta.IPFSHash,
			Sha256:   release.Meta.Sha256,
			URL:      release.DownloadURL(),
		})
	}

	return nil
}

// FindMissingDependencies returns all dependencies that are not present
func (i *Instance) FindMissingDependencies() ([]*manifest.DependencyLock, error) {
	missing := make([]*manifest.DependencyLock, 0)

	deps := i.Lockfile.Dependencies
	cacheDir := filepath.Join(i.Directory, "cache")

	for _, dep := range deps {
		if dep.URL == "" {
			continue // skip dependencies without download url
		}
		p := filepath.Join(dep.Project, dep.Version+".jar")
		if _, err := os.Stat(filepath.Join(cacheDir, p)); os.IsNotExist(err) {
			missing = append(missing, dep)
		}
	}

	return missing, nil
}

// LinkDependencies links or copies all missing dependencies into the mods folder
func (i *Instance) LinkDependencies() error {
	cacheDir := filepath.Join(i.Directory, "cache")

	files, err := ioutil.ReadDir(i.ModsDirectory)
	if err != nil {
		if os.IsNotExist(err) == true {
			os.MkdirAll(i.ModsDirectory, os.ModePerm)
		} else {
			return err
		}
	}

	for _, f := range files {
		if strings.HasPrefix("custom-", f.Name()) {
			fmt.Println("ignoring custom mod " + f.Name())
		} else {
			os.Remove(filepath.Join(i.ModsDirectory, f.Name()))
		}
	}

	for _, dep := range i.Lockfile.Dependencies {
		// skip packages with no binary
		if dep.URL == "" {
			continue
		}
		from := filepath.Join(cacheDir, dep.Project, dep.Version+".jar")
		to := filepath.Join(i.ModsDirectory, dep.Filename())

		// windows required admin permissions for symlinks (yea …)
		if runtime.GOOS == "windows" {
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

// EnsureDependencies downloads missing dependencies
func (i *Instance) EnsureDependencies(ctx context.Context) error {
	cacheDir := filepath.Join(i.Directory, "cache")

	missingFiles, err := i.FindMissingDependencies()
	if err != nil {
		return err
	}

	mgr := downloadmgr.New()
	for _, m := range missingFiles {
		p := filepath.Join(cacheDir, m.Project, m.Version+".jar")
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
