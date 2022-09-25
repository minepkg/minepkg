package modrinth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
)

const DefaultApiURL = "https://api.modrinth.com/"

var (
	ErrInvalidProjectIDOrSlug = errors.New("invalid project ID or slug")
	ErrInvalidVersionID       = errors.New("invalid version ID")
)

type Client struct {
	http    *http.Client
	baseURL *url.URL
}

func New(httpClient *http.Client) *Client {
	parsedDefaultURL, _ := url.Parse(DefaultApiURL)

	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		http:    httpClient,
		baseURL: parsedDefaultURL,
	}
}

// url joins the addedPath to the baseURL (panics if new path can not be parsed)
func (c *Client) url(addedPath ...string) *url.URL {
	// TODO: use url.JoinPath() when it's available
	u, err := url.Parse(path.Join(addedPath...))
	if err != nil {
		panic(err)
	}

	joined := c.baseURL.ResolveReference(u)
	return joined
}

// get is just a wrapper around http.Get() with context support
func (c *Client) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	return c.http.Do(req)
}

// decode is a helper that decodes json, and checks the status code
func (c *Client) decode(res *http.Response, v interface{}) error {
	if res.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	if err := json.NewDecoder(res.Body).Decode(v); err != nil {
		return err
	}

	return nil
}
