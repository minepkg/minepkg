package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Masterminds/semver/v3"
)

// ErrNoMatchingRelease is returned if a wanted package query could not be resolved
type ErrNoQueryResult struct {
	// Query is the query that did not resolve
	Query *ReleasesQuery
	// Err is the underlying error that describes why there was no package found
	Err error
}

func (e *ErrNoQueryResult) Error() string {
	if e.Query == nil {
		return e.Error()
	}
	minecraft := e.Query.Minecraft
	if e.Query.Minecraft == "" {
		minecraft = "*"
	}
	return fmt.Sprintf(
		"query for name: %s@%s (platform: %s, minecraft: %s) failed: %s",
		e.Query.Name,
		e.Query.VersionRange,
		e.Query.Platform,
		minecraft,
		e.Err.Error(),
	)
}

// ReleasesQuery is a query to find a release
type ReleasesQuery struct {
	// Platform can bei either fabric or forge
	Platform string
	// Name should be the name of the wanted package
	Name string
	// Minecraft version for the project. This has to be either '' or a valid version number.
	// a semver string is NOT allowed here
	Minecraft string
	// VersionRange can be any semver string specifying the desired package version
	VersionRange string
}

// FindRelease gets the latest release matching the passed requirements via `RequirementQuery`
func (m *MinepkgAPI) ReleasesQuery(ctx context.Context, query *ReleasesQuery) (*Release, error) {
	if query.Minecraft == "*" || query.Minecraft == "latest" {
		query.Minecraft = ""
	}
	if query.Minecraft != "" {
		var err error
		if _, err = semver.NewVersion(query.Minecraft); err != nil {
			return nil, ErrInvalidMinecraftRequirement
		}
	}

	urlQuery := url.Values{}
	urlQuery.Add("platform", query.Platform)
	urlQuery.Add("name", query.Name)
	if query.Minecraft != "" {
		urlQuery.Add("minecraft", query.Minecraft)
	}
	urlQuery.Add("versionRange", query.VersionRange)
	res, err := m.get(ctx, m.APIUrl+"/releases/_query?"+urlQuery.Encode())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// special handle for 404 errors
	if res.StatusCode == http.StatusNotFound {
		minepkgErr := &MinepkgError{}
		if err := parseJSON(res, minepkgErr); err != nil {
			return nil, errors.New("minepkg API did respond with expected error format. code: " + res.Status)
		}

		resolveErr := &ErrNoQueryResult{query, nil}
		switch minepkgErr.ResolveError {
		case "minecraft_req_not_satisfiable":
			resolveErr.Err = ErrNoReleaseForMinecraftVersion
		case "version_req_not_satisfiable":
			resolveErr.Err = ErrNoReleaseForVersion
		case "project_does_not_exist":
			resolveErr.Err = ErrProjectDoesNotExist
		case "no_releases_for_platform":
			resolveErr.Err = ErrNoReleasesForPlatform
		case "all_reqs_not_satisfiable":
			resolveErr.Err = ErrNoReleaseWithConstrains
		default:
			// unknown error
			return nil, err
		}
		// return enriched resolve error
		return nil, resolveErr
	}

	// all other errors
	if err := checkResponse(res); err != nil {
		return nil, err
	}

	var release Release
	if err := parseJSON(res, &release); err != nil {
		return nil, err
	}

	release.decorate(m) // sets the private client field

	return &release, nil
}
