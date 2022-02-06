package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/minepkg/minepkg/internals/credentials"
	"github.com/minepkg/minepkg/internals/minecraft"
	"github.com/minepkg/minepkg/internals/minecraft/mojang"
)

type Mojang struct {
	*mojang.MojangClient
	AuthData *mojang.AuthResponse
	Store    *credentials.Store
}

func (m *Mojang) SetAuthState(authData *mojang.AuthResponse) error {
	log.Printf("Restoring Mojang auth state")
	m.AuthData = authData
	return nil
}

func (m *Mojang) Prompt() error {
	// Mojang login
	fmt.Println("Please sign in with your Mojang (Minecraft) credentials")
	fmt.Println("Your password is sent encrypted to Mojang directly and NOT saved anywhere.")

	uPrompt := promptui.Prompt{
		Label:    "Please enter your Mojang username (email)",
		Validate: basicValidation,
	}
	username, err := uPrompt.Run()
	if err != nil {
		fmt.Println("Aborting")
		os.Exit(0)
	}

	pPrompt := promptui.Prompt{
		Label:    "Please enter your Mojang password",
		Validate: basicValidation,
		Mask:     '■',
	}
	password, err := pPrompt.Run()
	if err != nil {
		fmt.Println("Aborting")
		os.Exit(0)
	}

	auth, err := m.Login(username, password)
	if err != nil {
		return fmt.Errorf("probably invalid credentials. not sure: %w", err)
	}
	m.AuthData = auth

	return nil
}

func (m *Mojang) LaunchAuthData() (minecraft.LaunchAuthData, error) {
	auth, err := m.MojangEnsureToken(m.AuthData.AccessToken, m.AuthData.ClientToken)
	if err != nil {
		return nil, err
	}
	m.AuthData = auth
	if err := m.persist(); err != nil {
		return nil, err
	}
	return auth, nil
}

func (m *Mojang) persist() error {
	log.Println("Persisting Mojang auth data")
	data, _ := json.Marshal(m.AuthData)
	return m.Store.Set(&PersistentCredentials{
		Provider: "mojang",
		Data:     data,
	})
}

func basicValidation(input string) error {
	if len(input) == 0 {
		return errors.New("you have to enter something …")
	}
	return nil
}
