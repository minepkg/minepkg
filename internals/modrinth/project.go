package modrinth

import (
	"context"

	"github.com/pkg/errors"
)

var (
	ErrProjectNotFound = errors.Wrap(ErrResourceNotFound, "project not found")
)

// GetProject returns the project with the given ID
func (c *Client) GetProject(ctx context.Context, id string) (*Project, error) {
	reqUrl := c.url("v2/project", id)
	res, err := c.get(ctx, reqUrl.String())
	if err != nil {
		return nil, err
	}

	var project Project
	if err = decode(res, &project); err != nil {
		if err == ErrResourceNotFound {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	return &project, nil
}
