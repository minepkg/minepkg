package modrinth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type ListProjectVersionQuery struct {
	Loaders      []string `json:"loaders,omitempty"`
	GameVersions []string `json:"game_versions,omitempty"`
	Featured     *bool    `json:"featured,omitempty"`
}

func (l *ListProjectVersionQuery) queryString() string {
	values := url.Values{}
	if l.Loaders != nil {
		values.Add("loaders", strings.Join(l.Loaders, ","))
	}
	if l.GameVersions != nil {
		values.Add("game_versions", strings.Join(l.GameVersions, ","))
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
		reqUrl.RawQuery = query.queryString()
	}

	res, err := c.get(ctx, reqUrl.String())

	if err != nil {
		return nil, err
	}

	var result []Version
	if err = c.decode(res, &result); err != nil {
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

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	var result Version
	if err = c.decode(res, &result); err != nil {
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
	if err = c.decode(res, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
