package forge

import (
	"errors"
	"regexp"
	"strings"
)

// ErrorUnknown is returned if the error could not be parsed
var ErrorUnknown = errors.New("Unknown Error")

// ModRequirement describes a mod dependency
type ModRequirement struct {
	Name    string
	Version string
}

func (m ModRequirement) String() string {
	return m.Name + "@" + m.Version
}

// ErrorMissingMods is return if a mod requires one or multiple
// dependency mods that are not present
type ErrorMissingMods struct {
	ModID    string
	ModName  string
	Requires []ModRequirement
}

func (e ErrorMissingMods) Error() string {
	reqStrings := make([]string, len(e.Requires))
	for i, req := range e.Requires {
		reqStrings[i] = req.String()
	}
	return e.ModID + " requires " + strings.Join(reqStrings, ", ")
}

// ParseException tries to parse an Exception in a LogLine
// currently only returns a `ErrorMissingMods` if possible
// returns a `ErrorUnknown` otherwise
func ParseException(l *LogLine) error {
	r := regexp.MustCompile(`net.minecraftforge.fml.common.MissingModsException: Mod (.+) \((.+)\) requires \[(.+)\]$`)

	found := r.FindStringSubmatch(l.Message)
	if len(found) == 0 {
		return ErrorUnknown
	}

	requireStrings := strings.Split(found[3], "),")
	dep := regexp.MustCompile(`([a-zA-z_-]+)@.(\d+\.\d+.\d+)`)

	requires := make([]ModRequirement, len(requireStrings))
	for i, req := range requireStrings {
		f := dep.FindStringSubmatch(req)
		if len(f) == 0 {
			return ErrorUnknown
		}
		requires[i] = ModRequirement{f[1], f[2]}
	}

	err := &ErrorMissingMods{
		ModID:    found[1],
		ModName:  found[2],
		Requires: requires,
	}

	return err
}
