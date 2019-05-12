package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/fiws/minepkg/pkg/api"
	"github.com/manifoldco/promptui"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:     "login",
	Aliases: []string{"signin", "register", "signup"},
	Short:   "Signin to minepkg.io with using your Mojang account",
	Run: func(cmd *cobra.Command, args []string) {
		login()
	},
}

func login() {
	fmt.Println("Please sign in with your Mojang (Minecraft) credentials")
	fmt.Println("Your password is sent encrypted to Mojang directly and NOT saved anywhere.")

	uPrompt := promptui.Prompt{
		Label:    "Please enter your Mojang username (email)",
		Validate: basicValidation,
	}
	username, _ := uPrompt.Run()

	pPrompt := promptui.Prompt{
		Label:    "Please enter your Mojang password",
		Validate: basicValidation,
		Mask:     '■',
	}
	password, _ := pPrompt.Run()

	client := api.New()

	auth, err := client.Login(username, password)
	if err != nil {
		logger.Fail("Probably invalid credentials. not sure: " + err.Error())
	}
	loginData = auth
	creds, err := json.Marshal(auth)

	credFile := filepath.Join(globalDir, "credentials.json")
	if err := ioutil.WriteFile(credFile, creds, 0700); err != nil {
		logger.Fail("Count not write credentials file: " + err.Error())
	}
	fmt.Println("Succesfully loged into minepkg.io")
}

func basicValidation(input string) error {
	if len(input) == 0 {
		return errors.New("You have to enter something …")
	}
	return nil
}
