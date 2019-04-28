package api

import (
	"github.com/Masterminds/semver"
)

const (
	// TypeMod indicates a mod
	TypeMod = "mod"
	// TypeModpack indicates a modpack
	TypeModpack = "modpack"
)

// User describes a registered user
type User struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

// AuthResponse gets returned after signin with a token
// or username/password
type AuthResponse struct {
	// User contains the account data like name or email
	User *User `json:"user"`
	// Token is a jwt token
	Token string `json:"token"`
}

// LoginData combines the minepkg token + data with the mojang tokens
type LoginData struct {
	Minepkg *AuthResponse
	Mojang  *mojangAuthResponse
}

// Project is a project … realy
type Project struct {
	client      *MinepkgAPI
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Readme      string `json:"readme,omitempty"`
}

// Release is a released version of a project
type Release struct {
	client       *MinepkgAPI
	Project      string        `json:"Project"`
	Version      string        `json:"version"`
	Published    bool          `json:"published"`
	Requirements Requirements  `json:"requirements"`
	Dependencies []*Dependency `json:"dependencies"`
}

// SemverVersion returns the Version as a `semver.Version` struct
func (r *Release) SemverVersion() *semver.Version {
	return semver.MustParse(r.Version)
}

// Requirements contains the wanted Minecraft version
// and either the required Forge or Fabric version
type Requirements struct {
	Minecraft string `json:"minecraft"`
	Forge     string `json:"forge,omitempty"`
	Fabric    string `json:"fabric,omitempty"`
}

// Dependency in verbose form
type Dependency struct {
	client *MinepkgAPI
	// Provider is only minepkg for now. Kept for future extensions
	Provider string `json:"provider"`
	// Name is the name of the package (eg. storage-drawers)
	Name string `json:"name"`
	// VersionRequirement is a semver version Constraint
	// Example: `^2.9.22` or `5.x.x`
	VersionRequirement string `json:"versionRequirement"`
}

// ForgeVersion is a release of forge
type ForgeVersion struct {
	Branch      string     `json:"branch"`
	Build       int        `json:"build"`
	Files       [][]string `json:"files"`
	McVersion   string     `json:"mcversion"`
	Modified    int        `json:"modified"`
	Version     string     `json:"version"`
	Recommended bool       `json:"recommended"`
}

// ForgeVersionResponse is the response from the /meta/forge-versions endpoint
type ForgeVersionResponse struct {
	Versions []ForgeVersion `json:"versions"`
	Webpath  string         `json:"webpath"`
	Homepage string         `json:"homepage"`
	Adfocus  string         `json:"adfocus"`
}

// MinepkgError is the json response if the response
// was not succesfull
type MinepkgError struct {
	StatusCode string `json:"statusCode"`
	Status     string `json:"error"`
	Message    string `json:"message"`
}

func (m MinepkgError) Error() string {
	return m.Status + ": " + m.Message
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

type mojangAuthResponse struct {
	ClientToken string `json:"clientToken"`
	AccessToken string `json:"accessToken"`
}

type mojangError struct {
	ErrorCode    string `json:"error"`
	ErrorMessage string `json:"errorMessage"`
	Cause        string `json:"cause"`
}

func (m mojangError) Error() string {
	return m.ErrorMessage
}
