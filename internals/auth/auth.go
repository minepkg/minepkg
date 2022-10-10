package auth

import (
	"encoding/json"

	"github.com/minepkg/minepkg/internals/minecraft"
)

type AuthProvider interface {
	// Name returns the name of the auth provider
	Name() string
	// Prompt asks the user to authenticate
	Prompt() error
	// LaunchAuthData returns the auth data needed to launch the game
	LaunchAuthData() (minecraft.LaunchAuthData, error)
}

type PersistentCredentials struct {
	Provider string
	Data     json.RawMessage
}
