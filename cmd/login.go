package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/fiws/minepkg/pkg/api"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use:     "login",
	Aliases: []string{"signin"},
	Short:   "Sign in to Mojang and minepkg.io using your Mojang account",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		login()
	},
}

func login() *api.AuthResponse {
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

	client := api.New()

	auth, err := client.Login(username, password)
	if err != nil {
		logger.Fail("Probably invalid credentials. not sure: " + err.Error())
	}
	loginData = auth
	creds, err := json.Marshal(auth)

	os.MkdirAll(globalDir, os.ModePerm)
	credFile := filepath.Join(globalDir, "credentials.json")
	if err := ioutil.WriteFile(credFile, creds, 0700); err != nil {
		logger.Fail("Count not write credentials file: " + err.Error())
	}
	fmt.Println("Succesfully logged in to minepkg.io")

	return loginData
}

func basicValidation(input string) error {
	if len(input) == 0 {
		return errors.New("You have to enter something …")
	}
	return nil
}
