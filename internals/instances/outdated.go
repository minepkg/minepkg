package instances

import (
	"context"

	"github.com/minepkg/minepkg/internals/provider"
)

// type OutdatedList struct {

type OutdatedResult struct {
	Dependency Dependency
	Result     provider.Result
	Error      error
}

func (i Instance) Outdated(ctx context.Context) ([]OutdatedResult, error) {
	dependencies := i.GetDependencyList()

	// thread safe list of outdated dependencies
	outdated := make(chan *OutdatedResult, len(dependencies))

	// wg := sync.WaitGroup{}
	// wg.Add(len(dependencies))

	for _, dependency := range dependencies {
		go func(dependency Dependency) {
			// defer wg.Done()
			result, err := i.ProviderStore.ResolveLatest(ctx, dependency.ProviderRequest())
			outdated <- &OutdatedResult{
				Dependency: dependency,
				Result:     result,
				Error:      err,
			}
		}(dependency)
	}

	// build list
	outdatedList := make([]OutdatedResult, 0, len(dependencies))
	for range dependencies {
		outdatedList = append(outdatedList, *<-outdated)
	}

	return outdatedList, nil
}
