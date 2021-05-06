package providers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/minepkg/minepkg/pkg/manifest"
)

type HttpProvider struct {
	Client *http.Client
}

type httpResult struct {
	cacheKey   string
	dependency *manifest.InterpretedDependency
}

func (h *httpResult) Lock() *manifest.DependencyLock {
	lock := &manifest.DependencyLock{
		Name:     h.dependency.Name,
		Provider: h.dependency.Provider,
		Type:     "mod",
		URL:      h.dependency.Source,
		Version:  h.cacheKey,
	}

	return lock
}

func (h *httpResult) Dependencies() []*manifest.InterpretedDependency {
	// there currently is no way to resolve these … so empty they stay
	return []*manifest.InterpretedDependency{}
}

func (h *HttpProvider) Resolve(request *Request) (Result, error) {
	res, err := h.Client.Head(request.Dependency.Source)
	if err != nil {
		return nil, err
	}

	etag := res.Header.Get("etag")
	lastModified := res.Header.Get("Last-Modified")

	cacheKey := strings.TrimPrefix(strings.TrimSuffix(etag, `"`), `"`)
	if etag == "" && lastModified != "" {
		cacheKey = base64.StdEncoding.EncodeToString([]byte(lastModified))
	}

	if cacheKey == "" {
		return nil, fmt.Errorf(
			"the http server of %s does not set a \"ETag\" or \"Last-Modified\" header, which currently is a requirement",
			request.Dependency.Name,
		)
	}

	return &httpResult{dependency: request.Dependency, cacheKey: cacheKey}, nil
}
