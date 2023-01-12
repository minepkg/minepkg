// Package api is a client for the minepkg API https://api.minepkg/docs
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

var (
	// ErrNotFound gets returned when a 404 occurred
	ErrNotFound = errors.New("resource not found")
	// ErrBadRequest gets returned when a 400 occurred
	ErrBadRequest = errors.New("bad Request")
	// DefaultURL is "https://api.preview.minepkg.io/v1"
	DefaultURL = "https://api.preview.minepkg.io/v1"
)

// MinepkgClient contains credentials and methods to talk
// to the minepkg api
type MinepkgClient struct {
	// HTTP is the internal http client
	HTTP *http.Client
	// BaseAPI is the API url used. defaults to `https://api.preview.minepkg.io/v1`
	APIUrl string
	APIKey string
	JWT    string
	User   *User
}

// New returns a new MinepkgAPI client
func New() *MinepkgClient {
	return &MinepkgClient{
		HTTP:   http.DefaultClient,
		APIUrl: DefaultURL,
	}
}

// NewWithCustomHTTP returns a new MinepkgAPI client using a custom http client
// supplied as a first parameter
func NewWithCustomHTTP(client *http.Client) *MinepkgClient {
	return &MinepkgClient{
		HTTP:   client,
		APIUrl: DefaultURL,
	}
}

// HasCredentials returns true if a jwt or api is set
func (m *MinepkgClient) HasCredentials() bool {
	return m.JWT != "" || m.APIKey != ""
}

// GetAccount gets the account information of the current user
func (m *MinepkgClient) GetAccount(ctx context.Context) (*User, error) {
	res, err := m.get(ctx, m.APIUrl+"/account")
	if err != nil {
		return nil, err
	}

	user := User{}
	if err := decode(res, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// PutRelease uploads a new release
func (m *MinepkgClient) PutRelease(project string, version string, reader io.Reader) (*Release, error) {
	// prepare request
	req, err := http.NewRequest("PUT", m.APIUrl+"/projects/"+project+"@"+version, reader)
	if err != nil {
		return nil, err
	}
	m.decorate(req)
	req.Header.Set("Content-Type", "application/java-archive")

	// execute request and handle response
	res, err := m.HTTP.Do(req)
	if err != nil {
		return nil, err
	}

	// parse body
	release := Release{}
	if err := decode(res, &release); err != nil {
		return nil, err
	}

	return &release, nil
}

// PostCrashReport posts a new crash report
func (m *MinepkgClient) PostCrashReport(ctx context.Context, report *CrashReport) error {
	res, err := m.postJSON(context.TODO(), m.APIUrl+"/crash-reports", report)
	if err != nil {
		return err
	}
	if err := checkResponse(res); err != nil {
		return err
	}

	return err
}

// PostProjectMedia uploads a new image to a project
func (m *MinepkgClient) PostProjectMedia(ctx context.Context, project string, content io.Reader) error {

	req, err := http.NewRequest("POST", m.APIUrl+"/projects/"+project+"/media", content)
	// TODO: does not have to be png.. ?
	req.Header.Add("Content-Type", "image/png")
	req = req.WithContext(ctx)
	if err != nil {
		return err
	}

	m.decorate(req)
	res, err := m.HTTP.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	b, _ := ioutil.ReadAll(res.Body)
	fmt.Println(string(b))
	if err := checkResponse(res); err != nil {
		return err
	}

	return err
}

// get is a helper that does a GET request and also sets various things
func (m *MinepkgClient) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	req = req.WithContext(ctx)
	if err != nil {
		return nil, err
	}
	m.decorate(req)
	return m.HTTP.Do(req)
}

// delete is a helper that does a DELETE request and also sets various things
func (m *MinepkgClient) delete(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return nil, err
	}
	m.decorate(req)
	return m.HTTP.Do(req)
}

// postJSON posts json
func (m *MinepkgClient) postJSON(ctx context.Context, url string, data interface{}) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req = req.WithContext(ctx)
	if err != nil {
		return nil, err
	}

	m.decorate(req)
	return m.HTTP.Do(req)
}

// decorate decorates a request with the User-Agent header and a auth header if set
func (m *MinepkgClient) decorate(req *http.Request) {
	req.Header.Set("User-Agent", "minepkg (https://github.com/minepkg/minepkg)")
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	switch {
	case m.JWT != "":
		req.Header.Set("Authorization", "Bearer "+m.JWT)
	case m.APIKey != "":
		req.Header.Set("Authorization", "api-key "+m.APIKey)
	}
}

// DecorateRequest decorates a provided http request with the User-Agent header and a auth header if set
func (m *MinepkgClient) DecorateRequest(req *http.Request) {
	m.decorate(req)
}

// decode is a helper that decodes json, and checks the status code
func decode(res *http.Response, v interface{}) error {
	if err := checkResponse(res); err != nil {
		return err
	}

	if err := json.NewDecoder(res.Body).Decode(v); err != nil {
		return err
	}

	return nil
}

func checkResponse(res *http.Response) error {
	switch {
	case res.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case res.StatusCode >= 200 && res.StatusCode < 400:
		return nil
	case res.StatusCode >= 400:
		minepkgErr := &MinepkgError{}
		if err := json.NewDecoder(res.Body).Decode(minepkgErr); err != nil {
			return errors.New("minepkg API did respond with unexpected error format. code: " + res.Status)
		}
		return minepkgErr
	default:
		return errors.New("minepkg API did respond with unexpected status code " + res.Status)
	}
}
