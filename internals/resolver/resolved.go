package resolver

import (
	"github.com/minepkg/minepkg/internals/resolver/providers"
	"github.com/minepkg/minepkg/pkg/manifest"
)

type Resolved struct {
	Request *providers.Request
	result  providers.Result

	provider providers.Provider
}

func (r *Resolved) Lock() *manifest.DependencyLock {
	lock := r.result.Lock()
	if r.Request.Root != nil {
		lock.Dependent = r.Request.Root.Name
	}

	return lock
}
