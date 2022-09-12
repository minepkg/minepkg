package pkgid

import (
	"fmt"
	"strings"
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

func Parse(id string) *ID {
	return parseIt(id, false)
}

func ParseLikeVersion(id string) *ID {
	return parseIt(id, true)
}

func parseIt(id string, likeVersion bool) *ID {
	newId := &ID{Provider: "minepkg"}

	// special case for none
	if id == "none" {
		newId.Provider = "dummy"
		return newId
	}

	// special case for https
	if strings.HasPrefix(id, "https://") {
		newId.Provider = "https"
		newId.Version = id
		return newId
	}

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
		newId.Version = id
	}

	return newId
}
