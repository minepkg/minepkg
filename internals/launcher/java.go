package launcher

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver/v3"
	"github.com/minepkg/minepkg/internals/java"
)

func (l *Launcher) Java(ctx context.Context) (*java.Java, error) {
	if l.java != nil {
		return l.java, nil
	}
	v := "8"

	mcSemver := semver.MustParse(l.Instance.Lockfile.MinecraftVersion())
	if mcSemver.Equal(semver.MustParse("1.17.0")) || mcSemver.GreaterThan(semver.MustParse("1.17.0")) {
		v = "16"
	}

	javaFactory, err := l.javaFactory()
	if err != nil {
		return nil, err
	}

	java, err := javaFactory.Version(ctx, v)
	if err != nil {
		return nil, err
	}
	l.java = java

	return java, nil
}

func (l Launcher) javaFactory() (*java.Factory, error) {
	if l.javaFactoryInstance != nil {
		return l.javaFactoryInstance, nil
	}
	userCache, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	l.javaFactoryInstance = java.NewFactory(filepath.Join(userCache, "minepkg", "java"))
	return l.javaFactoryInstance, nil
}
