package ownhttp

import (
	"net/http"

	"golang.org/x/time/rate"
)

type ThrottleTransport struct {
	T       http.RoundTripper
	limiter *rate.Limiter
}

func (tt *ThrottleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	err := tt.limiter.Wait(req.Context())
	if err != nil {
		return nil, err
	}

	return tt.T.RoundTrip(req)
}

func NewThrottleTransport(T http.RoundTripper, limiter *rate.Limiter) *ThrottleTransport {
	if T == nil {
		T = http.DefaultTransport
	}
	return &ThrottleTransport{T, limiter}
}
