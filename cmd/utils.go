package cmd

import (
	"fmt"
	"strings"

	"github.com/fiws/minepkg/internals/instances"
	"github.com/fiws/minepkg/pkg/mojang"
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

func ensureMojangAuth() (*mojang.AuthResponse, error) {
	var loginData = &mojang.AuthResponse{}

	if credStore.MojangAuth == nil || credStore.MojangAuth.AccessToken == "" {
		loginData = login()
		if err := credStore.SetMojangAuth(loginData); err != nil {
			return nil, err
		}
		return credStore.MojangAuth, nil
	}

	loginData, err := mojangClient.MojangEnsureToken(
		credStore.MojangAuth.AccessToken,
		credStore.MojangAuth.ClientToken,
	)
	if err != nil {
		// TODO: check if expired or other problem!
		logger.Info("Your token maybe expired. Please login again")
		// TODO: error handling!
		loginData = login()
	}

	// only update access token and client token
	// because `SelectedProfile` is omited here
	credStore.MojangAuth.AccessToken = loginData.AccessToken
	credStore.MojangAuth.ClientToken = loginData.ClientToken

	// HACK: maybe not pass credstore its own field
	if err := credStore.SetMojangAuth(credStore.MojangAuth); err != nil {
		return nil, err
	}
	return credStore.MojangAuth, nil
}

func instanceReqOverwrites(instance *instances.Instance) {
	if overwriteFabricVersion != "" {
		instance.Manifest.Requirements.Fabric = overwriteFabricVersion
	}
	if overwriteMcVersion != "" {
		fmt.Println("mc version overwritten!")
		instance.Manifest.Requirements.Minecraft = overwriteMcVersion
	}
	if overwriteCompanion != "" {
		fmt.Println("companion overwritten!")
		instance.Manifest.Requirements.MinepkgCompanion = overwriteCompanion
	}
}
