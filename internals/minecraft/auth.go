package minecraft

type LaunchAuthData interface {
	GetAccessToken() string
	GetPlayerName() string
	GetUUID() string
	GetUserType() string
	GetXUID() string
}
