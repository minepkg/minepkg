package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/magiconair/properties"
	"github.com/manifoldco/promptui"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/internals/fabric"
	"github.com/minepkg/minepkg/pkg/manifest"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stoewer/go-strcase"
)

var projectName = regexp.MustCompile(`^([a-z0-9]|[a-z0-9][a-z0-9-]*[a-z0-9])$`)

func init() {
	runner := &initRunner{}
	cmd := commands.New(&cobra.Command{
		Use:   "init [name]",
		Short: "Creates a new mod or modpack in the current directory",
		Args:  cobra.MaximumNArgs(1),
	}, runner)

	cmd.Flags().BoolVarP(&runner.force, "force", "f", false, "Overwrite the minepkg.toml if one exists")
	cmd.Flags().BoolVarP(&runner.yes, "yes", "y", false, "Choose defaults for all questions. (same as --non-interactive)")

	rootCmd.AddCommand(cmd.Command)
}

type initRunner struct {
	force bool
	yes   bool
}

func (i *initRunner) RunE(cmd *cobra.Command, args []string) error {
	if _, err := ioutil.ReadFile("./minepkg.toml"); err == nil && !i.force {
		logger.Fail("This directory already contains a minepkg.toml. Use --force to overwrite it")
	}

	man := defaultManifest()

	if i.yes || viper.GetBool("nonInteractive") {
		// generate toml with defaults
		writeManifest(man)
		logger.Info(" ✓ Created minepkg.toml")
		return nil
	}

	logger.Info("[package]")
	cursorPos := 0
	if man.Package.Type == "mod" {
		cursorPos = 1
	}
	man.Package.Type = selectPrompt(&promptui.Select{
		Label:     "Type",
		Items:     []string{"modpack", "mod"},
		CursorPos: cursorPos,
		// Default: man.Package.Type,
	})

	cursorPos = 0
	if man.Package.Type == "forge" {
		cursorPos = 1
	}
	man.Package.Platform = selectPrompt(&promptui.Select{
		Label:     "Platform",
		Items:     []string{"fabric", "forge"},
		CursorPos: cursorPos,
	})

	man.Package.Name = stringPrompt(&promptui.Prompt{
		Label:   "Name",
		Default: man.Package.Name,
		Validate: func(s string) error {
			switch {
			case strings.ToLower(s) != s:
				return errors.New("may only contain lowercase characters")
			case strings.HasPrefix(s, "-"):
				return errors.New("may not start with a –")
			case strings.HasSuffix(s, "-"):
				return errors.New("may not end with a –")
			case !projectName.MatchString(s):
				return errors.New("may only contain alphanumeric characters and dashes -")
			}
			return nil
		},
	})

	man.Package.Description = stringPrompt(&promptui.Prompt{
		Label:     "Description",
		Default:   man.Package.Description,
		AllowEdit: true,
	})

	// TODO: maybe check local "LICENCE" file for popular licences
	man.Package.License = stringPrompt(&promptui.Prompt{
		Label:     "License",
		Default:   man.Package.License,
		AllowEdit: true,
	})

	man.Package.Version = stringPrompt(&promptui.Prompt{
		Label:     "Version",
		Default:   man.Package.Version,
		AllowEdit: true,
		Validate: func(s string) error {
			switch {
			case s == "":
				return nil
			case strings.HasPrefix(s, "v"):
				return errors.New("please do not include v as a prefix")
			}

			if _, err := semver.NewVersion(s); err != nil {
				return errors.New("not a valid semver version (major.minor.patch)")
			}

			return nil
		},
	})

	fmt.Println("")
	logger.Info("[requirements]")

	switch man.Package.Platform {
	case "fabric":
		fmt.Println("Leaving * here is usually fine")
		man.Requirements.Fabric = stringPrompt(&promptui.Prompt{
			Label:     "Minimum Fabric version",
			Default:   man.Requirements.Fabric,
			AllowEdit: true,
			// TODO: validation
		})
		man.Requirements.Minecraft = stringPrompt(&promptui.Prompt{
			Label:     "Supported Minecraft version",
			Default:   man.Requirements.Minecraft,
			AllowEdit: true,
			// TODO: validation
		})
	case "forge":
		man.Requirements.Fabric = ""
		man.Requirements.Forge = stringPrompt(&promptui.Prompt{
			Label:     "Minimum Forge version",
			Default:   "*",
			AllowEdit: true,
			// TODO: validation
		})

		man.Requirements.Minecraft = stringPrompt(&promptui.Prompt{
			Label:     "Supported Minecraft version",
			Default:   "1.16",
			AllowEdit: true,
			// TODO: validation
		})
	default:
		man.Requirements.Fabric = ""
		man.Requirements.Forge = ""
		man.Requirements.Minecraft = stringPrompt(&promptui.Prompt{
			Label:     "Supported Minecraft version",
			Default:   "1.16",
			AllowEdit: true,
			// TODO: validation
		})
	}

	// generate hooks section for mods
	if man.Package.Type == manifest.TypeMod {
		useHook := boolPrompt(&promptui.Prompt{
			Label:     "Do you want to use gradlew to build",
			Default:   "Y",
			IsConfirm: true,
		})
		if useHook {
			man.Dev.BuildCommand = "./gradlew build"
		} else {
			logger.Warn(`Please set the "dev.buildCommand" field in the minepkg.toml file by hand.`)
		}

		// add test-mansion as dev dependency to ease testing
		// maybe ask for this?
		man.AddDevDependency("test-mansion", "*")
	}

	// generate toml
	writeManifest(man)
	logger.Info(" ✓ Created minepkg.toml")

	if man.Package.Type == manifest.TypeModpack {
		os.MkdirAll("overwrites/configs", os.ModePerm)
	}

	return nil
}

func defaultManifest() *manifest.Manifest {
	fabricMan := &fabric.Manifest{}
	man := manifest.New()

	err := readJSON("./src/main/resources/fabric.mod.json", fabricMan)
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

		if mcDep, ok := fabricMan.Depends["minecraft"]; ok {
			man.Requirements.Minecraft = mcDep
		}

		if fabricReq, ok := fabricMan.Depends["fabricloader"]; ok {
			man.Requirements.Fabric = fabricReq
		} else {
			man.Requirements.Fabric = "*"
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

	man.Requirements.Fabric = "*"
	man.Requirements.Minecraft = "1.16"

	return man
}

var fallbackVersion = "0.1.0"

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

func writeManifest(man *manifest.Manifest) {
	if err := ioutil.WriteFile("minepkg.toml", man.Buffer().Bytes(), 0755); err != nil {
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
