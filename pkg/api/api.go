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
	"os"
)

var (
	// ErrorNotFound gets returned when a 404 occured
	ErrorNotFound = errors.New("Resource not found")
	// ErrorBadRequest gets returned when a 400 occured
	ErrorBadRequest = errors.New("Bad Request")
	baseAPI         = GetAPIUrl()
)

// GetAPIUrl retrurns the minepkg api url
// can be overwritten by the env variable `MINEPKG_API`
func GetAPIUrl() string {
	overwrite := os.Getenv("MINEPKG_API")
	if overwrite != "" {
		return overwrite
	}

	return "https://test-api.minepkg.io/v1"
}

// MinepkgAPI contains credentials and methods to talk
// to the minepkg api
type MinepkgAPI struct {
	// HTTP is the internal http client
	HTTP   *http.Client
	APIKey string
	JWT    string
	User   *User
}

// New returns a new MinepkgAPI instance
func New() *MinepkgAPI {
	return &MinepkgAPI{
		HTTP: http.DefaultClient,
	}
}

// NewWithClient returns a new MinepkgAPI instance using a custom http client
// supplied as a first paramter
func NewWithClient(client *http.Client) *MinepkgAPI {
	return &MinepkgAPI{
		HTTP: client,
	}
}

// GetAccount gets the account information
func (m *MinepkgAPI) GetAccount(ctx context.Context) (*User, error) {
	res, err := m.get(ctx, baseAPI+"/account")
	if err != nil {
		return nil, err
	}
	if err := checkResponse(res); err != nil {
		return nil, err
	}

	user := User{}
	if err := parseJSON(res, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetForgeVersions gets all available forge versions
// This currently does not work!
func (m *MinepkgAPI) GetForgeVersions(ctx context.Context) (*ForgeVersionResponse, error) {
	res, err := m.get(ctx, baseAPI+"/meta/forge-versions")
	if err != nil {
		return nil, err
	}
	if err := checkResponse(res); err != nil {
		return nil, err
	}

	fRes := ForgeVersionResponse{}
	if err := parseJSON(res, &fRes); err != nil {
		return nil, err
	}

	return &fRes, nil
}

// PutRelease uploads a new release
func (m *MinepkgAPI) PutRelease(project string, version string, reader io.Reader) (*Release, error) {
	// prepare request
	req, err := http.NewRequest("PUT", baseAPI+"/projects/"+project+"@"+version, reader)
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
	if err := checkResponse(res); err != nil {
		return nil, err
	}

	// parse body
	release := Release{}
	if err := parseJSON(res, &release); err != nil {
		return nil, err
	}

	return &release, nil
}

// PostCrashReport posts a new crash report
func (m *MinepkgAPI) PostCrashReport(ctx context.Context, report *CrashReport) error {
	res, err := m.postJSON(context.TODO(), baseAPI+"/crash-reports", report)
	if err != nil {
		return err
	}
	if err := checkResponse(res); err != nil {
		return err
	}

	return err
}

// PostProjectMedia uploads a new image to a project
func (m *MinepkgAPI) PostProjectMedia(ctx context.Context, project string, content io.Reader) error {

	req, err := http.NewRequest("POST", baseAPI+"/projects/"+project+"/media", content)
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

// get is a helper that does a get request and also sets various things
func (m *MinepkgAPI) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	req = req.WithContext(ctx)
	if err != nil {
		return nil, err
	}
	m.decorate(req)
	return m.HTTP.Do(req)
}

// postJSON posts json
func (m *MinepkgAPI) postJSON(ctx context.Context, url string, data interface{}) (*http.Response, error) {
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

func (m *MinepkgAPI) post(ctx context.Context, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	req = req.WithContext(ctx)
	if err != nil {
		return nil, err
	}

	m.decorate(req)
	return m.HTTP.Do(req)
}

func (m *MinepkgAPI) put(ctx context.Context, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("PUT", url, body)
	req = req.WithContext(ctx)
	if err != nil {
		return nil, err
	}

	m.decorate(req)
	return m.HTTP.Do(req)
}

func checkResponse(res *http.Response) error {
	switch {
	case res.StatusCode == http.StatusNotFound:
		return ErrorNotFound
	case res.StatusCode == http.StatusBadRequest:
		return ErrorBadRequest
	case res.StatusCode >= 400:
		minepkgErr := &MinepkgError{}
		if err := parseJSON(res, minepkgErr); err != nil {
			return errors.New("minepkg API did respond with unexpected status " + res.Status)
		}
		return minepkgErr
	}

	return nil
}

// decorate decorates a request with the User-Agent header and a auth header if set
func (m *MinepkgAPI) decorate(req *http.Request) {
	req.Header.Set("User-Agent", "minepkg (https://github.com/fiws/minepkg)")
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	switch {
	case m.JWT != "":
		req.Header.Set("Authorization", "Bearer "+m.JWT)
	case m.APIKey != "":
		req.Header.Set("API-KEY", m.APIKey)
	}
}

// DecorateRequest decorates a provided http request with the User-Agent header and a auth header if set
func (m *MinepkgAPI) DecorateRequest(req *http.Request) {
	m.decorate(req)
}

func parseJSON(res *http.Response, i interface{}) error {
	b, _ := ioutil.ReadAll(res.Body)
	return json.Unmarshal(b, i)
}
