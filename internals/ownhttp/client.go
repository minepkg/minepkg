package ownhttp

import (
	"net/http"
)

type AddHeaderTransport struct {
	T http.RoundTripper
}

func (adt *AddHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("User-Agent", "minepkg (github.com/minepkg/minepkg)")
	return adt.T.RoundTrip(req)
}

func NewAddHeaderTransport(T http.RoundTripper) *AddHeaderTransport {
	if T == nil {
		T = http.DefaultTransport
	}
	return &AddHeaderTransport{T}
}

// New returns a new http.Client with the AddHeaderTransport (setting the User-Agent header)
func New() *http.Client {
	return &http.Client{Transport: NewAddHeaderTransport(nil)}
}
