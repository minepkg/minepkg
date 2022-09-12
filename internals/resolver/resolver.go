package resolver

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/pkgid"
	"github.com/minepkg/minepkg/internals/provider"
	"github.com/minepkg/minepkg/pkg/manifest"
)

var (
	// ErrNoGlobalReqs is returned when GlobalReqs was not set
	ErrNoGlobalReqs          = errors.New("no GlobalReqs set. They are required to resolve")
	ErrUnexpectedEOF         = errors.New("file stream closed unexpectedly")
	ErrProviderDidNotResolve = errors.New("provider did not return a result")
)

// ErrNoMatchingRelease is returned if a wanted release dependency (package) could not be resolved given the requirements
type ErrNoMatchingRelease struct {
	// Package is the name of the package that can not be resolved
	Package string
	// Requirements are the requirements for this package to resolve (eg. minecraft version)
	Requirements *api.RequirementQuery
	// Parent is the release of the package that required this one (if any)
	Parent *api.Release
}

func (e *ErrNoMatchingRelease) Error() string {
	parent := "(root)"
	if e.Parent != nil {
		parent = e.Parent.Package.Name
	}
	return fmt.Sprintf(
		"No Release found for %s. Details: \n\tPlatform: %s\n\tPackage: %s\n\tVersion: %s\n\tMinecraft Version: %s\n\tDependency of: %s",
		e.Package,
		e.Requirements.Platform,
		e.Package,
		e.Requirements.Version,
		e.Requirements.Minecraft,
		parent,
	)
}

// Resolver resolves given the mods of given dependencies
type Resolver struct {
	Resolved       map[string]*manifest.DependencyLock
	BetterResolved []*Resolved
	manifest       *manifest.Manifest
	GlobalReqs     manifest.PlatformLock
	// IgnoreVersion will make the resolver ignore all version requirements and just fetch the latest version for everything
	IgnoreVersion bool
	// IncludeDev includes dev.dependencies
	IncludeDev   bool
	AlsoDownload bool

	resolvingFinished bool
	downloadWg        sync.WaitGroup
	subscribers       []chan *Resolved
	ProviderStore     *provider.Store
}

// New returns a new resolver
func New(man *manifest.Manifest, platformLock manifest.PlatformLock) *Resolver {
	resolver := &Resolver{
		Resolved:       make(map[string]*manifest.DependencyLock),
		BetterResolved: make([]*Resolved, 0, len(man.Dependencies)),
		manifest:       man,
		GlobalReqs:     platformLock,
		IgnoreVersion:  false,
		IncludeDev:     true,
		AlsoDownload:   false, // TODO: set to true when working properly
		downloadWg:     sync.WaitGroup{},
	}

	return resolver
}

func (r *Resolver) SetProviderStore(store *provider.Store) {
	r.ProviderStore = store
}

func (r *Resolver) ResolveFinished() bool {
	return r.resolvingFinished
}

func (r *Resolver) Subscribe() chan *Resolved {
	subChannel := make(chan *Resolved)
	r.subscribers = append(r.subscribers, subChannel)

	return subChannel
}

func (r *Resolver) notifySubscribers(result *Resolved) {
	for _, subscription := range r.subscribers {
		subscription <- result
	}
}

func (r *Resolver) closeSubscribers() {
	for _, subscription := range r.subscribers {
		close(subscription)
	}
}

// Resolve resolves a manifest
func (r *Resolver) Resolve(ctx context.Context) error {
	man := r.manifest
	defer r.closeSubscribers()

	if r.GlobalReqs == nil {
		return ErrNoGlobalReqs
	}

	if err := r.ResolveDependencies(ctx, man.InterpretedDependencies(), false); err != nil {
		return err
	}

	if r.IncludeDev {
		if err := r.ResolveDependencies(ctx, man.InterpretedDevDependencies(), false); err != nil {
			return err
		}
	}

	r.resolvingFinished = true

	if r.AlsoDownload {
		r.downloadWg.Wait()
	}

	return nil
}

// Resolve resolves all given dependencies
func (r *Resolver) ResolveDependencies(ctx context.Context, dependencies []*manifest.InterpretedDependency, isDev bool) error {

	resolving := 0
	resultsC := make(chan *Resolved)
	errorC := make(chan error)

	queryQueue := make(chan interface{}, 24) // 24 is good

	asyncResolve := func(dependency *manifest.InterpretedDependency, root *manifest.DependencyLock) {
		queryQueue <- nil
		// t := time.Now()
		ctx, cancel := context.WithTimeout(ctx, time.Minute*2)
		defer cancel()

		result, err := r.resolveSingle(ctx, dependency, root)
		if err != nil {
			errorC <- err
			return
		}

		resultsC <- result
		<-queryQueue
	}

	batchResolve := func(dependencies []*manifest.InterpretedDependency, root *manifest.DependencyLock) {
		for _, dep := range dependencies {
			// already resolved
			// TODO: check if this version is newer
			_, ok := r.Resolved[dep.Name]
			if ok {
				continue
			}
			resolving++

			r.Resolved[dep.Name] = nil

			go asyncResolve(dep, root)
		}
	}

	for {
		// start resolving the 1st level
		batchResolve(dependencies, nil)

		if resolving == 0 {
			return nil
		}

		select {
		case err := <-errorC:
			return err
		case resolved := <-resultsC:
			resolving--
			lock := resolved.result.Lock()

			if isDev {
				lock.IsDev = true
			}
			r.Resolved[lock.Name] = lock
			r.BetterResolved = append(r.BetterResolved, resolved)

			r.notifySubscribers(resolved)

			// resolve the dependencies of this package
			batchResolve(resolved.result.Dependencies(), resolved.result.Lock())
		}
	}
}

func (r *Resolver) resolveSingle(ctx context.Context, dependency *manifest.InterpretedDependency, root *manifest.DependencyLock) (*Resolved, error) {
	provider, ok := r.ProviderStore.Get(dependency.Provider)
	if !ok {
		return nil, fmt.Errorf("%s needs %s as install provider which is not supported", dependency.Name, dependency.Provider)
	}

	request := r.providerRequest(dependency.ID, root)

	if r.ProviderStore == nil {
		return nil, errors.New("no provider store")
	}
	result, err := r.ProviderStore.Resolve(ctx, request)
	if err != nil {
		return nil, err
	}

	if result == nil || result.Lock() == nil {
		return nil, ErrProviderDidNotResolve
	}

	resolved := &Resolved{
		Key:      dependency.Name,
		Request:  request,
		result:   result,
		provider: &provider,
	}

	return resolved, nil
}

func (r *Resolver) providerRequest(dep *pkgid.ID, root *manifest.DependencyLock) *provider.Request {
	return &provider.Request{
		Dependency:   dep,
		Requirements: r.GlobalReqs,
		Root:         root,
	}
}
