package minecraft

// LaunchAuthData is an interface defining the data required to authenticate
type LaunchAuthData interface {
	// GetAccessToken returns the access token (strictly required)
	GetAccessToken() string
	// GetAccessToken returns the users UUID (strictly required)
	GetUUID() string
	// GetPlayerName returns the users player name (the one that also appears in game)
	GetPlayerName() string
	// GetUserType returns the users user type (legacy, mojang or msa).
	// "legacy" is the old minecraft account type (should not be encountered anymore)
	// "mojang" was the new minecraft account type (being replaced by "msa")
	// "msa" is the new microsoft account type
	GetUserType() string
	// GetXUID returns the users XUID (only for xbox live accounts – user type "msa"))
	GetXUID() string
}
