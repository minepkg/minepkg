package manifest

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
)

const (
	ErrorLevelWarn = iota
	ErrorLevelFatal
)

type ValidationError struct {
	message string
	Path    string
	Level   int
}

func (e ValidationError) Error() string {
	return e.message
}

var (
	// ErrUnsupportedManifestVersion is returned when the manifest version is not supported.
	ErrUnsupportedManifestVersion = ValidationError{
		message: "version is not supported",
		Path:    "manifestVersion",
		Level:   ErrorLevelWarn,
	}
	// ErrNameEmpty is returned when the manifest name is empty.
	ErrNameEmpty = ValidationError{
		message: "name is empty",
		Path:    "package.name",
		Level:   ErrorLevelWarn,
	}
	// ErrNameInvalid is returned when the manifest name is invalid.
	ErrNameInvalid = ValidationError{
		message: "name is invalid",
		Path:    "package.name",
		Level:   ErrorLevelWarn,
	}
	// ErrInvalidType is returned when the manifest type is not supported.
	ErrInvalidType = ValidationError{
		message: "type is not supported",
		Path:    "package.type",
		Level:   ErrorLevelFatal,
	}
	// ErrNoMinecraftRequirement is returned when the manifest does not contain a Minecraft requirement.
	ErrNoMinecraftRequirement = ValidationError{
		message: "does not contain a Minecraft requirement",
		Path:    "requirements.minecraft",
		Level:   ErrorLevelFatal,
	}
	// ErrInvalidMinecraftRequirement is returned when the manifest contains an invalid Minecraft requirement.
	ErrInvalidMinecraftRequirement = ValidationError{
		message: "contains an invalid Minecraft requirement",
		Path:    "requirements.minecraft",
		Level:   ErrorLevelFatal,
	}
	// ErrNoLoaderRequirement is returned when the manifest does not contain a loader requirement.
	ErrNoLoaderRequirement = ValidationError{
		message: "does not contain either forge or fabric loader requirement",
		Path:    "requirements",
		Level:   ErrorLevelFatal,
	}
	// ErrInvalidFabricLoaderRequirement is returned when the manifest contains an invalid Fabric loader requirement.
	ErrInvalidFabricLoaderRequirement = ValidationError{
		message: "contains an invalid Fabric loader requirement",
		Path:    "requirements.fabric",
		Level:   ErrorLevelFatal,
	}
)

// helper regexes
var (
	validName = regexp.MustCompile(`^[a-z0-9-_]+$`)
)

type Problems []ValidationError

// Fatal returns the first fatal error in the list. If there are no fatal errors, it returns nil.
func (p *Problems) Fatal() error {
	for _, problem := range *p {
		if problem.Level == ErrorLevelFatal {
			return problem
		}
	}
	return nil
}

func validateMinecraftRequirement(mcVersion string) Problems {
	problems := Problems{}

	if mcVersion == "" {
		problems = append(problems, ErrNoMinecraftRequirement)
		return problems
	}

	_, err := semver.NewConstraint(mcVersion)
	if err != nil {
		fmt.Println(err)
		problems = append(problems, ErrInvalidMinecraftRequirement)
		return problems
	}

	if strings.HasPrefix(mcVersion, "*") || strings.HasPrefix(mcVersion, ">") || strings.HasPrefix(mcVersion, "^") {
		problems = append(problems, ValidationError{
			message: "Minecraft requirement is very broad, prefer patch requirement like ~1.17.0",
			Path:    "requirements.minecraft",
			Level:   ErrorLevelWarn,
		})
	}

	return problems
}

func validateFabricLoader(version string) Problems {
	problems := Problems{}

	_, err := semver.NewConstraint(version)
	if err != nil {
		problems = append(problems, ValidationError{
			message: "manifest contains an invalid fabric loader requirement",
			Path:    "requirements.fabric",
			Level:   ErrorLevelFatal,
		})
	}

	return problems
}

func validateForgeLoader(version string) Problems {
	problems := Problems{}

	_, err := semver.NewConstraint(version)
	if err != nil {
		problems = append(problems, ValidationError{
			message: "manifest contains an invalid fabric loader requirement",
			Path:    "requirements.fabric",
			Level:   ErrorLevelFatal,
		})
	}

	return problems
}

// Validate checks the manifest for correctness.
func (m *Manifest) Validate() Problems {
	problems := Problems{}

	problems = append(problems, validateMinecraftRequirement(m.Requirements.Minecraft)...)

	// manifest version
	if m.ManifestVersion != 0 {
		problems = append(problems, ErrUnsupportedManifestVersion)
	}

	// package name
	switch {
	case m.Package.Name == "":
		problems = append(problems, ErrNameEmpty)
	case !validName.MatchString(m.Package.Name):
		problems = append(problems, ErrNameInvalid)
	}

	// package type
	if m.Package.Type != TypeMod && m.Package.Type != TypeModpack {
		problems = append(problems, ErrInvalidType)
	}

	// license
	if m.Package.License == "" {
		problems = append(problems, ValidationError{
			message: "manifest does not contain a license",
			Path:    "package.license",
			Level:   ErrorLevelWarn,
		})
	}

	// loader requirements
	switch {
	case m.Requirements.FabricLoader != "":
		problems = append(problems, validateFabricLoader(m.Requirements.FabricLoader)...)
	case m.Requirements.ForgeLoader != "":
		problems = append(problems, validateForgeLoader(m.Requirements.ForgeLoader)...)
	default:
		problems = append(problems, ErrNoLoaderRequirement)
	}

	// TODO: validate other fields (dependencies, dev stuff)
	return problems
}
