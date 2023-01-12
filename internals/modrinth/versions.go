package modrinth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/pkg/errors"
)

var (
	// Is returned when a version is not found
	ErrVersionNotFound = errors.Wrap(ErrResourceNotFound, "version not found")
)

// ListProjectVersionQuery is used to filter the results of ListProjectVersion
type ListProjectVersionQuery struct {
	// Loaders is a list of loaders to filter by (e.g. "fabric", "forge", "quilt")
	Loaders []string `json:"loaders,omitempty"`
	// GameVersions is a list of Minecraft versions to filter by (e.g. "1.16.5", "1.17.1")
	GameVersions []string `json:"game_versions,omitempty"`
	// Featured is a boolean to filter by versions that are marked as "featured".
	// You'll need to pass in a reference if you want to include this field in the api request.
	// You can use the [ListProjectVersionQuery.SetFeatured] method to easily do this.
	Featured *bool `json:"featured,omitempty"`
}

func (l *ListProjectVersionQuery) SetFeatured(featured bool) *ListProjectVersionQuery {
	l.Featured = &featured
	return l
}

func sliceAsJson(s []string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// String converts this to a query string
func (l *ListProjectVersionQuery) String() string {
	values := url.Values{}
	if l.Loaders != nil {
		values.Add("loaders", sliceAsJson(l.Loaders))
	}
	if l.GameVersions != nil {
		values.Add("game_versions", sliceAsJson(l.GameVersions))
	}
	if l.Featured != nil {
		values.Add("featured", fmt.Sprintf("%t", *l.Featured))
	}
	return values.Encode()
}

// ListProjectVersion returns a list of versions for a project.
// `query` can be used to pre filter the results. Pass nil to not filter.
func (c *Client) ListProjectVersion(ctx context.Context, idOrSlug string, query *ListProjectVersionQuery) ([]Version, error) {
	reqUrl := c.url("v2/project", idOrSlug, "version")
	if idOrSlug == "" {
		return nil, ErrInvalidProjectIDOrSlug
	}

	if query != nil {
		reqUrl.RawQuery = query.String()
	}

	res, err := c.get(ctx, reqUrl.String())

	if err != nil {
		return nil, err
	}

	var result []Version
	if err = decode(res, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetVersion returns a single version for a project given a 6 char id (`IIJJKKLL`)
func (c *Client) GetVersion(ctx context.Context, id string) (*Version, error) {
	reqUrl := c.url("v2/version", id)

	if id == "" {
		return nil, ErrInvalidVersionID
	}

	res, err := c.get(ctx, reqUrl.String())
	if err != nil {
		return nil, err
	}

	var result Version
	if err = decode(res, &result); err != nil {
		if err == ErrResourceNotFound {
			return nil, ErrVersionNotFound
		}
		return nil, err
	}

	return &result, nil
}

var (
	// ErrInvalidFileHash is returned when the hash is not a valid sha1 hash
	ErrInvalidFileHash = errors.New("invalid file hash")
)

// GetVersionFile returns the version containing the file with the given hash.
// `hash` can be a sha1 or sha512 hash
// Caution: returns a `Version`, not a `File`
func (c *Client) GetVersionFile(ctx context.Context, hash string) (*Version, error) {
	url := c.url("v2/version_file", hash)

	query := url.Query()
	switch len(hash) {
	case 40:
		query.Add("algorithm", "sha1")
	case 128:
		query.Add("algorithm", "sha512")
	default:
		return nil, ErrInvalidFileHash
	}

	url.RawQuery = query.Encode()

	res, err := c.get(ctx, url.String())

	if err != nil {
		return nil, err
	}

	var result Version
	if err = decode(res, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
