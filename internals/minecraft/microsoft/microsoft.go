package microsoft

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

const (
	XBL_AUTHENTICATE   = "https://user.auth.xboxlive.com/user/authenticate"
	XBL_XSTS_AUTHORIZE = "https://xsts.auth.xboxlive.com/xsts/authorize"
	MC_API_XBOX_LOGIN  = "https://api.minecraftservices.com/authentication/login_with_xbox"
	MC_API_PROFILE     = "https://api.minecraftservices.com/minecraft/profile"
)

type MicrosoftClient struct {
	*http.Client
	// xblClient is a separate client because we need to set the token
	// and the horrifying Renegotiation option (see `New`)
	xblClient *http.Client
	Config    *oauth2.Config
	Token     *oauth2.Token
}

type Credentials struct {
	MicrosoftAuth    oauth2.Token
	MinecraftAuth    *XboxLoginResponse
	MinecraftProfile *GetProfileResponse
	ExpiresAt        time.Time
}

func (x *Credentials) GetAccessToken() string { return x.MinecraftAuth.AccessToken }
func (x *Credentials) GetPlayerName() string  { return x.MinecraftProfile.Name }
func (x *Credentials) GetUUID() string        { return x.MinecraftProfile.ID }

func (x *Credentials) IsExpired() bool {
	// add a minute current time for clock skew and stuff
	return x.ExpiresAt.Before(time.Now().Add(time.Minute))
}

func New(httpClient *http.Client, config *oauth2.Config) *MicrosoftClient {
	// shallow copy the http client so we don't modify the original
	lessSecureClient := *httpClient
	// we need this cause MS API
	// https://stackoverflow.com/questions/57420833/tls-no-renegotiation-error-on-http-request
	lessSecureClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{Renegotiation: tls.RenegotiateOnceAsClient},
	}

	// set some default config values
	if config.Scopes == nil {
		config.Scopes = []string{"XboxLive.signin", "offline_access"}
	}
	if config.Endpoint.AuthURL == "" {
		config.Endpoint = microsoft.AzureADEndpoint("consumers")
	}

	return &MicrosoftClient{
		Client:    httpClient,
		xblClient: &lessSecureClient,
		Config:    config,
		Token:     nil,
	}
}

func (m *MicrosoftClient) GetMinecraftCredentials(ctx context.Context) (*Credentials, error) {
	if m.Token == nil {
		return nil, fmt.Errorf("no token set")
	}

	// refresh token if needed
	newToken, err := m.Config.TokenSource(ctx, m.Token).Token()
	if err != nil {
		return nil, err
	}
	m.Token = newToken

	// 1. Authenticate with XBL
	xblAuth, err := m.xblAuth(ctx, m.Token.AccessToken)
	if err != nil {
		return nil, err
	}
	// 2. Get XSTS token
	xstsAuth, err := m.xstsAuth(ctx, xblAuth.Token)
	if err != nil {
		return nil, err
	}

	xstsToken := xstsAuth.Token
	if len(xstsAuth.DisplayClaims.Xui) == 0 {
		return nil, fmt.Errorf("XBL auth failed: no XUI claim")
	}
	userHash := xstsAuth.DisplayClaims.Xui[0].Uhs

	// 3. Get Minecraft token
	minecraftAuth, err := m.minecraftLoginWithXbox(ctx, userHash, xstsToken)
	if err != nil {
		return nil, err
	}

	// 4. Get Minecraft profile
	profile, err := m.getProfile(ctx, minecraftAuth.AccessToken)
	if err != nil {
		return nil, err
	}

	creds := &Credentials{
		MicrosoftAuth:    *m.Token,
		MinecraftAuth:    minecraftAuth,
		MinecraftProfile: profile,
		ExpiresAt:        time.Now().Add(time.Duration(minecraftAuth.ExpiresIn) * time.Second),
	}

	return creds, nil
}

func jsonPostReqFromText(url string, text string) (*http.Request, error) {
	body := bytes.NewBufferString(text)
	req, _ := http.NewRequest("POST", url, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return req, nil
}
