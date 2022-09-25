package provider

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrProviderNotFound    = errors.New("provider not found")
	ErrProviderUnsupported = errors.New("provider does not support this operation")
	ErrURLNotConvertible   = errors.New("url not convertible")
)

type Store struct {
	providers map[string]Provider
}

func NewStore(providers map[string]Provider) *Store {
	return &Store{providers}
}

func (s *Store) Add(provider Provider) {
	s.providers[provider.Name()] = provider
}

func (s *Store) Has(provider string) bool {
	_, ok := s.providers[provider]
	return ok
}

// Get returns a provider by name
func (s *Store) Get(name string) (Provider, bool) {
	provider, ok := s.providers[name]
	return provider, ok
}

func (s *Store) Resolve(ctx context.Context, request *Request) (Result, error) {
	provider, ok := s.providers[request.Dependency.Provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, request.Dependency.Provider)
	}

	res, err := provider.Resolve(ctx, request)
	if err != nil {
		betterErr := fmt.Errorf("%s could not resolve %s: %w", provider.Name(), request.Dependency.LegacyID(), err)
		return nil, betterErr
	}

	return res, nil
}

func (s *Store) ResolveLatest(ctx context.Context, request *Request) (Result, error) {
	provider, ok := s.providers[request.Dependency.Provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, request.Dependency.Provider)
	}

	latestResolver, ok := provider.(LatestResolver)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderUnsupported, request.Dependency.Provider)
	}

	res, err := latestResolver.ResolveLatest(ctx, request)
	if err != nil {
		betterErr := fmt.Errorf("%s could not resolve %s: %w", provider.Name(), request.Dependency.Name, err)
		return nil, betterErr
	}

	return res, nil
}

func (s *Store) ConvertURL(ctx context.Context, url string) (string, error) {
	convertProvider := make(map[string]URLConverter)

	for _, provider := range s.providers {
		if converter, ok := provider.(URLConverter); ok {
			convertProvider[provider.Name()] = converter
		}
	}

	for _, converter := range convertProvider {
		if converter.CanConvertURL(url) {
			return converter.ConvertURL(ctx, url)
		}
	}

	return "", ErrURLNotConvertible
}
