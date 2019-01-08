package cmd

import (
	"github.com/fiws/minepkg/pkg/api"
	"context"
	"github.com/BurntSushi/toml"
	"bytes"
	"github.com/fiws/minepkg/pkg/manifest"
	"gopkg.in/src-d/go-git.v4"
	"github.com/Masterminds/semver"
	"path/filepath"
	"fmt"
	"github.com/stoewer/go-strcase"
	"errors"
	"strings"
	"github.com/manifoldco/promptui"
	"regexp"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
)

var projectName = regexp.MustCompile(`^([a-z0-9]|[a-z0-9][a-z0-9-]*[a-z0-9])$`)

var (
	force bool
	loader string
)

func init() {
	initCmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite all the things")
	initCmd.Flags().StringVarP(&loader, "loader", "l", "forge", "Set the required loader to forge or fabric.")
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Creates a new mod or modpack in the current directory. Can also generate a minepkg.toml for existing directories.",
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := ioutil.ReadFile("./minepkg.toml"); err == nil && force != true {
			logger.Fail("This directory already contains a minepkg.toml. Use --force to overwrite it")
		}

		man := manifest.Manifest{}

		loader = strings.ToLower(loader)
		if loader != "forge" && loader != "fabric" {
			logger.Fail("Allowed values for loader option: forge or fabric")
		}
		var (
			emptyDir bool
			gitRepo bool
		)

		_ = emptyDir

		// chVersions := make(chan *instances.MinecraftReleaseResponse)
		// go func(ch chan *instances.MinecraftReleaseResponse)  {
		// 	res, err := instances.GetMinecraftReleases(context.TODO())
		// 	if err != nil {
		// 		logger.Fail(err.Error())
		// 	}
		// 	ch <- res
		// 	}(chVersions)
			
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

		man.Package.Name = stringPrompt(&promptui.Prompt{
			Label: "Project name",
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

		// use git tags is the default for git repos
		vDefault := "N"
		if gitRepo {
			vDefault = "Y"
		}
		
		useGitTags := boolPrompt(&promptui.Prompt {
			Label: "Use git tags for versioning",
			Default: vDefault,
			IsConfirm: true,
		})

		if useGitTags == false {
			versionPrompt := promptui.Prompt{
				Label: "Version",
				Default: "1.0.0",
				Validate: func(s string) error {
					if _, err := semver.NewVersion(s); err != nil {
						return errors.New("Not a valid semver version (major.minor.patch)")
					}
					if strings.HasPrefix(s, "v") {
						return errors.New("Please do not include v as a prefix")
					}
					return nil
				},
			}
			if version, err := versionPrompt.Run(); err != nil {
				stahp()
			} else {
				man.Package.Version = version
			}
		}
		
		// res := <- chVersions

		modType := stringPrompt(&promptui.Prompt{
			Label: "Is this a Forge or a Fabric mod?",
			Default: "Forge",
			// TODO: validation
		})
		modType = strings.ToLower(modType)

		if loader == "forge" {
			forgeReleases := <- chForgeVersions
			man.Requirements.Forge = stringPrompt(&promptui.Prompt{
				Label: "Minimum Forge version",
				Default: forgeReleases.Versions[0].Version,
				// TODO: validation
			})

			man.Requirements.Minecraft = stringPrompt(&promptui.Prompt{
				Label: "Supported Minecraft version",
				Default: forgeReleases.Versions[0].McVersion,
				// TODO: validation
			})
		} else {
			man.Requirements.Forge = stringPrompt(&promptui.Prompt{
				Label: "Minimum Fabric version",
				// TODO: validation
			})
			man.Requirements.Minecraft = stringPrompt(&promptui.Prompt{
				Label: "Supported Minecraft version",
				Default: "1.14.x",
				// TODO: validation
			})
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

func stahp() {
	fmt.Println("Aborting")
	os.Exit(1)
}

