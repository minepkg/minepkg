package manifest

import "strings"

// InterpretedDependency is a key-value dependency that has been interpreted.
// It can help to fetch the dependency more easily
type InterpretedDependency struct {
	// Provider is the system that should be used to fetch this dependency.
	// This usually is `minepkg` and can also be `https`. There might be more providers in the future
	Provider string
	// Name is the name of the package
	Name string
	// Source is what `Provider` will need to fetch the given Dependency
	// In practice this is a version number for `Provider === "minepkg"` and
	// a https url for `Provider === "https"`
	Source string
}

// InterpretedDependencies returns the dependencies in a `[]*InterpretedDependency` slice.
// See `InterpretedDependency` for details
func (m *Manifest) InterpretedDependencies() []*InterpretedDependency {
	interpreted := make([]*InterpretedDependency, len(m.Dependencies))

	i := 0
	for name, source := range m.Dependencies {
		switch {
		case strings.HasPrefix(source, "https://"):
			interpreted[i] = &InterpretedDependency{Name: name, Provider: "https", Source: source}
		default:
			interpreted[i] = &InterpretedDependency{Name: name, Provider: "minepkg", Source: source}
		}
		i++
	}

	return interpreted
}
