package bump

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/minepkg/minepkg/internals/utils"
)

func (b *bumpRunner) gitChecks() error {
	dirty, err := utils.SimpleGitExec("status --porcelain")
	if err != nil {
		return err
	}

	if len(dirty) != 0 {
		// are all untracked files?
		allUntracked := true
		for _, line := range dirty {
			if !strings.HasPrefix(line, "??") {
				allUntracked = false
				break
			}
		}

		if allUntracked {
			fmt.Println("You have untracked files in your git directory. Ignoring them for now. Please commit or gitignore them!")
		} else {
			fmt.Println("You have uncommitted changes in your git directory. Please commit or stash them first!")
			fmt.Println("Uncommitted files:")
			for _, line := range dirty {
				fmt.Println(" - " + line[3:])
			}
			return fmt.Errorf("uncommitted files in git directory")
		}
	}

	fmt.Println(" ✓ Directory is not dirty")

	_, err = utils.SimpleGitExec("rev-parse --verify --quiet " + b.targetTag)
	if err == nil {
		return fmt.Errorf("git tag %s already exists", b.targetTag)
	}

	fmt.Println(" ✓ Git tag does not already exist")

	upstream, err := utils.SimpleGitExec("rev-parse --symbolic-full-name --abbrev-ref @{upstream}")
	if err != nil {
		// TODO: check for unexpected errors if possible
		fmt.Println(" ? No git upstream. Assuming offline only git repo and skipping push")
		b.noPush = true
		return nil
	}
	upstreamPair := strings.Split(upstream.Last(), "/")
	if len(upstreamPair) != 2 {
		return fmt.Errorf("invalid upstream git output. please report this")
	}

	// fetch from git remote
	if _, err = utils.SimpleGitExec("fetch --no-tags --quiet --recurse-submodules=no -v " + strings.Join(upstreamPair, " ")); err != nil {
		return err
	}

	fmt.Println(" ✓ Valid upstream")

	upstreamCommitsOutput, err := utils.SimpleGitExec("rev-list --count HEAD..HEAD@{upstream}")
	if err != nil {
		return err
	}
	upstreamCommits, err := strconv.Atoi(upstreamCommitsOutput.Last())
	if err != nil {
		return fmt.Errorf("invalid git output. please report this error: %w", err)
	}
	if upstreamCommits != 0 {
		return fmt.Errorf("there are %d un-synced commits upstream! Please run something like \"git pull --rebase\" first", upstreamCommits)
	}

	fmt.Println(" ✓ No missing commits from upstream")
	b.upstreamPair = upstreamPair

	return nil
}
