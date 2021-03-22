package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/fiws/minepkg/internals/commands"
	"github.com/fiws/minepkg/internals/instances"
	"github.com/magiconair/properties"
	"github.com/spf13/cobra"
)

func init() {
	runner := &bumpRunner{}
	cmd := commands.New(&cobra.Command{
		Use:   "bump <major|minor|patch|version-number>",
		Args:  cobra.ExactArgs(1),
		Short: "Bumps the version number of this package",
	}, runner)

	cmd.Flags().BoolVar(&runner.noTag, "no-tag", false, "Do not create a git tag")
	cmd.Flags().BoolVar(&runner.noGit, "no-git", false, "Skips git checks & tag creation")

	rootCmd.AddCommand(cmd.Command)
}

type bumpRunner struct {
	noTag bool
	noGit bool
}

func (i *bumpRunner) RunE(cmd *cobra.Command, args []string) error {

	instance, err := instances.NewInstanceFromWd()
	fmt.Println("Basic checks")
	if err != nil {
		return fmt.Errorf("invalid package: %w", err)
	}
	currentVersion, err := semver.NewVersion(instance.Manifest.Package.Version)
	if err != nil {
		return fmt.Errorf("package.version in minepkg.toml file not a valid semver (https://semver.org/) version")
	}
	targetVersion := ""

	fmt.Printf("✓ minepkg.toml exists and has valid version (%s)\n", currentVersion.String())

	switch args[0] {
	case "patch":
		targetVersion = currentVersion.IncPatch().String()
	case "minor":
		targetVersion = currentVersion.IncMinor().String()
	case "major":
		targetVersion = currentVersion.IncMajor().String()
	default:
		v, err := semver.NewVersion(strings.TrimPrefix(args[0], "v"))
		if err != nil {
			return fmt.Errorf("given version must be a valid semver version. https://semver.org/")
		}
		targetVersion = v.String()
	}

	fmt.Printf("✓ Target version %s is valid\n", targetVersion)

	if !i.noGit && isGit() {
		fmt.Println("\nGit checks")
		dirty, err := simpleGitExec("status --porcelain")
		if err != nil {
			return err
		}
		if dirty != "" {
			return fmt.Errorf("uncommited files in git directory. Please commit them first")
		}

		fmt.Println("✓ Directory is not dirty")

		_, err = simpleGitExec("rev-parse --verify --quiet v" + targetVersion)
		if err == nil {
			return fmt.Errorf("git tag v%s already exists", targetVersion)
		}

		fmt.Println("✓ Git tag does not already exist")

		upstream, err := simpleGitExec("rev-parse --symbolic-full-name --abbrev-ref @{upstream}")
		if err != nil {
			return err
		}
		targetPair := strings.Split(upstream, "/")
		if len(targetPair) != 2 {
			return fmt.Errorf("invalid upstream git output. please report this")
		}

		// fetch from remote
		if _, err = simpleGitExec("fetch --no-tags --quiet --recurse-submodules=no -v " + strings.Join(targetPair, " ")); err != nil {
			return err
		}

		fmt.Println("✓ Valid upstream")

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

		fmt.Println("✓ No missing commits from upstream")
	} else {
		fmt.Println("Not in git directory. Skipping checks")
	}

	fmt.Println("\nStarting the bumpening to: " + targetVersion)

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
		fmt.Println(" commiting changes")
		_, err = simpleGitExec("commit -am " + targetVersion)
		if err != nil {
			return err
		}

		if !i.noTag {
			fmt.Println(" creating tag")
			_, err = simpleGitExec("tag v" + targetVersion + " -m v" + targetVersion)
			if err != nil {
				return err
			}
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
