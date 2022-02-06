package instances

type LaunchCredentials struct {
	PlayerName  string
	UUID        string
	AccessToken string
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
