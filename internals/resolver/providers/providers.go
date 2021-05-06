package providers

import (
	"github.com/minepkg/minepkg/pkg/manifest"
)

type Provider interface {
	Resolve(request *Request) (Result, error)
}

type Request struct {
	Dependency   *manifest.InterpretedDependency
	Requirements manifest.PlatformLock

	ignoreVersionsFlag bool
}

type Result interface {
	Lock() *manifest.DependencyLock
	Dependencies() []*manifest.InterpretedDependency
}
