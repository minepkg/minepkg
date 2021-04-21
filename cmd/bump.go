package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/jwalton/gchalk"
	"github.com/magiconair/properties"
	"github.com/manifoldco/promptui"
	"github.com/mattn/go-isatty"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/internals/instances"
	"github.com/minepkg/minepkg/internals/utils"
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

	targetVersion string
	targetTag     string
	upstreamPair  []string
}

func (b *bumpRunner) RunE(cmd *cobra.Command, args []string) error {

	instance, err := instances.NewInstanceFromWd()

	if err != nil {
		return err
	}
	currentVersion, err := semver.NewVersion(instance.Manifest.Package.Version)
	if err != nil {
		return fmt.Errorf("package.version in minepkg.toml file not a valid semver (https://semver.org/) version")
	}
	targetVersion := ""

	var userInput string
	if len(args) == 0 {
		userInput, err = b.interactiveVersionInput(currentVersion)
		if err != nil {
			return err
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

	b.targetVersion = targetVersion
	b.targetTag = "v" + targetVersion

	fmt.Printf(" ✓ Target version %s is valid\n", targetVersion)

	fmt.Println("\nGit checks")
	if !b.noGit && isGit() {
		// run git checks (this also sets b.upstreamPair)
		if err := b.gitChecks(); err != nil {
			return err
		}
	} else {
		fmt.Println("  Not in git directory. Skipping checks")
	}

	fmt.Println("\nBumping version to: " + gchalk.Bold(targetVersion))

	// bump the manifest version
	fmt.Println("► updating minepkg.toml")
	instance.Manifest.Package.Version = targetVersion
	if err := instance.SaveManifest(); err != nil {
		return err
	}

	// bump the gradle.properties files
	fmt.Println("► updating gradle.properties")
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

	if !b.noGit {
		if err := b.gitActions(); err != nil {
			return err
		}
	}

	fmt.Println("\n" + commands.Emoji("✅ ") +
		"Bump complete. You can now publish with " +
		gchalk.Bold("minepkg publish"))

	return nil
}

func (b *bumpRunner) interactiveVersionInput(currentVersion *semver.Version) (string, error) {
	if viper.GetBool("nonInteractive") {
		return "", errors.New("you need to pass a version in non interactive mode")
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

	userInput := ""

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

	return userInput, nil
}

func (b *bumpRunner) gitChecks() error {
	dirty, err := utils.SimpleGitExec("status --porcelain")
	if err != nil {
		return err
	}
	if dirty != "" {
		return fmt.Errorf("uncommitted files in git directory. Please commit them first")
	}

	fmt.Println(" ✓ Directory is not dirty")

	_, err = utils.SimpleGitExec("rev-parse --verify --quiet " + b.targetTag)
	if err == nil {
		return fmt.Errorf("git tag %s already exists", b.targetTag)
	}

	fmt.Println(" ✓ Git tag does not already exist")

	upstream, err := utils.SimpleGitExec("rev-parse --symbolic-full-name --abbrev-ref @{upstream}")
	if err != nil {
		return err
	}
	upstreamPair := strings.Split(upstream, "/")
	if len(upstreamPair) != 2 {
		return fmt.Errorf("invalid upstream git output. please report this")
	}

	// fetch from remote
	if _, err = utils.SimpleGitExec("fetch --no-tags --quiet --recurse-submodules=no -v " + strings.Join(upstreamPair, " ")); err != nil {
		return err
	}

	fmt.Println(" ✓ Valid upstream")

	upstreamCommitsStr, err := utils.SimpleGitExec("rev-list --count HEAD..HEAD@{upstream}")
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
	b.upstreamPair = upstreamPair

	return nil
}

func (b *bumpRunner) gitActions() error {
	var err error
	// commit changes
	fmt.Println("► commiting changes")
	_, err = utils.SimpleGitExec("commit -am " + b.targetVersion)
	if err != nil {
		return err
	}

	if !b.noTag {
		fmt.Println("► creating tag")
		_, err = utils.SimpleGitExec("tag v" + b.targetVersion + " -m " + b.targetTag)
		if err != nil {
			return err
		}
	}

	if !b.noPush {
		fmt.Println("► pushing commits")
		_, err = utils.SimpleGitExec("push")
		if err != nil {
			return err
		}
		fmt.Println("► pushing tag")
		_, err = utils.SimpleGitExec("push " + b.upstreamPair[0] + " " + b.targetTag)
		if err != nil {
			return err
		}
	}

	if !b.noTag {
		return b.gitCreateReleasePrompt()
	}

	return nil
}

var remoteGitHubSSH = regexp.MustCompile(`^git@github.com:(.+)\.git`)
var remoteGitHubHttps = regexp.MustCompile(`https://github.com/(.+)\.git`)

func (b *bumpRunner) gitCreateReleasePrompt() error {
	origin, err := utils.SimpleGitExec("config --get remote.origin.url")
	if err != nil {
		return err
	}

	match := remoteGitHubSSH.FindStringSubmatch(origin)
	if len(match) != 2 {
		match = remoteGitHubHttps.FindStringSubmatch(origin)
	}
	if len(match) == 2 {
		v := url.Values{}
		v.Add("tag", b.targetTag)
		v.Add("title", b.targetVersion)
		url := fmt.Sprintf("https://github.com/%s/releases/new?%s", match[1], v.Encode())

		if isInteractive() {
			openBrowser := boolPrompt(&promptui.Prompt{
				Label:     "Open browser to create GitHub release now (recommended)",
				Default:   "Y",
				IsConfirm: true,
			})
			if openBrowser {
				utils.OpenBrowser(url)
			} else {
				fmt.Println("Ok not opening. You can still create the release here if you want:")
				fmt.Println("  " + url)
			}
		} else {
			fmt.Println(gchalk.Bold("\nYou can now create a new GitHub release here:"))
			fmt.Println("  " + url)
		}

	}
	return nil
}

func isGit() bool {
	_, err := os.Stat(".git")
	return err == nil
}

func isInteractive() bool {
	return !viper.GetBool("nonInteractive") &&
		(isatty.IsTerminal(os.Stdout.Fd()) ||
			isatty.IsCygwinTerminal(os.Stdout.Fd()))
}
