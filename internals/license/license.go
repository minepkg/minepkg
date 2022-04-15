package license

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// GithubLicense is what https://api.github.com/licenses/<name> returns
type GithubLicense struct {
	Key            string   `json:"key"`
	Name           string   `json:"name"`
	SpdxID         string   `json:"spdx_id"`
	URL            string   `json:"url"`
	NodeID         string   `json:"node_id"`
	HTMLURL        string   `json:"html_url"`
	Description    string   `json:"description"`
	Implementation string   `json:"implementation"`
	Permissions    []string `json:"permissions"`
	Conditions     []string `json:"conditions"`
	Limitations    []string `json:"limitations"`
	Body           string   `json:"body"`
	Featured       bool     `json:"featured"`
}

// GetLincense fetches the given license from githubs api
func GetLicense(licenseName string) (*GithubLicense, error) {
	var license GithubLicense
	r, err := http.Get("https://api.github.com/licenses/" + licenseName)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		return nil, fmt.Errorf("status code was not 200")
	}

	if err := json.NewDecoder(r.Body).Decode(&license); err != nil {
		return nil, err
	}

	return &license, nil
}
