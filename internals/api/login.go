package api

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dchest/uniuri"
	"github.com/minepkg/minepkg/internals/utils"
	"golang.org/x/oauth2"
)

// OAuthLoginConfig describes your oauth app
type OAuthLoginConfig struct {
	ClientID     string
	ClientSecret string
	Scopes       []string
}

// OAuthLogin opens a browser that prompts the user to authorize this app
// and returns oauth credentials
func (m *MinepkgClient) OAuthLogin(c *OAuthLoginConfig) *oauth2.Token {

	// ctx := context.Background()
	conf := &oauth2.Config{
		ClientID:     "minepkg-cli",
		ClientSecret: "",
		Scopes:       []string{"offline", "full_access"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  m.APIUrl + "/oauth2/auth",
			TokenURL: m.APIUrl + "/oauth2/token",
		},
		// locally started server
		RedirectURL: "http://localhost:27893",
	}

	state := uniuri.New()
	pkceVerifier := uniuri.NewLen(128)
	pkceHash := sha256.New()
	pkceHash.Write([]byte(pkceVerifier))
	pkceChallenge := base64.RawURLEncoding.EncodeToString(pkceHash.Sum(nil))

	pkceMethod := oauth2.SetAuthURLParam("code_challenge_method", "S256")
	pkceValue := oauth2.SetAuthURLParam("code_challenge", pkceChallenge)
	url := conf.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		pkceMethod,
		pkceValue,
	)

	var responseErr error
	code := ""
	server := &http.Server{Addr: "localhost:27893"}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			<!DOCTYPE html>
			<html lang="en">
			<head>
				<script>window.close();</script>
			</head>
			<body>
				<h1>Login attempt done.</h1>
				<div>You can close this window now</div>
			</body>
			</html>
		`)
		r.Close = true
		query := r.URL.Query()
		code = query.Get("code")
		resState := query.Get("state")
		switch {
		case resState != state:
			responseErr = errors.New("request was intercepted – try logging in again")

		case code == "":
			// TODO: better description
			maybeErr := query.Get("error")
			responseErr = errors.New("Web login failed with " + maybeErr)
		}
		go server.Shutdown(context.TODO())
	})

	utils.OpenBrowser(url)
	// todo: error handling!
	server.ListenAndServe()

	if responseErr != nil {
		fmt.Println("Could not login:\n  " + responseErr.Error())
		os.Exit(1)
	}

	// we have the code
	token, err := conf.Exchange(
		context.TODO(),
		code,
		pkceMethod,
		oauth2.SetAuthURLParam("code_verifier", pkceVerifier),
	)
	if err != nil {
		log.Fatal(err)
	}
	// set token for this api client
	m.JWT = token.AccessToken
	return token
}
