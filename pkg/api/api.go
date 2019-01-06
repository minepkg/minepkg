package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
)

const baseAPI = "https://test-api.minepkg.io/v1"

var (
	// ErrorNotFound gets returned when a 404 occured
	ErrorNotFound = errors.New("Resource not found")
	// ErrorBadRequest gets returned when a 400 occured
	ErrorBadRequest = errors.New("Bad Request")
)

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
func (m *MinepkgAPI) Login(username string, password string) (*AuthResponse, error) {
	payload := mojangLogin{
		Agent:       mojangAgent{Name: "Minecraft", Version: 1},
		Username:    username,
		Password:    password,
		RequestUser: true,
	}
	//data, _ := json.Marshal(payload)
	res, err := m.postJSON("https://authserver.mojang.com/authenticate", payload)
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

	return m.MojangTokenLogin(authRes.AccessToken, authRes.ClientToken)
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
	res, err := m.postJSON(baseAPI+"/account/_mojangLogin", data)
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
	res, err := m.postJSON(baseAPI+"/account/_mojangTokenLogin", loginData)
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

// GetProject gets a single project
func (m *MinepkgAPI) GetProject(name string) (*Project, error) {
	res, err := m.get(baseAPI + "/projects/" + name)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(res); err != nil {
		return nil, err
	}

	project := Project{}
	if err := parseJSON(res, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

// GetRelease gets a single release from a project
func (m *MinepkgAPI) GetRelease(project string, version string) (*Release, error) {
	res, err := m.get(baseAPI + "/projects/" + project + "@" + version)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(res); err != nil {
		return nil, err
	}

	release := Release{}
	if err := parseJSON(res, &release); err != nil {
		return nil, err
	}

	return &release, nil
}

// GetReleaseList gets a all available releases for a project
func (m *MinepkgAPI) GetReleaseList(project string) ([]*Release, error) {
	res, err := m.get(baseAPI + "/projects/" + project + "/releases")
	if err != nil {
		return nil, err
	}
	if err := checkResponse(res); err != nil {
		return nil, err
	}

	releases := make([]*Release, 0, 20)
	if err := parseJSON(res, &releases); err != nil {
		return nil, err
	}

	return releases, nil
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
func (m *MinepkgAPI) get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	m.decorate(req)
	return m.http.Do(req)
}

// postJSON posts json
func (m *MinepkgAPI) postJSON(url string, data interface{}) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	m.decorate(req)
	return m.http.Do(req)
}

func (m *MinepkgAPI) post(url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	m.decorate(req)
	return m.http.Do(req)
}

func (m *MinepkgAPI) put(url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("PUT", url, body)
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
			return errors.New("minepkg API did response with unexpected status " + res.Status)
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
