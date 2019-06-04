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

	"github.com/fiws/minepkg/pkg/api"

	"github.com/fiws/minepkg/pkg/manifest"
)

// UpdateLockfileDependencies resolves all dependencies
func (i *Instance) UpdateLockfileDependencies() error {
	if i.Lockfile == nil {
		i.Lockfile = manifest.NewLockfile()
	} else {
		i.Lockfile.ClearDependencies()
	}

	res := NewResolver(i.MinepkgAPI)
	err := res.ResolveManifest(i.Manifest)
	if err != nil {
		return err
	}
	for _, release := range res.Resolved {
		i.Lockfile.AddDependency(&manifest.DependencyLock{
			Project:  release.Project,
			Version:  release.Version,
			IPFSHash: release.IPFSHash,
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

// Resolver resolves given the mods of given dependencies
type Resolver struct {
	Resolved map[string]*api.Release
	client   *api.MinepkgAPI
}

// NewResolver returns a new resolver
func NewResolver(client *api.MinepkgAPI) *Resolver {
	return &Resolver{Resolved: make(map[string]*api.Release), client: client}
}

// ResolveManifest resolves a manifest
func (r *Resolver) ResolveManifest(man *manifest.Manifest) error {

	for name, version := range man.Dependencies {
		release, err := r.client.FindRelease(context.TODO(), name, version)
		if err != nil {
			return err
		}
		err = r.ResolveSingle(release)
		if err != nil {
			return err
		}
	}

	return nil
}

// Resolve find all dependencies from the given `id`
// and adds it to the `resolved` map. Nothing is returned
func (r *Resolver) Resolve(releases []*api.Release) error {
	for _, release := range releases {
		r.ResolveSingle(release)
	}

	return nil
}

// ResolveSingle resolves all dependencies of a single release
func (r *Resolver) ResolveSingle(release *api.Release) error {

	r.Resolved[release.Project] = release
	// TODO: parallelize
	for _, d := range release.Dependencies {
		_, ok := r.Resolved[d.Name]
		if ok == true {
			return nil
		}
		r.Resolved[d.Name] = nil
		release, err := d.Resolve(context.TODO())
		if err != nil {
			return err
		}
		r.ResolveSingle(release)
	}

	return nil
}
