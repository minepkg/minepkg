package resolver

import (
	"github.com/minepkg/minepkg/internals/provider"
	"github.com/minepkg/minepkg/pkg/manifest"
)

type Resolved struct {
	Key     string
	Request *provider.Request
	result  provider.Result

	provider *provider.Provider
}

func (r *Resolved) Lock() *manifest.DependencyLock {
	lock := r.result.Lock()
	if r.Request.Root != nil {
		lock.Dependent = r.Request.Root.Name
	}

	return lock
}
