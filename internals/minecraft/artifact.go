package minecraft

import "encoding/json"

// Artifact is an object describing a "thing" that can be downloaded
// It is used to download libraries and the minecraft client itself
type Artifact struct {
	// Path of the jar file relative to the libraries folder
	// Path is not set for the minecraft client itself
	Path string `json:"path,omitempty"`
	Sha1 string `json:"sha1"`
	// Size in bytes
	Size json.Number `json:"size"`
	// URL to download the jar file
	URL string `json:"url"`
}
