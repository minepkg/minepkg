package instances

import (
	"context"

	"github.com/Masterminds/semver/v3"
	"github.com/minepkg/minepkg/internals/java"
)

func (i *Instance) Java() (*java.Java, error) {
	if i.java != nil {
		return i.java, nil
	}
	v := uint8(8)

	mcSemver := semver.MustParse(i.Lockfile.MinecraftVersion())
	if mcSemver.Equal(semver.MustParse("1.17.0")) || mcSemver.GreaterThan(semver.MustParse("1.17.0")) {
		v = 16
	}

	java, err := i.JavaFactory.Version(context.TODO(), v)
	if err != nil {
		return nil, err
	}
	i.java = java
	return java, nil
}
