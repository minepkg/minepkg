package instances

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

const mcVersionsURL string = "https://launchermeta.mojang.com/mc/game/version_manifest.json"

var (
	// TypeSnapshot is a snapshot release
	TypeSnapshot = "snapshot"
	// TypeRelease is a full "normal" release
	TypeRelease = "release"
	// TypeOldBeta is a "old_beta" release
	TypeOldBeta = "old_beta"
	// TypeOldAlpha is a "old_alpha" release
	TypeOldAlpha = "old_alpha"
)

// MinecraftRelease is a released minecraft version
type MinecraftRelease struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	Time        string `json:"time"`
	ReleaseTime string `json:"releaseTime"`
}

// MinecraftReleaseResponse is the response from the "launchermeta" mojang api
type MinecraftReleaseResponse struct {
	Latest struct {
		Release  string `json:"release"`
		Snapshot string `json:"snapshot"`
	} `json:"latest"`
	Versions []MinecraftRelease
}

// GetMinecraftReleases returns all available Minecraft releases
func GetMinecraftReleases(ctx context.Context) (*MinecraftReleaseResponse, error) {
	req, _ := http.NewRequest("GET", mcVersionsURL, nil)

	res, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	parsed := MinecraftReleaseResponse{}
	if err := json.Unmarshal(buf, &parsed); err != nil {
		return nil, err
	}

	return &parsed, nil
}
