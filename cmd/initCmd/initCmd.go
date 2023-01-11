package initCmd

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/manifoldco/promptui"
	"github.com/minepkg/minepkg/internals/cmdlog"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/internals/utils"
	"github.com/minepkg/minepkg/pkg/manifest"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var logger = cmdlog.DefaultLogger

var projectName = regexp.MustCompile(`^([a-z0-9]|[a-z0-9][a-z0-9-]*[a-z0-9])$`)

func New() *cobra.Command {
	runner := &initRunner{}
	cmd := commands.New(&cobra.Command{
		Use:   "init [name]",
		Short: "Creates a new mod or modpack in the current directory",
		Args:  cobra.MaximumNArgs(1),
	}, runner)

	cmd.Flags().BoolVarP(&runner.force, "force", "f", false, "Overwrite the minepkg.toml if one exists")
	cmd.Flags().BoolVarP(&runner.yes, "yes", "y", false, "Choose defaults for all questions. (same as --non-interactive)")

	return cmd.Command
}

type initRunner struct {
	force bool
	yes   bool
}

func (i *initRunner) RunE(cmd *cobra.Command, args []string) error {
	if _, err := ioutil.ReadFile("./minepkg.toml"); err == nil && !i.force {
		return fmt.Errorf("this directory already contains a minepkg.toml. Use --force to overwrite it")
	}

	man := defaultManifest()

	if i.yes || viper.GetBool("nonInteractive") {
		if man.Package.Type == "mod" {
			if err := i.modFinalization(man); err != nil {
				return err
			}
		}
		// write toml with defaults
		writeManifest(man)
		logger.Info(" ✓ Created minepkg.toml")
		return nil
	}

	logger.Info("[package]")
	cursorPos := 0
	if man.Package.Type == "mod" {
		cursorPos = 1
	}
	man.Package.Type = utils.SelectPrompt(&promptui.Select{
		Label:     "Type",
		Items:     []string{"modpack", "mod"},
		CursorPos: cursorPos,
		// Default: man.Package.Type,
	})

	cursorPos = 0
	if man.Package.Type == "forge" {
		cursorPos = 1
	}
	man.Package.Platform = utils.SelectPrompt(&promptui.Select{
		Label:     "Platform",
		Items:     []string{"fabric", "forge"},
		CursorPos: cursorPos,
	})

	man.Package.Name = utils.StringPrompt(&promptui.Prompt{
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
		AllowEdit: true,
	})

	man.Package.Description = utils.StringPrompt(&promptui.Prompt{
		Label:     "Description",
		Default:   man.Package.Description,
		AllowEdit: true,
	})

	// TODO: maybe check local "LICENSE" file for popular licenses
	man.Package.License = utils.StringPrompt(&promptui.Prompt{
		Label:     "License",
		Default:   man.Package.License,
		AllowEdit: true,
	})

	man.Package.Author = utils.StringPrompt(&promptui.Prompt{
		Label:     "Author",
		Default:   man.Package.Author,
		AllowEdit: true,
	})

	man.Package.Source = utils.StringPrompt(&promptui.Prompt{
		Label:     "Source",
		Default:   man.Package.Source,
		AllowEdit: true,
	})

	man.Package.Version = utils.StringPrompt(&promptui.Prompt{
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

	fmt.Printf("\n")
	logger.Info("[requirements]")

	switch man.Package.Platform {
	case "fabric":
		// fmt.Println("Leaving * here is usually fine")
		man.Requirements.FabricLoader = utils.StringPrompt(&promptui.Prompt{
			Label:     "Supported Fabric version",
			Default:   man.Requirements.FabricLoader,
			AllowEdit: true,
			// TODO: validation
		})
		man.Requirements.Minecraft = utils.StringPrompt(&promptui.Prompt{
			Label:     "Supported Minecraft version",
			Default:   man.Requirements.Minecraft,
			AllowEdit: true,
			// TODO: validation
		})
	case "forge":
		man.Requirements.FabricLoader = ""
		man.Requirements.ForgeLoader = utils.StringPrompt(&promptui.Prompt{
			Label:     "Supported Forge version",
			Default:   "*",
			AllowEdit: true,
			// TODO: validation
		})

		man.Requirements.Minecraft = utils.StringPrompt(&promptui.Prompt{
			Label:     "Supported Minecraft version",
			Default:   "~1.16.2",
			AllowEdit: true,
			// TODO: validation
		})
	default:
		man.Requirements.FabricLoader = ""
		man.Requirements.ForgeLoader = ""
		man.Requirements.Minecraft = utils.StringPrompt(&promptui.Prompt{
			Label:     "Supported Minecraft version",
			Default:   "~1.16.2",
			AllowEdit: true,
			// TODO: validation
		})
	}

	// sets dev.buildCommand
	if man.Package.Type == manifest.TypeMod {
		if err := i.modFinalization(man); err != nil {
			return err
		}
	}

	empty, err := IsEmpty(".")
	if err != nil {
		return err
	}
	// asks for template && creates license
	if empty {
		i.emptyDirFinalization(man)
	}

	// generate toml
	writeManifest(man)
	logger.Info(" ✓ Created minepkg.toml")

	if man.Package.Type == manifest.TypeModpack {
		os.MkdirAll("overwrites/config", os.ModePerm)
	}

	return nil
}

func (i *initRunner) modFinalization(man *manifest.Manifest) error {
	// check this folder for gradle
	if _, err := os.Stat("./gradlew"); os.IsNotExist(err) {
		// no? check the folder above
		if _, err := os.Stat("../gradlew"); !os.IsNotExist(err) {
			man.Dev.BuildCommand = "../gradlew build"
		}
	} else {
		man.Dev.BuildCommand = "./gradlew build"
	}

	// add test-mansion as dev dependency to ease testing
	// maybe ask for this?
	man.AddDevDependency("test-mansion", "*")

	return nil
}

func IsEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

var fallbackVersion = "0.1.0"

func writeManifest(man *manifest.Manifest) {
	if err := ioutil.WriteFile("minepkg.toml", man.Buffer().Bytes(), 0755); err != nil {
		logger.Fail(err.Error())
	}
}
