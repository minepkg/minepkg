package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/jwalton/gchalk"
	"github.com/magiconair/properties"
	"github.com/manifoldco/promptui"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/internals/fabric"
	"github.com/minepkg/minepkg/internals/license"
	"github.com/minepkg/minepkg/internals/utils"
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

	// rootCmd.AddCommand(cmd.Command)
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
		AllowEdit: true,
	})

	man.Package.Description = stringPrompt(&promptui.Prompt{
		Label:     "Description",
		Default:   man.Package.Description,
		AllowEdit: true,
	})

	// TODO: maybe check local "LICENSE" file for popular licenses
	man.Package.License = stringPrompt(&promptui.Prompt{
		Label:     "License",
		Default:   man.Package.License,
		AllowEdit: true,
	})

	man.Package.Author = stringPrompt(&promptui.Prompt{
		Label:     "Author",
		Default:   man.Package.Author,
		AllowEdit: true,
	})

	man.Package.Source = stringPrompt(&promptui.Prompt{
		Label:     "Source",
		Default:   man.Package.Source,
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

	fmt.Printf("\n")
	logger.Info("[requirements]")

	switch man.Package.Platform {
	case "fabric":
		// fmt.Println("Leaving * here is usually fine")
		man.Requirements.Fabric = stringPrompt(&promptui.Prompt{
			Label:     "Supported Fabric version",
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
			Label:     "Supported Forge version",
			Default:   "*",
			AllowEdit: true,
			// TODO: validation
		})

		man.Requirements.Minecraft = stringPrompt(&promptui.Prompt{
			Label:     "Supported Minecraft version",
			Default:   "~1.16.2",
			AllowEdit: true,
			// TODO: validation
		})
	default:
		man.Requirements.Fabric = ""
		man.Requirements.Forge = ""
		man.Requirements.Minecraft = stringPrompt(&promptui.Prompt{
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
		os.MkdirAll("overwrites/configs", os.ModePerm)
	}

	return nil
}

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

func (i *initRunner) modFinalization(man *manifest.Manifest) error {
	// useHook := boolPrompt(&promptui.Prompt{
	// 	Label:     "Do you want to use gradlew to build",
	// 	Default:   "Y",
	// 	IsConfirm: true,
	// })
	// if useHook {
	// 	man.Dev.BuildCommand = "./gradlew build"
	// } else {
	// 	logger.Warn(`Please set the "dev.buildCommand" field in the minepkg.toml file by hand.`)
	// }

	// add test-mansion as dev dependency to ease testing
	// maybe ask for this?
	man.AddDevDependency("test-mansion", "*")

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

		if len(fabricMan.Authors) != 0 {
			man.Package.Author = fabricMan.Authors[0]
		} else {
			man.Package.Author = getDefaultAuthor()
		}

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
	man.Package.Author = getDefaultAuthor()

	source := viper.GetString("init.defaultSource")
	if source != "" {
		if u, err := url.Parse(source); err == nil {
			u.Path = path.Join(u.Path, man.Package.Name)
			source = u.String()
		}
	}
	man.Package.Source = source

	man.Requirements.Fabric = "*"
	man.Requirements.Minecraft = "~1.16.2"

	return man
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
