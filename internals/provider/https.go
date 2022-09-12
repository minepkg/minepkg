package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/minepkg/minepkg/internals/pkgid"
	"github.com/minepkg/minepkg/pkg/manifest"
)

type HTTPSProvider struct {
	Client *http.Client
}

type httpsResult struct {
	cacheKey   string
	dependency *pkgid.ID
}

func NewHTTPSProvider() *HTTPSProvider {
	return &HTTPSProvider{
		Client: http.DefaultClient,
	}
}

func (h *httpsResult) Lock() *manifest.DependencyLock {
	lock := &manifest.DependencyLock{
		Name:     h.dependency.Name,
		Provider: h.dependency.Provider,
		Type:     "mod",
		URL:      h.dependency.Version,
		Version:  h.cacheKey,
	}

	return lock
}

func (h *httpsResult) Dependencies() []*manifest.InterpretedDependency {
	// there currently is no way to resolve these … so empty they stay
	return []*manifest.InterpretedDependency{}
}

func (h *HTTPSProvider) Name() string { return "https" }

func (h *HTTPSProvider) Resolve(ctx context.Context, request *Request) (Result, error) {
	if !strings.HasPrefix(request.Dependency.Version, "https://") {
		return nil, fmt.Errorf("refusing to resolve non https url %s", request.Dependency.Version)
	}
	req, err := http.NewRequest("HEAD", request.Dependency.Version, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	res, err := h.Client.Do(req)
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

	return &httpsResult{dependency: request.Dependency, cacheKey: cacheKey}, nil
}

func (h *HTTPSProvider) Fetch(ctx context.Context, toFetch Result) (io.Reader, int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", toFetch.Lock().URL, nil)
	if err != nil {
		return nil, 0, err
	}

	fileRes, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}

	return fileRes.Body, int(fileRes.ContentLength), nil
}
