package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/minepkg/minepkg/internals/modrinth"
	"github.com/minepkg/minepkg/pkg/manifest"
)

var (
	ErrVersionNotSupported = errors.New("for modrinth version can only be '*', 'latest' a version id, sha1 or sha512")
	ErrVersionsNotFound    = errors.New("no versions found")
	ErrVersionHasNoFiles   = errors.New("version has no files")
)

type ModrinthProvider struct {
	Client *modrinth.Client
}

type modrinthResult struct {
	name    string
	version *modrinth.Version
	file    *modrinth.File
}

func (m *modrinthResult) Lock() *manifest.DependencyLock {
	lock := &manifest.DependencyLock{
		Name:        m.name,
		Version:     m.version.ID,
		VersionName: m.version.VersionNumber,
		Type:        "mod",
		URL:         m.file.URL,
		Provider:    "modrinth",
		Sha512:      m.file.Hashes.Sha512,
	}

	return lock
}

func (m *modrinthResult) Dependencies() []*manifest.InterpretedDependency {
	return nil
}

func NewModrinthProvider() *ModrinthProvider {
	return &ModrinthProvider{
		Client: modrinth.New(),
	}
}

func (m *ModrinthProvider) Name() string { return "modrinth" }

func (m *ModrinthProvider) Resolve(ctx context.Context, request *Request) (Result, error) {

	var wantedVersion *modrinth.Version

	var err error
	if request.Dependency.Version == "*" || request.Dependency.Version == "latest" {
		wantedVersion, err = m.resolveLatest(ctx, request)
	} else {
		wantedVersion, err = m.resolveById(ctx, request.Dependency.Version)
		if err != nil {
			return nil, err
		}
	}

	// this should never happen, we assume all versions have files
	if len(wantedVersion.Files) == 0 {
		return nil, ErrVersionHasNoFiles
	}

	return &modrinthResult{
		name:    request.Dependency.Name,
		version: wantedVersion,
		file:    fileFromVersion(wantedVersion),
	}, nil
}

func (m *ModrinthProvider) ResolveLatest(ctx context.Context, request *Request) (Result, error) {
	version, err := m.resolveLatest(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(version.Files) == 0 {
		return nil, ErrVersionHasNoFiles
	}

	return &modrinthResult{
		name:    request.Dependency.Name,
		version: version,
		file:    fileFromVersion(version),
	}, nil
}

func (m *ModrinthProvider) resolveById(ctx context.Context, id string) (*modrinth.Version, error) {
	switch len(id) {
	case 8:
		// we fetch by version id
		return m.Client.GetVersion(ctx, id)
	case 40:
		fallthrough
	case 128:
		// we fetch the version by hash
		return m.Client.GetVersionFile(ctx, id)
	default:
		return nil, ErrVersionNotSupported
	}
}

func (m *ModrinthProvider) resolveLatest(ctx context.Context, request *Request) (*modrinth.Version, error) {
	query := &modrinth.ListProjectVersionQuery{
		Loaders:      []string{request.Requirements.PlatformName()},
		GameVersions: []string{request.Requirements.MinecraftVersion()},
	}

	versions, err := m.Client.ListProjectVersion(ctx, request.Dependency.Name, query)
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, ErrVersionsNotFound
	}

	// grab the first version (not sure if this is good)
	return &versions[0], nil
}

func (m *ModrinthProvider) Fetch(ctx context.Context, toFetch Result) (io.Reader, int, error) {
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

func (m *ModrinthProvider) CanConvertURL(url string) bool {
	return strings.HasPrefix(url, "https://cdn.modrinth.com/data/")
}

func (m *ModrinthProvider) ConvertURL(ctx context.Context, url string) (string, error) {
	if !m.CanConvertURL(url) {
		return "", fmt.Errorf("url %s is not a modrinth CDN url", url)
	}

	// url looks like https://cdn.modrinth.com/data/iFnEtHsI/versions/1.4.6/alaskanativecraft-1.4.6.jar
	// we need to extract the id (iFnEtHsI) and the version (1.4.6)

	// id is easy cause it is always the same length
	id := url[30:38]

	// version starts at the same position, but we need to find the end
	version := strings.SplitN(url[48:], "/", 2)[0]

	// now we can fetch the version
	versions, err := m.Client.ListProjectVersion(ctx, id, nil)
	if err != nil {
		return "", err
	}

	// now we need to find the version we want
	var wantedVersion *modrinth.Version

	for _, v := range versions {
		if v.VersionNumber == version || v.ID == version {
			wantedVersion = &v
			break
		}
	}

	if wantedVersion == nil {
		return "", fmt.Errorf("version %s not found", version)
	}

	// get the project slug
	project, err := m.Client.GetProject(ctx, id)
	if err != nil {
		return "", err
	}

	// now we can return install pkgid
	return fmt.Sprintf("modrinth:%s@%s", project.Slug, wantedVersion.ID), nil
}

func fileFromVersion(version *modrinth.Version) *modrinth.File {
	if len(version.Files) == 0 {
		return nil
	}

	// grab primary version, fallback to latest
	resolvedFile := version.Files[0]
	for _, f := range version.Files {
		if f.Primary {
			resolvedFile = f
			break
		}
	}

	return &resolvedFile
}
