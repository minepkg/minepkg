package pkgid

import (
	"fmt"
	"strings"

	"github.com/minepkg/minepkg/pkg/manifest"
)

type ID struct {
	Provider string
	Platform string
	Name     string
	Version  string
}

func (m *ID) LegacyID() string {
	return fmt.Sprintf("%s@%s", m.Name, m.Version)
}

func NewFromManifest(m *manifest.Manifest) *ID {
	return &ID{
		Provider: "minepkg",
		Platform: m.PlatformString(),
		Name:     m.Package.Name,
		Version:  m.Package.Version,
	}
}

func Parse(id string) *ID {
	return parseIt(id, false)
}

func ParseLikeVersion(id string) *ID {
	return parseIt(id, true)
}

func parseIt(id string, likeVersion bool) *ID {
	newId := &ID{}

	parts := strings.SplitN(id, ":", 2)
	if len(parts) == 2 {
		newId.Provider = parts[0]
		id = parts[1]
	}

	parts = strings.SplitN(id, "/", 2)
	if len(parts) == 2 {
		newId.Platform = parts[0]
		id = parts[1]
	}

	parts = strings.SplitN(id, "@", 2)
	if len(parts) == 2 {
		newId.Name = parts[0]
		newId.Version = parts[1]
	} else {
		if likeVersion {
			newId.Version = id
		} else {
			newId.Name = id
		}
	}

	return newId
}
