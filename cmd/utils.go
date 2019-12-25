package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/fiws/minepkg/pkg/api"
)

// MinepkgMapping is a server mapping (very unfinished)
type MinepkgMapping struct {
	Platform string `json:"platform"`
	Modpack  string `json:"modpack"`
}

func splitPackageName(id string) (string, string) {
	arr := strings.Split(id, "@")
	return arr[0], arr[1]
}

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
			loginData = login()
		}
	} else {
		logger.Info("You need to sign in with your mojang account to launch minecraft")
		loginData = login()
	}

	newCreds, err := apiClient.MojangEnsureToken(
		loginData.Mojang.AccessToken,
		loginData.Mojang.ClientToken,
	)
	if err != nil {
		// TODO: check if expired or other problem!
		logger.Info("Your token maybe expired. Please login again")
		loginData = login()
	} else {
		// only refresh tokens
		if loginData.Mojang == nil {
			loginData.Mojang = &api.MojangAuthResponse{}
		}
		loginData.Mojang.AccessToken = newCreds.AccessToken
		loginData.Mojang.ClientToken = newCreds.ClientToken
	}

	apiClient.JWT = loginData.Token
	apiClient.User = loginData.User

	// TODO: only do when needed
	{
		creds, err := json.Marshal(loginData)
		if err != nil {
			return nil, err
		}
		credFile := filepath.Join(globalDir, "credentials.json")
		if err := ioutil.WriteFile(credFile, creds, 0700); err != nil {
			logger.Fail("Could not write credentials file: " + err.Error())
		}
	}

	return loginData, nil
}
