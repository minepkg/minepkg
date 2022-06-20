package microsoft

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

func (m *MicrosoftClient) SetOauthToken(token *oauth2.Token) {
	m.Token = token
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, m.xblClient)
	m.xblClient = m.Config.Client(ctx, token)
}

func (m *MicrosoftClient) Oauth(ctx context.Context) error {
	conf := m.Config

	state := uniuri.New()
	pkceVerifier := uniuri.NewLen(128)
	pkceHash := sha256.New()
	pkceHash.Write([]byte(pkceVerifier))
	pkceChallenge := base64.RawURLEncoding.EncodeToString(pkceHash.Sum(nil))

	pkceMethod := oauth2.SetAuthURLParam("code_challenge_method", "S256")
	pkceValue := oauth2.SetAuthURLParam("code_challenge", pkceChallenge)
	consumerTenant := oauth2.SetAuthURLParam("tenant", "consumer")
	url := conf.AuthCodeURL(
		state,
		// oauth2.AccessTypeOffline,
		pkceMethod,
		pkceValue,
		consumerTenant,
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
			fmt.Printf("%+v\n", query)
		}
		go server.Shutdown(ctx)
	})

	utils.OpenBrowser(url)
	// todo: error handling!
	server.ListenAndServe()
	defer server.Shutdown(ctx)

	if responseErr != nil {
		fmt.Println("Could not login:\n  " + responseErr.Error())
		os.Exit(1)
	}

	// we have the code
	token, err := conf.Exchange(
		ctx,
		code,
		pkceMethod,
		oauth2.SetAuthURLParam("code_verifier", pkceVerifier),
	)
	if err != nil {
		log.Fatal(err)
	}

	m.SetOauthToken(token)
	return nil
}
