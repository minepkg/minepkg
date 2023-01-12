package modrinth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

const DefaultApiURL = "https://api.modrinth.com/"

var (
	// An error that is returned if the provided project ID or slug is invalid
	// currently this is only returned if it was an empty string
	ErrInvalidProjectIDOrSlug = errors.New("invalid project ID or slug")
	// An error that is returned if the provided version ID is invalid
	// currently this is only returned if it was an empty string
	ErrInvalidVersionID = errors.New("invalid version ID")
	// A generic error that is returned if a resource was not found
	// Some methods return more specific errors that wrap this error (e.g. ErrProjectNotFound)
	ErrResourceNotFound = errors.New("resource not found")
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
	joined, err := url.JoinPath(c.baseURL.String(), addedPath...)
	if err != nil {
		panic(err)
	}

	url, err := url.Parse(joined)
	if err != nil {
		panic(err)
	}

	return url
}

// get is just a wrapper around http.Get() with context support
func (c *Client) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	return c.http.Do(req)
}

// decode is a helper that decodes json, and checks the status code
func decode(res *http.Response, v interface{}) error {
	if res.StatusCode != 200 {
		switch res.StatusCode {
		case 404:
			return ErrResourceNotFound
		default:
			return fmt.Errorf("unexpected status code: %d", res.StatusCode)
		}
	}

	if err := json.NewDecoder(res.Body).Decode(v); err != nil {
		return err
	}

	return nil
}
