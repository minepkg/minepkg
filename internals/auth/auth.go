package auth

import (
	"encoding/json"

	"github.com/minepkg/minepkg/internals/minecraft"
)

type AuthProvider interface {
	// Prompt asks the user to authenticate
	Prompt() error
	// LaunchAuthData returns the auth data needed to launch the game
	LaunchAuthData() (minecraft.LaunchAuthData, error)
}

type PersistentCredentials struct {
	Provider string
	Data     json.RawMessage
}
