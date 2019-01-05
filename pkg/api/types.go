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

// Project is a project … realy
type Project struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// Release is a released version of a project
type Release struct {
	Version      *semver.Version `json:"version"`
	Requirements Requirements    `json:"requirements"`
	Dependencies []*Dependency   `json:"dependencies"`
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
	// Provider is only minepkg for now. Kept for future extensions
	Provider string `json:"provider"`
	// Name is the name of the package (eg. storage-drawers)
	Name string `json:"name"`
	// VersionRequirement is a semver version Constraint
	// Example: `^2.9.22` or `5.x.x`
	VersionRequirement *semver.Constraints `json:"versionRequirement"`
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