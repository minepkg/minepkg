package providers

import (
	"context"
	"errors"
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
		Name:     m.name,
		Version:  m.version.VersionNumber,
		Type:     "mod",
		URL:      m.file.URL,
		Provider: "modrinth",
		Sha512:   m.file.Hashes.Sha512,
	}

	return lock
}

func (m *modrinthResult) Dependencies() []*manifest.InterpretedDependency {
	return nil
}

func (m *ModrinthProvider) Resolve(ctx context.Context, request *Request) (Result, error) {

	var wantedVersion *modrinth.Version

	sourceParts := strings.SplitN(request.Dependency.Source, "@", 2)
	if len(sourceParts) == 2 && (sourceParts[1] != "*" && sourceParts[1] != "latest") {
		switch len(sourceParts[1]) {
		case 8:
			// we fetch by version id
			version, err := m.Client.GetVersion(ctx, sourceParts[1])
			if err != nil {
				return nil, err
			}
			wantedVersion = version
		case 40:
			fallthrough
		case 128:
			// we fetch the version by hash
			version, err := m.Client.GetVersionFile(ctx, sourceParts[1])
			if err != nil {
				return nil, err
			}
			wantedVersion = version
		default:
			return nil, ErrVersionNotSupported
		}
	} else {
		// no version specified, use latest
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
		wantedVersion = &versions[0]
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
