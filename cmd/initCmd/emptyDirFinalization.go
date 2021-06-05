package initCmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jwalton/gchalk"
	"github.com/manifoldco/promptui"
	"github.com/minepkg/minepkg/internals/fabric"
	"github.com/minepkg/minepkg/internals/license"
	"github.com/minepkg/minepkg/internals/utils"
	"github.com/minepkg/minepkg/pkg/manifest"
	"github.com/spf13/viper"
)

func (i *initRunner) emptyDirFinalization(man *manifest.Manifest) error {
	if man.Package.Type == manifest.TypeMod {
		fmt.Println("\nThis directory is empty. Do you want to use a template?")
		q := &promptui.Select{
			Label: "Template",
			Items: []string{
				"Fabric Example Mod (https://github.com/FabricMC/fabric-example-mod)",
				"No template",
			},
		}
		selection, _, err := q.Run()
		if err != nil {
			fmt.Println("Aborting")
			os.Exit(1)
		}

		fmt.Println()
		if selection == 0 {
			if _, err := utils.SimpleGitExec("clone https://github.com/FabricMC/fabric-example-mod ."); err != nil {
				return err
			}
			logger.Info(" ✓ Cloned template repository")
		}
	}

	if err := writeLicense(man); err == nil {
		logger.Info(" ✓ Created LICENSE file for you (please double check it)")
	}

	if err := writeReadme(man); err == nil {
		logger.Info(" ✓ Created README.md file")
	}

	if man.Package.Type == manifest.TypeMod && man.Package.Platform == "fabric" {
		if err := syncFabricMod(man); err == nil {
			logger.Info(" ✓ Updated fabric.mod.json to match your minepkg.toml")
		}
	}

	if man.Package.Source != "" {
		newRemote := man.Package.Source
		if !strings.HasSuffix(man.Package.Source, ".git") {
			newRemote = man.Package.Source + ".git"
		}

		if _, err := utils.SimpleGitExec("remote set-url origin " + newRemote); err == nil {
			logger.Info(" ✓ Setting your git remote to " + newRemote)
		}

		if viper.GetString("init.defaultSource") == "" {
			u, err := url.Parse(man.Package.Source)
			if err != nil {
				return err
			}

			dir, _ := path.Split(u.Path)
			u.Path = dir

			newDefaultSource := u.String()

			viper.Set("init.defaultSource", newDefaultSource)

			configDir, err := os.UserConfigDir()
			if err != nil {
				return err
			}

			if err = viper.WriteConfigAs(filepath.Join(configDir, "minepkg/config.toml")); err == nil {
				dot := gchalk.Yellow("·")
				fmt.Printf(
					"  %s Setting default source to %s for next init\n",
					dot,
					newDefaultSource,
				)
				fmt.Printf(
					`  %s Change with "minepkg config set init.defaultSource <value>"`,
					dot,
				)
			}
		}
	}

	return nil
}

func syncFabricMod(man *manifest.Manifest) error {
	file, err := os.Open("./src/main/resources/fabric.mod.json")
	if err != nil {
		return err
	}
	defer file.Close()

	var fabricManifest fabric.Manifest
	if err := json.NewDecoder(file).Decode(&fabricManifest); err != nil {
		return err
	}

	fabricManifest.ID = man.Package.Name
	fabricManifest.Name = man.Package.Name
	fabricManifest.License = man.Package.License
	fabricManifest.Authors = []string{man.AuthorName()}
	fabricManifest.Description = man.Package.Description

	fabricManifest.Contact.Email = man.AuthorEmail()
	fabricManifest.Contact.Homepage = man.Package.Homepage
	fabricManifest.Contact.Sources = man.Package.Source

	if fabricV, ok := man.Dependencies["fabric"]; ok {
		fabricManifest.Depends["fabric"] = fabricV
	}
	fabricManifest.Depends["fabricloader"] = man.Requirements.Fabric
	fabricManifest.Depends["minecraft"] = man.Requirements.Minecraft

	newFile, err := os.Create("./src/main/resources/fabric.mod.json")
	if err != nil {
		return err
	}
	defer newFile.Close()
	encoder := json.NewEncoder(newFile)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(fabricManifest)
}

func writeLicense(man *manifest.Manifest) error {
	license, err := license.GetLicense(man.Package.License)
	if err != nil {
		return err
	}

	author := man.AuthorName()
	if author == "" {
		author = "Contributors"
	}

	replacer := strings.NewReplacer(
		"[year]", strconv.Itoa(time.Now().Year()),
		"[fullname]", author,
	)

	finalLicense := replacer.Replace(license.Body)
	return os.WriteFile("LICENSE", []byte(finalLicense), 0655)
}

func writeReadme(man *manifest.Manifest) error {
	content := []byte(fmt.Sprintf("# %s\n\n%s\n", man.Package.Name, man.Package.Description))
	return os.WriteFile("README.md", content, 0655)
}
