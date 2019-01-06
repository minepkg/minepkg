package cmd

import (
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
)

func init() {
	initCmd.Flags().BoolVarP(&dry, "force", "f", false, "Overwrite all the things")
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Creates a new mod or modpack in the current directory. Can also generate a minepkg.toml for existing directories.",
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := ioutil.ReadFile("./minepkg.toml"); err == nil && force != true {
			logger.Fail("This directory already contains a minepkg.toml. Use --force to overwrite it")
		}

		man := manifest.Manifest{}

		var (
			emptyDir bool
			gitRepo bool
		)

		_ = emptyDir

		if _, err := git.PlainOpen("./"); err == nil {
			gitRepo = true
		}

		wd, _ := os.Getwd()
		namePrompt := promptui.Prompt{
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
		}

		if name, err := namePrompt.Run(); err != nil {
			stahp()
		} else {
			man.Package.Name = name
		}

		// use git tags is the default for git repos
		vDefault := "N"
		if gitRepo {
			vDefault = "Y"
		}
		gitVersionPrompt := promptui.Prompt{
			Label: "Use git tags for versioning",
			Default: vDefault,
			IsConfirm: true,
		}
		_, err := gitVersionPrompt.Run()
		if err != nil && err.Error() == "^C" {
			stahp()
		}

		if err != nil {
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

		// generate toml
		buf := bytes.Buffer{}
		if err := toml.NewEncoder(&buf).Encode(man); err != nil {
			logger.Fail(err.Error())
		}
		if err := ioutil.WriteFile("minepkg.toml", buf.Bytes(), 755); err != nil {
			logger.Fail(err.Error())
		}
		logger.Info(" ✓ Created minepkg.toml")
	},
}

func stahp() {
	fmt.Println("Aborting")
	os.Exit(1)
}

