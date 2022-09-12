package modrinth

import (
	"context"
	"fmt"
)

func (c *Client) GetProject(ctx context.Context, id string) (*Project, error) {
	reqUrl := c.url("v2/project", id)
	res, err := c.get(ctx, reqUrl.String())
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		if res.StatusCode == 404 {
			return nil, ErrVersionNotFound
		}
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	var project Project
	if err = c.decode(res, &project); err != nil {
		return nil, err
	}

	return &project, nil
}
