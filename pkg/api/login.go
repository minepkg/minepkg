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
	"os/exec"
	"runtime"

	"github.com/dchest/uniuri"
	"golang.org/x/oauth2"
)

// OAuthLoginConfig describes your oauth app
type OAuthLoginConfig struct {
	ClientID     string
	ClientSecret string
	Scopes       []string
}

// OAuthLogin opens a browser that promts the user to authorise this app
// and returns oauth credentials
func (m *MinepkgAPI) OAuthLogin(c *OAuthLoginConfig) *oauth2.Token {

	// ctx := context.Background()
	conf := &oauth2.Config{
		ClientID:     "minepkg-cli",
		ClientSecret: "",
		Scopes:       []string{"offline", "full_access"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  GetAPIUrl() + "/oauth2/auth",
			TokenURL: GetAPIUrl() + "/oauth2/token",
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
			responseErr = errors.New("Request was intercepted. Try logging in again")

		case code == "":
			// TODO: better description
			maybeErr := query.Get("error")
			responseErr = errors.New("Web login failed with " + maybeErr)
		}
		go server.Shutdown(context.TODO())
	})

	openbrowser(url)
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
	return token
}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}

}
