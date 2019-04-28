package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	baseAPI         = getAPIUrl()
)

func getAPIUrl() string {
	overwrite := os.Getenv("MINEPKG_API")
	if overwrite != "" {
		return overwrite
	}

	return "https://test-api.minepkg.io/v1"
}

// MinepkgAPI contains credentials and methods to talk
// to the minepkg api
type MinepkgAPI struct {
	http   *http.Client
	APIKey string
	JWT    string
	User   *User
}

// New returns a new MinepkgAPI instance
func New() *MinepkgAPI {
	return &MinepkgAPI{
		http: http.DefaultClient,
	}
}

// NewWithClient returns a new MinepkgAPI instance using a custom http client
// supplied as a first paramter
func NewWithClient(client *http.Client) *MinepkgAPI {
	return &MinepkgAPI{
		http: client,
	}
}

// Login is a convinient method that uses username/password credentials
// to fetche a token from Mojangs Authentication Server. It then uses (only) that token
// to login to minepkg
func (m *MinepkgAPI) Login(username string, password string) (*LoginData, error) {
	payload := mojangLogin{
		Agent:       mojangAgent{Name: "Minecraft", Version: 1},
		Username:    username,
		Password:    password,
		RequestUser: true,
	}
	//data, _ := json.Marshal(payload)
	res, err := m.postJSON(context.TODO(), "https://authserver.mojang.com/authenticate", payload)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		mojangErr := mojangError{}
		if err := parseJSON(res, &mojangErr); err != nil {
			return nil, errors.New("Mojang API did response with unexpected status " + res.Status)
		}
		return nil, mojangErr
	}

	authRes := mojangAuthResponse{}
	if err := parseJSON(res, &authRes); err != nil {
		return nil, err
	}
	// TODO: check all the stuff
	minepkgLogin, err := m.MojangTokenLogin(authRes.AccessToken, authRes.ClientToken)
	if err != nil {
		return nil, err
	}

	return &LoginData{
		Minepkg: minepkgLogin,
		Mojang:  &authRes,
	}, nil
}

// MojangLogin signs in using Mojang credentials
// Prefer to use `MojangTokenLogin` wherever possible. We never store
// your password and all data in transit is encrypted, but you don't have
// to rely on that promise by using `MojangTokenLogin`!
func (m *MinepkgAPI) MojangLogin(username string, password string) (*AuthResponse, error) {
	data := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{username, password}
	res, err := m.postJSON(context.TODO(), baseAPI+"/account/_mojangLogin", data)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("minepkg API did not respond with 200")
	}

	auth := AuthResponse{}
	if err := parseJSON(res, &auth); err != nil {
		return nil, err
	}

	m.JWT = auth.Token
	m.User = auth.User

	return &auth, nil
}

// MojangTokenLogin signs in using Mojang a provided mojang `accessToken`
// and `clientToken` from the Mojang Authentication server. (docs: https://wiki.vg/Authentication)
// This way our servers never see your password.
func (m *MinepkgAPI) MojangTokenLogin(accessToken string, clientToken string) (*AuthResponse, error) {
	loginData := struct {
		AccessToken string `json:"accessToken"`
		ClientToken string `json:"clientToken"`
	}{accessToken, clientToken}
	res, err := m.postJSON(context.TODO(), baseAPI+"/account/_mojangTokenLogin", loginData)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		minepkgErr := &MinepkgError{}
		if err := parseJSON(res, minepkgErr); err != nil {
			return nil, errors.New("minepkg API did response with unexpected status " + res.Status)
		}
		return nil, minepkgErr
	}

	auth := AuthResponse{}
	if err := parseJSON(res, &auth); err != nil {
		return nil, err
	}

	m.JWT = auth.Token
	m.User = auth.User

	return &auth, nil
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

// GetForgeVersions gets a single release from a project
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
	res, err := m.http.Do(req)
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

// get is a helper that does a get request and also sets various things
func (m *MinepkgAPI) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	req = req.WithContext(ctx)
	if err != nil {
		return nil, err
	}
	m.decorate(req)
	return m.http.Do(req)
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
	return m.http.Do(req)
}

func (m *MinepkgAPI) post(ctx context.Context, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	req = req.WithContext(ctx)
	if err != nil {
		return nil, err
	}

	m.decorate(req)
	return m.http.Do(req)
}

func (m *MinepkgAPI) put(ctx context.Context, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("PUT", url, body)
	req = req.WithContext(ctx)
	if err != nil {
		return nil, err
	}

	m.decorate(req)
	return m.http.Do(req)
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
	req.Header.Set("Content-Type", "application/json")
	switch {
	case m.JWT != "":
		req.Header.Set("Authorization", "Bearer "+m.JWT)
	case m.APIKey != "":
		req.Header.Set("API-KEY", m.APIKey)
	}
}

func parseJSON(res *http.Response, i interface{}) error {
	b, _ := ioutil.ReadAll(res.Body)
	return json.Unmarshal(b, i)
}
