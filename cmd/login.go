package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/minepkg/minepkg/internals/auth"
	"github.com/minepkg/minepkg/internals/instances"
	"github.com/minepkg/minepkg/internals/minecraft/microsoft"
	"github.com/minepkg/minepkg/internals/minecraft/mojang"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

func init() {
	rootCmd.AddCommand(loginCmd)
}

func (r *Root) restoreAuth() {
	authStore := r.minecraftAuthStore

	authData := &auth.PersistentCredentials{}
	err := authStore.Get(authData)
	if err != nil {
		log.Println("Failed to restore auth data:", err)
	}

	if authData == nil {
		log.Println("No auth data to restore found")
		return
	}

	switch authData.Provider {
	case "mojang":
		state := &mojang.AuthResponse{}
		if err := json.Unmarshal(authData.Data, state); err != nil {
			log.Println("Failed to restore auth data:", err)
		}
		r.useMojangAuth().SetAuthState(state)
	case "microsoft":
		state := &microsoft.Credentials{}
		if err := json.Unmarshal(authData.Data, state); err != nil {
			log.Println("Failed to restore auth data:", err)
		}
		r.useMicrosoftAuth().SetAuthState(state)
	default:
		log.Println("Unknown auth provider:", authData.Provider)
	}
}

func (r *Root) useMojangAuth() *auth.Mojang {
	provider := &auth.Mojang{
		MojangClient: mojang.New(r.HTTPClient),
		Store:        r.minecraftAuthStore,
	}
	r.authProvider = provider
	return provider
}

func (r *Root) useMicrosoftAuth() *auth.Microsoft {
	provider := &auth.Microsoft{
		MicrosoftClient: microsoft.New(root.HTTPClient, &oauth2.Config{
			ClientID:     "056aa695-390f-4d6d-a1b6-fc52d083ccc9",
			ClientSecret: "",
			RedirectURL:  "http://localhost:27893",
		}),
		Store: r.minecraftAuthStore,
	}
	r.authProvider = provider
	return provider
}

func (r *Root) getLaunchCredentialsOrLogin() (*instances.LaunchCredentials, error) {
	if r.authProvider == nil {
		r.restoreAuth()
	}

	// still nothing, we need to login
	if r.authProvider == nil {
		err := r.login()
		if err != nil {
			return nil, err
		}
	}

	creds, err := r.authProvider.LaunchAuthData()
	if err != nil {
		fmt.Println("Could not get launch credentials:", err)
		fmt.Println("Please report the error above and try to login again now:")
		if err = r.login(); err != nil {
			return nil, err
		}
		// we try again
		creds, err = r.authProvider.LaunchAuthData()
		if err != nil {
			return nil, err
		}
	}

	return &instances.LaunchCredentials{
		PlayerName:  creds.GetPlayerName(),
		UUID:        creds.GetUUID(),
		AccessToken: creds.GetAccessToken(),
	}, nil
}

func (r *Root) login() error {
	methodP := promptui.Select{
		Label: "Please choose a login method",
		Items: []string{"Microsoft Account (Opens in Browser)", "Mojang Account (Email & Password)"},
	}
	method, _, err := methodP.Run()
	if err != nil {
		fmt.Println("Aborting")
		os.Exit(0)
	}

	// MS login
	if method == 0 {
		r.useMicrosoftAuth()
		err := r.authProvider.Prompt()
		if err != nil {
			return fmt.Errorf("ms login failed: %w", err)
		}

		return nil
	}

	// Mojang login
	r.useMojangAuth()
	err = r.authProvider.Prompt()
	if err != nil {
		return fmt.Errorf("mojang login failed: %w", err)
	}

	return nil
}

var loginCmd = &cobra.Command{
	Use:     "login",
	Aliases: []string{"signin"},
	Short:   "Sign in with Microsoft or Mojang in order to start Minecraft",
	Args:    cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		err := root.login()
		if err != nil {
			return err
		}
		fmt.Println("Successfully logged in!")
		return nil
	},
	Hidden: true,
}
