package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/fiws/minepkg/pkg/api"

	"github.com/fatih/color"
)

var infoColor = color.New(color.FgCyan).Add(color.Bold)
var successColor = color.New(color.FgGreen)

// HumanUint32 returns the number in a human readable format
func HumanUint32(num uint32) string {
	switch {
	case num >= 1000000000:
		return fmt.Sprintf("%v B", num/1000000000)
	case num >= 1000000:
		return fmt.Sprintf("%v M", num/1000000)
	case num >= 1000:
		return fmt.Sprintf("%v K", num/1000)
	}
	return fmt.Sprintf("%v", num)
}

// HumanFloat32 returns the number in a human readable format
func HumanFloat32(num float32) string {
	switch {
	case num >= 1000000000:
		return fmt.Sprintf("%v B", num/1000000000)
	case num >= 1000000:
		return fmt.Sprintf("%v M", num/1000000)
	case num >= 1000:
		return fmt.Sprintf("%v K", num/1000)
	}
	return fmt.Sprintf("%v", num)
}

func ensureMojangAuth() (*api.AuthResponse, error) {
	var loginData = &api.AuthResponse{}
	// check if user is logged in
	if rawCreds, err := ioutil.ReadFile(filepath.Join(globalDir, "credentials.json")); err == nil {
		if err := json.Unmarshal(rawCreds, &loginData); err == nil && loginData.Token != "" {
			apiClient.JWT = loginData.Token
			apiClient.User = loginData.User
		} else {
			logger.Info("You need to sign in with your mojang account to launch minecraft")
			login()
		}
	}

	newCreds, err := apiClient.MojangEnsureToken(
		loginData.Mojang.AccessToken,
		loginData.Mojang.ClientToken,
	)
	if err != nil {
		// TODO: check if expired or other problem!
		logger.Info("Your token maybe expired. Please login again")
		login()
	}

	loginData.Mojang.AccessToken = newCreds.AccessToken
	loginData.Mojang.ClientToken = newCreds.ClientToken

	// TODO: only do when needed
	{
		creds, err := json.Marshal(loginData)
		if err != nil {
			return nil, err
		}
		credFile := filepath.Join(globalDir, "credentials.json")
		if err := ioutil.WriteFile(credFile, creds, 0700); err != nil {
			logger.Fail("Count not write credentials file: " + err.Error())
		}
	}

	return loginData, nil
}
