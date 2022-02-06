package minecraft

import (
	"fmt"
	"net/http"
)

type APIClient struct {
	*http.Client
}

type APIErrorResponse struct {
	Path      string `json:"path"`
	ErrorType string `json:"errorType"`
	// ErrorCode is a string like "NOT_FOUND". The underlying json field name is "error"
	ErrorCode        string `json:"error"`
	ErrorMessage     string `json:"errorMessage"`
	DeveloperMessage string `json:"developerMessage"`
}

func (a *APIErrorResponse) Error() string {
	return fmt.Sprintf("%s: %s", a.ErrorType, a.ErrorMessage)
}

type GetProfileResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Skins []struct {
		ID      string `json:"id"`
		State   string `json:"state"`
		URL     string `json:"url"`
		Variant string `json:"variant"`
		Alias   string `json:"alias"`
	} `json:"skins"`
	Capes []interface{} `json:"capes"`
}

type LaunchAuthData interface {
	GetAccessToken() string
	GetPlayerName() string
	GetUUID() string
}

func New(httpClient *http.Client) *APIClient {
	return &APIClient{
		Client: httpClient,
	}
}
