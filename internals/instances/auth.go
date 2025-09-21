package instances

type LaunchCredentials struct {
	// PlayerName is the name that the player has chosen (appears in the game)
	PlayerName string
	// UUID is the player's UUID (strictly required)
	UUID string
	// AccessToken is the mojang api access token (strictly required)
	AccessToken string

	// UserType show if the account is a Microsoft account
	// allowed values: "msa"
	UserType string
	// XUID is the player's XUID (for Xbox Live accounts) – kinda optional
	XUID string
	// ClientID is the oauth id that was used to authenticate the player (for Xbox Live accounts) – kinda optional
	ClientID string
}

func (i *Instance) getLaunchCredentials() (*LaunchCredentials, error) {
	creds := i.AuthCredentials
	if creds == nil {
		return nil, ErrNoCredentials
	}

	// do not allow non paid accounts to start minecraft
	// unpaid accounts should not have a profile
	if creds.UUID == "" {
		return nil, ErrNoPaidAccount
	}

	return creds, nil
}

func (i *Instance) SetLaunchCredentials(creds *LaunchCredentials) {
	i.AuthCredentials = creds
}
