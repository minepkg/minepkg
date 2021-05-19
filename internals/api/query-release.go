package api

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Masterminds/semver/v3"
)

// ReleasesQuery is a query to find a release
type ReleasesQuery struct {
	// Platform can bei either fabric or forge
	Platform string
	// Name should be the name of the wanted package
	Name string
	// Minecraft version for the project. This has to be either '*' or a valid version number.
	// any semver string is NOT allowed here
	Minecraft string
	// Version to return. this can be any semver string
	Version string
}

// FindRelease gets the latest release matching the passed requirements via `RequirementQuery`
func (m *MinepkgAPI) ReleasesQuery(ctx context.Context, query *ReleasesQuery) (*Release, error) {
	if query.Minecraft != "*" {
		var err error
		if _, err = semver.NewVersion(query.Minecraft); err != nil {
			return nil, ErrInvalidMinecraftRequirement
		}
	}

	urlQuery := url.Values{}
	urlQuery.Add("platform", query.Platform)
	urlQuery.Add("name", query.Name)
	urlQuery.Add("minecraft", query.Minecraft)
	urlQuery.Add("versionRange", query.Version)
	res, err := m.get(ctx, m.APIUrl+"/releases/_query?"+urlQuery.Encode())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// TODO: return more specific errors here
	if err := checkResponse(res); err != nil {
		if err == ErrNotFound {
			e, ok := err.(MinepkgError)
			if !ok {
				return nil, err
			}
			fmt.Println("BULLSHIT")
			fmt.Println(e)
			fmt.Printf("%+v", query)
			switch e.ResolveError {
			case "minecraft_req_not_satisfiable":
				return nil, ErrNoReleaseForMinecraftVersion
			case "version_req_not_satisfiable":
				return nil, ErrNoReleaseForVersion
			case "project_does_not_exist":
				return nil, ErrProjectDoesNotExist
			case "no_releases_for_platform":
				return nil, ErrNoReleasesForPlatform
			case "all_reqs_not_satisfiable":
				return nil, ErrNoReleaseWithConstrains
			default:
				return nil, ErrNoReleaseWithConstrains
			}
		}
		return nil, err
	}

	var release Release
	if err := parseJSON(res, &release); err != nil {
		return nil, err
	}

	release.decorate(m) // sets the private client field

	return &release, nil
}
