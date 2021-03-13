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
	"github.com/Masterminds/semver/v3"
	"github.com/fiws/minepkg/internals/api"
	"github.com/fiws/minepkg/pkg/manifest"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/stoewer/go-strcase"
)

var projectName = regexp.MustCompile(`^([a-z0-9]|[a-z0-9][a-z0-9-]*[a-z0-9])$`)

var (
	force  bool
	loader string
	yes    bool
)

func init() {
	initCmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite the minepkg.toml if one exists")
	initCmd.Flags().BoolVarP(&yes, "yes", "y", false, "Choose defaults for all questions. (non-interactive mode)")
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Creates a new mod or modpack in the current directory",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := ioutil.ReadFile("./minepkg.toml"); err == nil && force != true {
			logger.Fail("This directory already contains a minepkg.toml. Use --force to overwrite it")
		}

		man := manifest.Manifest{}

		chForgeVersions := make(chan *api.ForgeVersionResponse)
		go func(ch chan *api.ForgeVersionResponse) {
			res, err := apiClient.GetForgeVersions(context.TODO())
			if err != nil {
				logger.Fail(err.Error())
			}
			ch <- res
		}(chForgeVersions)

		wd, _ := os.Getwd()
		defaultName := strcase.KebabCase(filepath.Base(wd))
		hasGradle := checkGradle()

		if yes == true {
			man.Package.Name = defaultName
			man.Package.Type = "modpack"
			man.Package.Platform = "fabric"
			man.Package.Version = "0.1.0"

			man.Requirements.Fabric = "*"
			man.Requirements.Minecraft = "1.15"
			// generate toml
			writeManifest(&man)
			logger.Info(" ✓ Created minepkg.toml")
			return
		}

		logger.Info("[package]")
		man.Package.Type = selectPrompt(&promptui.Select{
			Label: "Type",
			Items: []string{"modpack", "mod"},
		})

		man.Package.Platform = selectPrompt(&promptui.Select{
			Label: "Platform",
			Items: []string{"fabric", "forge"},
		})

		man.Package.Name = stringPrompt(&promptui.Prompt{
			Label:   "Name",
			Default: defaultName,
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

		man.Package.Version = stringPrompt(&promptui.Prompt{
			Label:   "Version",
			Default: "",
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

		fmt.Println("")
		logger.Info("[requirements]")

		fmt.Println(man.PlatformString())

		switch man.Package.Platform {
		case "fabric":
			fmt.Println("Leaving * here is usually fine")
			man.Requirements.Fabric = stringPrompt(&promptui.Prompt{
				Label:   "Minimum Fabric version",
				Default: "*",
				// TODO: validation
			})
			man.Requirements.Minecraft = stringPrompt(&promptui.Prompt{
				Label:   "Supported Minecraft version",
				Default: "1.16",
				// TODO: validation
			})
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
		default:
			man.Requirements.Minecraft = stringPrompt(&promptui.Prompt{
				Label:   "Supported Minecraft version",
				Default: "1.14",
				// TODO: validation
			})
		}

		// generate hooks section for mods
		if man.Package.Type == manifest.TypeMod && hasGradle {
			useHook := boolPrompt(&promptui.Prompt{
				Label:     "Do you want to use \"./gradlew build\" as your build command",
				Default:   "Y",
				IsConfirm: true,
			})
			if useHook == true {
				man.Dev.BuildCommand = "./gradlew build"
			}
		}

		// generate toml
		writeManifest(&man)
		logger.Info(" ✓ Created minepkg.toml")
	},
}

func checkGradle() bool {
	files, err := ioutil.ReadDir("./")
	if err != nil {
		logger.Fail(err.Error())
	}
	for _, f := range files {
		if f.Name() == "gradlew" {
			return true
		}
	}
	return false
}

func writeManifest(man *manifest.Manifest) {
	// generate toml
	buf := bytes.Buffer{}
	if err := toml.NewEncoder(&buf).Encode(man); err != nil {
		logger.Fail(err.Error())
	}
	if err := ioutil.WriteFile("minepkg.toml", buf.Bytes(), 0755); err != nil {
		logger.Fail(err.Error())
	}
}

func selectPrompt(prompt *promptui.Select) string {
	_, res, err := prompt.Run()
	if err != nil {
		fmt.Println("Aborting")
		os.Exit(1)
	}
	return res
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
