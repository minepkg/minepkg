package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"

	"github.com/dchest/uniuri"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

func init() {
	rootCmd.AddCommand(mloginCmd)
}

var mloginCmd = &cobra.Command{
	Use:     "minepkg-login",
	Aliases: []string{"signin"},
	Short:   "Sign in to minepkg.io",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("hello. fake login")

		v := url.Values{}
		v.Set("client_id", "minepkg-cli")
		v.Add("response_type", "code")
		v.Add("redirect_uri", "http://localhost:27893")
		v.Add("state", "TODO")
		v.Add("code_challenge_method", "TODO")
		v.Add("code_challenge", "TODO")
		token := getToken("http://localhost:3000/cli/auth?" + v.Encode())

		service := "minepkg"
		user := "access_token"

		// set token in keyring
		err := keyring.Set(service, user, token.AccessToken)
		if err != nil {
			log.Fatal(err)
		}

		// set refresh token
		err = keyring.Set(service, "refresh_token", token.RefreshToken)
		if err != nil {
			log.Fatal(err)
		}

		// get password
		secret, err := keyring.Get(service, user)
		if err != nil {
			log.Fatal(err)
		}

		log.Println(`Login succesfull, but not used anywhere jet ¯\_(ツ)_/¯`)
		log.Println("But here is yo token: " + secret)
	},
}

func getToken(uri string) *oauth2.Token {

	// ctx := context.Background()
	conf := &oauth2.Config{
		ClientID:     "minepkg-cli",
		ClientSecret: "",
		Scopes:       []string{"offline", "full_access"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://localhost:8080/v1/oauth2/auth",
			TokenURL: "http://localhost:8080/v1/oauth2/token",
		},
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

	code := ""
	server := &http.Server{Addr: "localhost:27893"}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			<!DOCTYPE html>
			<html lang="en">
			<head>
				<script>setTimeout(() => window.close(), 1000);</script>
			</head>
			<body>
				<h1>Login succesfull!</h1>
				<div>You can close this window now</div>
			</body>
			</html>
		`)
		r.Close = true
		query := r.URL.Query()
		code = query.Get("code")
		resState := query.Get("state")
		if resState != state {
			log.Fatal("Request was intercepted. Try logging in again")
		}
		if code == "" {
			maybeErr := query.Get("because")
			fmt.Println("Login failed because: " + maybeErr)
		} else {
			fmt.Println("Login succesfull!")
		}
		go server.Shutdown(context.TODO())
	})
	http.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			<!DOCTYPE html>
			<html lang="en">
			<head>
				<script>window.location = '%s';</script>
			</head>
			<body>
			</body>
			</html>
		`, uri)
	})

	openbrowser(url)
	// todo: error handling!
	server.ListenAndServe()

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
