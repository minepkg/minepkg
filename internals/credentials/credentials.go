package credentials

import (
	"encoding/json"
	"fmt"

	"github.com/fiws/minepkg/pkg/mojang"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

var (
	minepkgAuthService = "minepkg"
	minepkgAuthUser    = "minepkg_auth_data"

	mojangAuthService = "minepkg"
	mojangAuthUser    = "mojang_auth_data"
)

// Store stures the minepkg & mojang tokens
type Store struct {
	MinepkgAuth *oauth2.Token
	MojangAuth  *mojang.AuthResponse
}

// New creates a new downloadmgr
func New() *Store {
	store := &Store{}
	store.Find()
	return store
}

// Find tries to find existing credentials
func (s *Store) Find() {
	// find minepkg credentials
	minepkgAuth, err := keyring.Get(minepkgAuthService, minepkgAuthUser)
	if err == nil {
		err := json.Unmarshal([]byte(minepkgAuth), &s.MinepkgAuth)
		if err != nil {
			fmt.Println("Oh oh. Could not decode minepkg auth. This should never happen!")
			fmt.Println(err.Error())
		}
	}

	// find mojang credentials
	mojangAuth, err := keyring.Get(mojangAuthService, mojangAuthUser)
	if err == nil {
		err := json.Unmarshal([]byte(mojangAuth), &s.MojangAuth)
		if err != nil {
			fmt.Println("Oh oh. Could not decode mojang auth. This should never happen!")
			fmt.Println(err.Error())
		}
	}
}

// SetMojangAuth sets `MojangAuth` and persists it to disk
func (s *Store) SetMojangAuth(auth *mojang.AuthResponse) error {
	s.MojangAuth = auth

	authJSONBlob, err := json.Marshal(s.MojangAuth)
	if err != nil {
		return err
	}
	return keyring.Set(mojangAuthService, mojangAuthUser, string(authJSONBlob))
}

// SetMinepkgAuth sets `MinepkgAuth` and persists it to disk
func (s *Store) SetMinepkgAuth(auth *oauth2.Token) error {
	s.MinepkgAuth = auth

	authJSONBlob, err := json.Marshal(s.MinepkgAuth)
	if err != nil {
		return err
	}
	return keyring.Set(minepkgAuthService, minepkgAuthUser, string(authJSONBlob))
}

// TODO: implement fallback to credential store

// // check if user is logged in
// if rawCreds, err := ioutil.ReadFile(filepath.Join(globalDir, "credentials.json")); err == nil {
// 	if err := json.Unmarshal(rawCreds, &loginData); err == nil && loginData.Token != "" {
// 		apiClient.JWT = loginData.Token
// 		apiClient.User = loginData.User
// 	}
// }

// creds, err := json.Marshal(auth)

// os.MkdirAll(globalDir, os.ModePerm)
// credFile := filepath.Join(globalDir, "credentials.json")
// if err := ioutil.WriteFile(credFile, creds, 0700); err != nil {
// 	logger.Fail("Count not write credentials file: " + err.Error())
// }
