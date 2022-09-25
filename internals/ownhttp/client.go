package ownhttp

import (
	"net/http"
)

// New returns a new http.Client with the AddHeaderTransport (setting the User-Agent header)
func New() *http.Client {
	return &http.Client{Transport: NewAddHeaderTransport(nil)}
}
