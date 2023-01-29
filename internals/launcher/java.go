package launcher

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Masterminds/semver/v3"
	"github.com/minepkg/minepkg/internals/java"
)

func (l *Launcher) Java(ctx context.Context) (*java.Java, error) {
	if l.java != nil {
		return l.java, nil
	}

	javaFactory, err := l.javaFactory()
	if err != nil {
		return nil, err
	}

	// java version overwritten
	if l.JavaVersion != "" {
		java, err := javaFactory.Version(ctx, l.JavaVersion)
		if err != nil {
			return nil, err
		}
		l.java = java
		return java, nil
	}

	// default to java 8
	v := "8"

	// but use java version from launcher json if set
	if l.LaunchManifest.JavaVersion != nil && l.LaunchManifest.JavaVersion.MajorVersion != 0 {
		v = fmt.Sprintf("%d", l.LaunchManifest.JavaVersion.MajorVersion)
	} else {
		// or fallback to v16 for 1.17
		mcSemver := semver.MustParse(l.Instance.Lockfile.MinecraftVersion())
		if mcSemver.Equal(semver.MustParse("1.17.0")) || mcSemver.GreaterThan(semver.MustParse("1.17.0")) {
			v = "16"
		}
	}

	// do not allow to go lower than 8
	intV, _ := strconv.Atoi(v)
	if intV < 8 {
		v = "8"
		log.Println("Java version from launcher manifest is too low, falling back to 8")
	}

	log.Println("Using java version", v)

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
