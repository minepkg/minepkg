package instances

import (
	"io/ioutil"

	"github.com/minepkg/minepkg/pkg/manifest"
	"github.com/pelletier/go-toml"
)

func LockfileFromPath(p string) (*manifest.Lockfile, error) {
	rawLockfile, err := ioutil.ReadFile(p)
	if err != nil {
		// this is bad
		return nil, err
	}

	lockfile := manifest.Lockfile{}
	err = toml.Unmarshal(rawLockfile, &lockfile)
	if err != nil {
		return nil, err
	}
	if lockfile.Dependencies == nil {
		lockfile.Dependencies = make(map[string]*manifest.DependencyLock)
	}

	return &lockfile, nil
}
