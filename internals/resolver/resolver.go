package resolver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/globals"
	"github.com/minepkg/minepkg/internals/resolver/providers"
	"github.com/minepkg/minepkg/pkg/manifest"
)

var (
	// ErrNoGlobalReqs is returned when GlobalReqs was not set
	ErrNoGlobalReqs  = errors.New("no GlobalReqs set. They are required to resolve")
	ErrUnexpectedEOF = errors.New("file stream closed unexpectedly")
)

// ErrNoMatchingRelease is returned if a wanted releaseendency (package) could not be resolved given the requirements
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
	Providers         map[string]providers.Provider
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
		Providers:      make(map[string]providers.Provider, 2),
		downloadWg:     sync.WaitGroup{},
	}

	resolver.Providers["minepkg"] = &providers.MinepkgProvider{
		Client: globals.ApiClient,
	}

	resolver.Providers["https"] = &providers.HttpProvider{
		Client: http.DefaultClient,
	}

	return resolver
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
	throttle := make(chan interface{}, 24) // 24 is good

	throttleDownload := make(chan interface{}, 8)

	asyncResolve := func(dependency *manifest.InterpretedDependency) {
		throttle <- nil
		// t := time.Now()
		ctx, cancel := context.WithTimeout(ctx, time.Minute*2)
		defer cancel()

		result, err := r.resolveSingle(ctx, dependency)
		if err != nil {
			errorC <- err
			return
		}

		resultsC <- result
		<-throttle
	}

	batchResolve := func(dependencies []*manifest.InterpretedDependency) {
		for _, dep := range dependencies {
			// already resolved
			// TODO: check if this version is newer
			_, ok := r.Resolved[dep.Name]
			if ok {
				continue
			}
			resolving++

			r.Resolved[dep.Name] = nil

			go asyncResolve(dep)
		}
	}

	download := func(resolved *Resolved) {
		throttleDownload <- nil
		r.downloadWg.Add(1)
		resolved.Fetch(context.TODO())
		r.downloadWg.Done()
		<-throttleDownload
	}

	for {
		// start resolving the 1st level
		batchResolve(dependencies)

		if resolving == 0 {
			return nil
		}

		select {
		case err := <-errorC:
			return err
		case resolved := <-resultsC:
			resolving--
			lock := resolved.Result.Lock()

			if isDev {
				lock.IsDev = true
			}
			r.Resolved[lock.Name] = lock
			r.BetterResolved = append(r.BetterResolved, resolved)

			// download this package
			if r.AlsoDownload {
				go func(resolved *Resolved) {
					download(resolved)
					r.notifySubscribers(resolved)
				}(resolved)
			} else {
				r.notifySubscribers(resolved)
			}

			// resolve the dependencies of this package
			batchResolve(resolved.Result.Dependencies())
		}
	}
}

func (r *Resolver) resolveSingle(ctx context.Context, dependency *manifest.InterpretedDependency) (*Resolved, error) {
	provider, ok := r.Providers[dependency.Provider]
	if !ok {
		return nil, fmt.Errorf("%s needs %s as install provider which is not supported", dependency.Name, dependency.Provider)
	}

	request := r.providerRequest(dependency)

	result, err := provider.Resolve(ctx, r.providerRequest(dependency))
	if err != nil {
		return nil, err
	}

	resolved := &Resolved{
		Request:  request,
		Result:   result,
		provider: provider,
	}

	return resolved, nil
}

func (r *Resolver) providerRequest(dep *manifest.InterpretedDependency) *providers.Request {
	return &providers.Request{
		Dependency:   dep,
		Requirements: r.GlobalReqs,
	}
}

type Resolved struct {
	Request *providers.Request
	Result  providers.Result

	provider         providers.Provider
	bytesTransferred uint64
	totalBytes       uint64
}

func (r *Resolved) Fetch(ctx context.Context) error {
	// TODO: check cache here
	reader, size, err := r.provider.Fetch(ctx, r.Result)
	if err != nil {
		return err
	}
	r.totalBytes = uint64(size)

	src := io.TeeReader(reader, &WriteCounter{&r.bytesTransferred})
	dest, err := os.CreateTemp("", "minepkg")
	if err != nil {
		return err
	}
	if _, err := io.Copy(dest, src); err != nil {
		return err
	}

	if r.bytesTransferred != r.totalBytes {
		return ErrUnexpectedEOF
	}

	return nil
}

func (r *Resolved) Transferred() uint64 {
	if r.totalBytes == 0 {
		return 0
	}

	return r.bytesTransferred
}

func (r *Resolved) Size() uint64 {
	if r.totalBytes == 0 {
		return 0
	}

	return r.totalBytes
}

func (r *Resolved) Progress() float64 {
	if r.totalBytes == 0 {
		return 0
	}

	return float64(r.bytesTransferred) / float64(r.totalBytes)
}

// WriteCounter counts the number of bytes written to it.
type WriteCounter struct {
	Total *uint64 // Total # of bytes transferred
}

// Write implements the io.Writer interface.
func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	*wc.Total += uint64(n)
	return n, nil
}
