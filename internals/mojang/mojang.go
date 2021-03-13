// Package mojang allows to login to a mojang account in order to start Minecraft
package mojang

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
)

var (
	// ErrorNotFound gets returned when a 404 occured
	ErrorNotFound = errors.New("resource not found")
	// ErrorBadRequest gets returned when a 400 occured
	ErrorBadRequest = errors.New("bad Request")
)

// Client contains credentials and methods to talk
// to the mojang api
type Client struct {
	// HTTP is the internal http client
	HTTP *http.Client
}

// New returns a new MojangAPI instance
func New() *Client {
	return &Client{
		HTTP: http.DefaultClient,
	}
}

// NewWithClient returns a new MojangAPI instance using a custom http client
// supplied as a first paramter
func NewWithClient(client *http.Client) *Client {
	return &Client{
		HTTP: client,
	}
}

// Login is a convinient method that uses username/password credentials
// to fetch a token from Mojangs Authentication Server. It then uses (only) that token
// to login to minepkg
func (m *Client) Login(username string, password string) (*AuthResponse, error) {
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

	authRes := AuthResponse{}
	if err := parseJSON(res, &authRes); err != nil {
		return nil, err
	}
	return &authRes, nil
}

// MojangEnsureToken checks if the token need to be refreshed and does so it required. it returns a valid token
func (m *Client) MojangEnsureToken(accessToken string, clientToken string) (*AuthResponse, error) {
	ok, _ := m.mojangCheckValid(accessToken, clientToken)
	if ok {
		return &AuthResponse{AccessToken: accessToken, ClientToken: clientToken}, nil
	}
	return m.MojangRefreshToken(accessToken, clientToken)
}

func (m *Client) mojangCheckValid(accessToken string, clientToken string) (bool, error) {
	loginData := struct {
		AccessToken string `json:"accessToken"`
		ClientToken string `json:"clientToken"`
	}{accessToken, clientToken}

	res, err := m.postJSON(context.TODO(), "https://authserver.mojang.com/validate", loginData)
	if err != nil {
		return false, err
	}

	if res.StatusCode != http.StatusNoContent {
		err := &mojangError{}
		return false, err
	}

	return true, nil
}

// MojangRefreshToken refreshed the given token
func (m *Client) MojangRefreshToken(accessToken string, clientToken string) (*AuthResponse, error) {
	loginData := struct {
		AccessToken string `json:"accessToken"`
		ClientToken string `json:"clientToken"`
	}{accessToken, clientToken}

	res, err := m.postJSON(context.TODO(), "https://authserver.mojang.com/refresh", loginData)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		minepkgErr := &mojangError{}
		if err := parseJSON(res, minepkgErr); err != nil {
			return nil, errors.New("mojang API did response with unexpected status " + res.Status)
		}
		return nil, minepkgErr
	}

	auth := AuthResponse{}
	if err := parseJSON(res, &auth); err != nil {
		return nil, err
	}

	return &auth, nil
}

// postJSON posts json
func (m *Client) postJSON(ctx context.Context, url string, data interface{}) (*http.Response, error) {
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
func (m *Client) decorate(req *http.Request) {
	req.Header.Set("User-Agent", "minepkg (https://github.com/minepkg/minepkg)")
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
}

// DecorateRequest decorates a provided http request with the User-Agent header and a auth header if set
func (m *Client) DecorateRequest(req *http.Request) {
	m.decorate(req)
}

func parseJSON(res *http.Response, i interface{}) error {
	b, _ := ioutil.ReadAll(res.Body)
	return json.Unmarshal(b, i)
}
