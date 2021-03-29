package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/fiws/minepkg/internals/commands"
	"github.com/fiws/minepkg/internals/instances"
	"github.com/jwalton/gchalk"
	"github.com/magiconair/properties"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	runner := &bumpRunner{}
	cmd := commands.New(&cobra.Command{
		Use:   "bump <major|minor|patch|version-number>",
		Args:  cobra.MaximumNArgs(1),
		Short: "Bumps the version number of this package",
	}, runner)

	cmd.Flags().BoolVar(&runner.noTag, "no-tag", false, "Do not create a git tag")
	cmd.Flags().BoolVar(&runner.noGit, "no-git", false, "Skips git checks & tag creation")
	cmd.Flags().BoolVar(&runner.noPush, "no-push", false, "Skips git push")

	rootCmd.AddCommand(cmd.Command)
}

type bumpRunner struct {
	noGit  bool
	noTag  bool
	noPush bool
}

var remoteGitHubSSH = regexp.MustCompile(`^git@github.com:(.+)\.git`)

func (i *bumpRunner) RunE(cmd *cobra.Command, args []string) error {

	instance, err := instances.NewInstanceFromWd()

	if err != nil {
		return fmt.Errorf("invalid package: %w", err)
	}
	currentVersion, err := semver.NewVersion(instance.Manifest.Package.Version)
	if err != nil {
		return fmt.Errorf("package.version in minepkg.toml file not a valid semver (https://semver.org/) version")
	}
	targetVersion := ""

	var userInput string
	if len(args) == 0 {
		if viper.GetBool("nonInteractive") {
			return errors.New("you need to pass a version in non interactive mode")
		}
		prompt := &promptui.Select{
			Label: "Bump",
			Items: []string{
				fmt.Sprintf("patch %s → %s", gchalk.Dim(currentVersion.String()), gchalk.Bold(currentVersion.IncPatch().String())),
				fmt.Sprintf("minor %s → %s", gchalk.Dim(currentVersion.String()), gchalk.Bold(currentVersion.IncMinor().String())),
				fmt.Sprintf("major %s → %s", gchalk.Dim(currentVersion.String()), gchalk.Bold(currentVersion.IncMajor().String())),
				"custom version",
			},
			CursorPos: 0,
		}
		sel, _, err := prompt.Run()
		if err != nil {
			fmt.Println("Aborting")
			os.Exit(1)
		}

		switch sel {
		case 0:
			userInput = "patch"
		case 1:
			userInput = "minor"
		case 2:
			userInput = "major"
		case 3:
			userInput = stringPrompt(&promptui.Prompt{
				Label:     "New Version",
				Default:   currentVersion.String(),
				AllowEdit: true,
				Validate: func(s string) error {
					switch {
					case s == "":
						return nil
					case s == currentVersion.String():
						return errors.New("can not be the current version")
					}

					if _, err := semver.NewVersion(s); err != nil {
						return errors.New("not a valid semver version (major.minor.patch)")
					}

					return nil
				},
			})
		}
	} else {
		userInput = args[0]
	}

	fmt.Println("Basic checks")
	// this was checked before prompt … but it looks better down here
	fmt.Printf(" ✓ minepkg.toml exists and has valid version (%s)\n", currentVersion.String())

	switch userInput {
	case "patch":
		targetVersion = currentVersion.IncPatch().String()
	case "minor":
		targetVersion = currentVersion.IncMinor().String()
	case "major":
		targetVersion = currentVersion.IncMajor().String()
	default:
		v, err := semver.NewVersion(strings.TrimPrefix(userInput, "v"))
		if err != nil {
			return fmt.Errorf("given version must be a valid semver version. https://semver.org/")
		}
		targetVersion = v.String()
	}

	targetTag := "v" + targetVersion

	fmt.Printf(" ✓ Target version %s is valid\n", targetVersion)

	var upstreamPair []string

	fmt.Println("\nGit checks")
	if !i.noGit && isGit() {
		dirty, err := simpleGitExec("status --porcelain")
		if err != nil {
			return err
		}
		if dirty != "" {
			return fmt.Errorf("uncommitted files in git directory. Please commit them first")
		}

		fmt.Println(" ✓ Directory is not dirty")

		_, err = simpleGitExec("rev-parse --verify --quiet " + targetTag)
		if err == nil {
			return fmt.Errorf("git tag %s already exists", targetTag)
		}

		fmt.Println(" ✓ Git tag does not already exist")

		upstream, err := simpleGitExec("rev-parse --symbolic-full-name --abbrev-ref @{upstream}")
		if err != nil {
			return err
		}
		upstreamPair = strings.Split(upstream, "/")
		if len(upstreamPair) != 2 {
			return fmt.Errorf("invalid upstream git output. please report this")
		}

		// fetch from remote
		if _, err = simpleGitExec("fetch --no-tags --quiet --recurse-submodules=no -v " + strings.Join(upstreamPair, " ")); err != nil {
			return err
		}

		fmt.Println(" ✓ Valid upstream")

		upstreamCommitsStr, err := simpleGitExec("rev-list --count HEAD..HEAD@{upstream}")
		if err != nil {
			return err
		}
		upstreamCommits, err := strconv.Atoi(upstreamCommitsStr)
		if err != nil {
			return fmt.Errorf("invalid git output. please report this error: %w", err)
		}
		if upstreamCommits != 0 {
			return fmt.Errorf("there are %d unsynced commits upstream! Please run something like \"git pull --rebase\" first", upstreamCommits)
		}

		fmt.Println(" ✓ No missing commits from upstream")
	} else {
		fmt.Println("  Not in git directory. Skipping checks")
	}

	fmt.Println("\nBumping version to: " + gchalk.Bold(targetVersion))

	// bump the manifest version
	fmt.Println(" updating minepkg.toml")
	instance.Manifest.Package.Version = targetVersion
	if err := instance.SaveManifest(); err != nil {
		return err
	}

	// bump the gradle.properties files
	fmt.Println(" updating gradle.properties")
	props, err := properties.LoadFile("./gradle.properties", properties.UTF8)
	if err != nil {
		return nil
	}

	props.Set("mod_version", targetVersion)
	f, err := os.Create("./gradle.properties")
	if err != nil {
		return err
	}
	props.WriteComment(f, "# ", properties.UTF8)

	if !i.noGit {
		// commit changes
		fmt.Println("► commiting changes")
		_, err = simpleGitExec("commit -am " + targetVersion)
		if err != nil {
			return err
		}

		if !i.noTag {
			fmt.Println("► creating tag")
			_, err = simpleGitExec("tag v" + targetVersion + " -m " + targetTag)
			if err != nil {
				return err
			}
		}

		if !i.noPush {
			fmt.Println("► pushing commits")
			_, err = simpleGitExec("push")
			if err != nil {
				return err
			}
			fmt.Println("► pushing tag")
			_, err = simpleGitExec("push " + upstreamPair[0] + " " + targetTag)
			if err != nil {
				return err
			}
		}

		origin, err := simpleGitExec("config --get remote.origin.url")
		if err != nil {
			return err
		}
		match := remoteGitHubSSH.FindStringSubmatch(origin)
		if len(match) != 1 {
			fmt.Println(gchalk.Bold("\nYou should now create a new release here:"))
			v := url.Values{}
			v.Add("tag", targetTag)
			v.Add("title", targetVersion)
			fmt.Printf("  https://github.com/%s/releases/new?%s\n", match[1], v.Encode())
		}
	}

	return nil
}

func isGit() bool {
	_, err := os.Stat(".git")
	return err == nil
}

var lineMatch = regexp.MustCompile("(.*)\r?\n?$")

func simpleGitExec(args string) (string, error) {
	splitArgs := strings.Split(args, " ")
	cmd := exec.Command("git", splitArgs...)
	out, err := cmd.Output()
	cleanOut := lineMatch.FindStringSubmatch(string(out))
	return cleanOut[1], err
}
