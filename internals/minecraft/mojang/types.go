package mojang

// AuthResponse is the response returned by a successful mojang login
type AuthResponse struct {
	AccessToken     string   `json:"accessToken"`
	ClientToken     string   `json:"clientToken"`
	SelectedProfile *Profile `json:"selectedProfile"`
}

func (a *AuthResponse) GetAccessToken() string { return a.AccessToken }
func (a *AuthResponse) GetPlayerName() string  { return a.SelectedProfile.Name }
func (a *AuthResponse) GetUUID() string        { return a.SelectedProfile.ID }

// Profile is a profile that potentially can be used to launch minecraft
type Profile struct {
	Agent         string `json:"agent"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	UserID        string `json:"userId"`
	TokenID       string `json:"tokenId"`
	CreatedAt     int    `json:"createdAt"`
	LegacyProfile bool   `json:"legacyProfile"`
	Suspended     bool   `json:"suspended"`
	Paid          bool   `json:"paid"`
	Migrated      bool   `json:"migrated"`
}

type mojangAgent struct {
	Name    string `json:"name"`
	Version uint8  `json:"version"`
}

type mojangLogin struct {
	Agent       mojangAgent `json:"agent"`
	Username    string      `json:"username"`
	Password    string      `json:"password"`
	RequestUser bool        `json:"requestUser"`
}

type mojangError struct {
	ErrorCode    string `json:"error"`
	ErrorMessage string `json:"errorMessage"`
	Cause        string `json:"cause"`
}

func (m mojangError) Error() string {
	return m.ErrorMessage
}
