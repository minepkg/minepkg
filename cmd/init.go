package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver"
	"github.com/fiws/minepkg/pkg/api"
	"github.com/fiws/minepkg/pkg/manifest"
	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	"github.com/stoewer/go-strcase"
	"gopkg.in/src-d/go-git.v4"

	"github.com/spf13/cobra"
)

var projectName = regexp.MustCompile(`^([a-z0-9]|[a-z0-9][a-z0-9-]*[a-z0-9])$`)

var (
	force  bool
	loader string
)

func init() {
	initCmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite all the things")
	initCmd.Flags().StringVarP(&loader, "loader", "l", "forge", "Set the required loader to forge, fabric or none.")
}

var initCmd = &cobra.Command{
	Use:   "init [modpack/mod]",
	Short: "Creates a new mod or modpack in the current directory. Can also generate a minepkg.toml for existing directories.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := ioutil.ReadFile("./minepkg.toml"); err == nil && force != true {
			logger.Fail("This directory already contains a minepkg.toml. Use --force to overwrite it")
		}

		manifestType := manifest.TypeMod
		if len(args) == 0 || args[0] == "" || args[0] == "modpack" {
			manifestType = manifest.TypeModpack
		}

		man := manifest.Manifest{}

		loader = strings.ToLower(loader)
		if loader != "forge" && loader != "fabric" && loader != "none" {
			logger.Fail("Allowed values for loader option: forge, fabric or none")
		}
		var (
			// emptyDir bool
			gitRepo bool
		)

		// files, err := ioutil.ReadDir("./build/libs")
		// if err != nil {
		// 	logger.Fail(err.Error())
		// }
		// emptyDir = len(files) == 0

		chForgeVersions := make(chan *api.ForgeVersionResponse)
		go func(ch chan *api.ForgeVersionResponse) {
			res, err := apiClient.GetForgeVersions(context.TODO())
			if err != nil {
				logger.Fail(err.Error())
			}
			ch <- res
		}(chForgeVersions)

		if _, err := git.PlainOpen("./"); err == nil {
			gitRepo = true
		}

		wd, _ := os.Getwd()

		logger.Info("[package]")
		man.Package.Type = manifestType
		man.Package.Name = stringPrompt(&promptui.Prompt{
			Label:   "Name",
			Default: strcase.KebabCase(filepath.Base(wd)),
			Validate: func(s string) error {
				switch {
				case strings.ToLower(s) != s:
					return errors.New("May only contain lowercase characters")
				case strings.HasPrefix(s, "-"):
					return errors.New("May not start with a –")
				case strings.HasSuffix(s, "-"):
					return errors.New("May not end with a –")
				case projectName.MatchString(s) != true:
					return errors.New("May only contain alphanumeric characters and dashes -")
				}
				return nil
			},
		})

		man.Package.Description = stringPrompt(&promptui.Prompt{
			Label:   "Description",
			Default: "",
		})

		// TODO: maybe check local "LICENCE" file for popular licences
		man.Package.License = stringPrompt(&promptui.Prompt{
			Label:   "License",
			Default: "MIT",
		})

		// not using git. ask for the version
		if gitRepo == false {
			logger.Info("You can use git tags for versioning if you want: just submit an empty version")
			man.Package.Version = stringPrompt(&promptui.Prompt{
				Label:   "Version",
				Default: "1.0.0",
				Validate: func(s string) error {
					switch {
					case s == "":
						return nil
					case strings.HasPrefix(s, "v"):
						return errors.New("Please do not include v as a prefix")
					}

					if _, err := semver.NewVersion(s); err != nil {
						return errors.New("Not a valid semver version (major.minor.patch)")
					}

					return nil
				},
			})
		} else {
			logger.Info(
				aurora.Gray("Version:").String() +
					" [Using git tags]" +
					aurora.Gray(" (see \"minepkg help manifest\")").String())
		}

		fmt.Println("")
		logger.Info("[requirements]")

		switch loader {
		case "forge":
			forgeReleases := <-chForgeVersions
			man.Requirements.Forge = stringPrompt(&promptui.Prompt{
				Label:   "Minimum Forge version",
				Default: forgeReleases.Versions[0].Version,
				// TODO: validation
			})

			man.Requirements.Minecraft = stringPrompt(&promptui.Prompt{
				Label:   "Supported Minecraft version",
				Default: forgeReleases.Versions[0].McVersion,
				// TODO: validation
			})
		case "fabric":
			man.Requirements.Fabric = stringPrompt(&promptui.Prompt{
				Label: "Minimum Fabric version",
				// TODO: validation
			})
			man.Requirements.Minecraft = stringPrompt(&promptui.Prompt{
				Label:   "Supported Minecraft version",
				Default: "1.14",
				// TODO: validation
			})
		case "none":
			man.Requirements.Minecraft = stringPrompt(&promptui.Prompt{
				Label:   "Supported Minecraft version",
				Default: "1.14",
				// TODO: validation
			})
		}

		// generate hooks section for mods
		if manifestType == manifest.TypeMod {
			files, err := ioutil.ReadDir("./")
			if err != nil {
				logger.Fail(err.Error())
			}
			for _, f := range files {
				if f.Name() == "gradlew" {
					fmt.Println("")
					logger.Info("[hooks]")
					useHook := boolPrompt(&promptui.Prompt{
						Label:     "Do you want to use \"./gradlew build\" as you build hook",
						Default:   "Y",
						IsConfirm: true,
					})
					if useHook == true {
						man.Hooks.Build = "./gradlew build"
					}
				}
			}
		}

		// generate toml
		buf := bytes.Buffer{}
		if err := toml.NewEncoder(&buf).Encode(man); err != nil {
			logger.Fail(err.Error())
		}
		if err := ioutil.WriteFile("minepkg.toml", buf.Bytes(), 0755); err != nil {
			logger.Fail(err.Error())
		}
		logger.Info(" ✓ Created minepkg.toml")
	},
}

func stringPrompt(prompt *promptui.Prompt) string {
	res, err := prompt.Run()
	if err != nil {
		fmt.Println("Aborting")
		os.Exit(1)
	}
	return res
}

func boolPrompt(prompt *promptui.Prompt) bool {
	_, err := prompt.Run()
	if err != nil {
		if err.Error() == "^C" {
			fmt.Println("Aborting")
			os.Exit(1)
		}
		return false
	}
	return true
}