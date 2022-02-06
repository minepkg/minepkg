package auth

import (
	"context"
	"encoding/json"
	"log"

	"github.com/minepkg/minepkg/internals/credentials"
	"github.com/minepkg/minepkg/internals/minecraft"
	"github.com/minepkg/minepkg/internals/minecraft/microsoft"
)

type Microsoft struct {
	*microsoft.MicrosoftClient
	authData *microsoft.Credentials
	Store    *credentials.Store
}

func (m *Microsoft) SetAuthState(authData *microsoft.Credentials) error {
	log.Printf("Restoring MS auth state")
	m.authData = authData
	m.SetOauthToken(&authData.MicrosoftAuth)
	return nil
}

func (m *Microsoft) Prompt() error {
	ctx := context.Background()
	if err := m.Oauth(context.Background()); err != nil {
		return err
	}

	creds, err := m.GetMinecraftCredentials(ctx)
	if err != nil {
		return err
	}
	m.authData = creds
	if err := m.persist(); err != nil {
		return err
	}
	return nil
}

func (m *Microsoft) LaunchAuthData() (minecraft.LaunchAuthData, error) {
	// not auth data or it is expired
	if m.authData == nil || m.authData.IsExpired() {
		log.Println("Refreshing MS auth data")
		return m.refreshAuthData()
	}
	// we have valid and unexpired auth data
	log.Println("Using Cached MS auth data")
	return m.authData, nil
}

func (m *Microsoft) refreshAuthData() (*microsoft.Credentials, error) {
	creds, err := m.GetMinecraftCredentials(context.Background())
	if err != nil {
		return nil, err
	}
	m.authData = creds
	if err := m.persist(); err != nil {
		return nil, err
	}
	return creds, err
}

func (m *Microsoft) persist() error {
	log.Println("Persisting MS auth data")
	data, _ := json.Marshal(m.authData)
	return m.Store.Set(&PersistentCredentials{
		Provider: "microsoft",
		Data:     data,
	})
}
