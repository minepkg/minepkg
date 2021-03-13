package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/fiws/minepkg/internals/mojang"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use:     "login",
	Aliases: []string{"signin"},
	Short:   "Sign in to Mojang in order to start Minecraft",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		login()
	},
}

func login() *mojang.AuthResponse {
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

	auth, err := mojangClient.Login(username, password)
	if err != nil {
		logger.Fail("Probably invalid credentials. not sure: " + err.Error())
	}
	credStore.SetMojangAuth(auth)

	return auth
}

func basicValidation(input string) error {
	if len(input) == 0 {
		return errors.New("you have to enter something …")
	}
	return nil
}
