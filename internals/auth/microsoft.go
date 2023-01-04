package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/minepkg/minepkg/internals/credentials"
	"github.com/minepkg/minepkg/internals/minecraft"
	"github.com/minepkg/minepkg/internals/minecraft/microsoft"
	"golang.org/x/oauth2"
)

type Microsoft struct {
	*microsoft.MicrosoftClient
	authData *microsoft.Credentials
	Store    *credentials.Store
}

// MicrosoftCredentialStorage is used to trim down the auth data to the minimum required
// otherwise the windows keyring will return an error ("The stub received bad data.")
type MicrosoftCredentialStorage struct {
	MicrosoftAuth oauth2.Token `json:"ms"`
	PlayerName    string       `json:"pn"`
	UUID          string       `json:"id"`
	AccessToken   string       `json:"at"`
	ExpiresAt     time.Time    `json:"exp"`
	XUID          string       `json:"xuid,omitempty"`
}

func (m *Microsoft) Name() string {
	return "Microsoft"
}

func (m *Microsoft) SetAuthState(authData *MicrosoftCredentialStorage) error {
	log.Printf("Restoring MS auth state")
	m.authData = &microsoft.Credentials{
		ExpiresAt:     authData.ExpiresAt,
		MicrosoftAuth: authData.MicrosoftAuth,
		MinecraftAuth: &microsoft.XboxLoginResponse{
			AccessToken: authData.AccessToken,
		},
		MinecraftProfile: &microsoft.GetProfileResponse{
			ID:   authData.UUID,
			Name: authData.PlayerName,
		},
	}
	m.SetOauthToken(&authData.MicrosoftAuth)
	return nil
}

func (m *Microsoft) Prompt() error {
	ctx := context.Background()
	if err := m.Oauth(context.Background()); err != nil {
		return fmt.Errorf("failed to authenticate with microsoft: %w", err)
	}

	creds, err := m.GetMinecraftCredentials(ctx)
	if err != nil {
		return fmt.Errorf("failed to get minecraft credentials: %w", err)
	}
	m.authData = creds
	if err := m.persist(); err != nil {
		return fmt.Errorf("failed to persist auth data: %w", err)
	}
	return nil
}

func (m *Microsoft) LaunchAuthData() (minecraft.LaunchAuthData, error) {
	// not auth data or it is expired
	if m.authData == nil {
		log.Println("Refreshing MS auth data (non existent)")
		return m.refreshAuthData()
	}
	if m.authData.IsExpired() {
		log.Println("Refreshing MS auth data (expired)")
		return m.refreshAuthData()
	}
	// we have valid and unexpired auth data
	log.Println("Using cached MS auth data")
	return m.authData, nil
}

func (m *Microsoft) refreshAuthData() (*microsoft.Credentials, error) {
	creds, err := m.GetMinecraftCredentials(context.Background())
	if err != nil {
		return nil, err
	}
	m.authData = creds
	if err := m.persist(); err != nil {
		return nil, fmt.Errorf("failed to persist auth data: %w", err)
	}
	return creds, err
}

func (m *Microsoft) persist() error {
	log.Println("Persisting MS auth data")
	trimmedData := &MicrosoftCredentialStorage{
		MicrosoftAuth: m.authData.MicrosoftAuth,
		AccessToken:   m.authData.MinecraftAuth.AccessToken,
		UUID:          m.authData.MinecraftProfile.ID,
		PlayerName:    m.authData.MinecraftProfile.Name,
		ExpiresAt:     m.authData.ExpiresAt,
	}
	// TODO: we should probably find a better way to do this
	if runtime.GOOS != "windows" {
		trimmedData.XUID = m.authData.XUID
		log.Println("storing additional XUID data for non windows platforms", trimmedData.XUID)
	}
	data, err := json.Marshal(trimmedData)
	if err != nil {
		return fmt.Errorf("failed to marshal auth data: %w", err)
	}
	return m.Store.Set(&PersistentCredentials{
		Provider: "microsoft",
		Data:     data,
	})
}
