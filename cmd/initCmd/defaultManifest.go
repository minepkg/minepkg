package initCmd

import (
	"fmt"
	"net/url"
	"os"
	"os/user"
	"path"
	"path/filepath"

	"github.com/Masterminds/semver/v3"
	"github.com/magiconair/properties"
	"github.com/minepkg/minepkg/internals/fabric"
	"github.com/minepkg/minepkg/internals/utils"
	"github.com/minepkg/minepkg/pkg/manifest"
	"github.com/spf13/viper"
	"github.com/stoewer/go-strcase"
)

func defaultManifest() *manifest.Manifest {
	fabricMan := &fabric.Manifest{}
	man := manifest.New()

	err := utils.ReadJSONFile("./src/main/resources/fabric.mod.json", fabricMan)
	if err == nil {
		fmt.Println("Detected Fabric mod! Using fabric.mod.json for default values")
		if fabricMan.ID != "" {
			man.Package.Name = fabricMan.ID
		}
		man.Package.Type = "mod"
		man.Package.Platform = "fabric"
		// TODO: check for placeholder!
		man.Package.Version = defaultVersion(fabricMan)
		if fabricMan.License != "" {
			man.Package.License = fabricMan.License
		}
		man.Package.Description = fabricMan.Description
		man.Package.Source = fabricMan.Contact.Sources
		man.Package.Homepage = fabricMan.Contact.Homepage

		if len(fabricMan.Authors) != 0 {
			man.Package.Author = fabricMan.Authors[0]
		} else {
			man.Package.Author = getDefaultAuthor()
		}

		if mcDep, ok := fabricMan.Depends["minecraft"]; ok {
			man.Requirements.Minecraft = mcDep
		} else {
			man.Requirements.Minecraft = "~1.16.2"
		}

		if fabricReq, ok := fabricMan.Depends["fabricloader"]; ok {
			man.Requirements.FabricLoader = fabricReq
		} else {
			man.Requirements.FabricLoader = "*"
		}

		if fabricDep, ok := fabricMan.Depends["fabric"]; ok {
			man.Dependencies["fabric"] = fabricDep
		} else {
			man.Dependencies["fabric"] = "*"
		}
		return man
	}

	wd, _ := os.Getwd()
	defaultName := strcase.KebabCase(filepath.Base(wd))

	// manifest with some defaults

	man.Package.Name = defaultName
	man.Package.Type = "modpack"
	man.Package.Platform = "fabric"
	man.Package.Version = "0.1.0"
	man.Package.License = "MIT"
	man.Package.Author = getDefaultAuthor()

	source := viper.GetString("init.defaultSource")
	if source != "" {
		if u, err := url.Parse(source); err == nil {
			u.Path = path.Join(u.Path, man.Package.Name)
			source = u.String()
		}
	}
	man.Package.Source = source

	man.Requirements.FabricLoader = "*"
	man.Requirements.Minecraft = "~1.16.2"

	return man
}

func defaultVersion(fm *fabric.Manifest) string {
	fabricIsValid := func() bool {
		// is not set or placeholder?
		if fm == nil || fm.Version == "" || fm.Version == "${version}" {
			return false
		}

		// is valid semver?
		_, err := semver.NewVersion(fm.Version)
		return err == nil
	}()

	if fabricIsValid {
		return fm.Version
	}

	gradleProps, err := properties.LoadFile("./gradle.properties", properties.UTF8)
	if err != nil {
		return fallbackVersion
	}

	return gradleProps.GetString("mod_version", fallbackVersion)
}

func getDefaultAuthor() string {
	author := ""

	userName, err := utils.SimpleGitExec("config user.name")
	if err != nil {
		osUser, err := user.Current()
		if err != nil {
			return author
		}
		author = osUser.Name
		if author == "" {
			author = osUser.Username
		}
		return author
	}

	author = userName

	email, err := utils.SimpleGitExec("config user.email")
	if err != nil || email == "" {
		return author
	}

	return fmt.Sprintf("%s <%s>", author, email)
}
