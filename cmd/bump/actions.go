package bump

import (
	"fmt"
	"os"

	"github.com/jwalton/gchalk"
	"github.com/magiconair/properties"
	"github.com/minepkg/minepkg/internals/utils"
)

type action struct {
	enabled      bool
	reason       string
	enabledText  string
	disabledText string
	successText  string
	run          func() error
}

func (a *action) StatusText() string {
	if a.enabled {
		return "+ " + a.enabledText
	}

	disabledText := a.disabledText
	if a.reason != "" {
		disabledText += " (" + a.reason + ")"
	}
	return "- " + gchalk.Gray(disabledText)
}

func (a *action) Run() error {
	if !a.enabled {
		return nil
	}
	if err := a.run(); err != nil {
		return err
	}
	fmt.Println("► " + a.successText)
	return nil
}

func (b *bumpRunner) gitCommitAction() *action {
	return &action{
		enabled:      !b.noGit,
		reason:       "",
		enabledText:  "Create a Git commit called " + b.targetVersion,
		disabledText: "Skip committing",
		successText:  "Git commit created",
		run: func() error {
			_, err := utils.SimpleGitExec("commit -am " + b.targetVersion)
			return err
		},
	}
}

func (b *bumpRunner) gitTagAction() *action {
	return &action{
		enabled:      !b.noGit && !b.noTag,
		reason:       "",
		enabledText:  "Create a git tag called " + b.targetTag,
		disabledText: "Skip creating a git tag",
		successText:  "Git tag created",
		run: func() error {
			_, err := utils.SimpleGitExec("tag v" + b.targetVersion + " -m " + b.targetTag)
			return err
		},
	}
}

func (b *bumpRunner) gitPushAction() *action {
	action := &action{
		enabled:      !b.noGit && !b.noPush,
		reason:       "",
		enabledText:  "Push Git commits",
		disabledText: "Skip git push",
		successText:  "Git commits pushed",
		run: func() error {
			_, err := utils.SimpleGitExec("push")
			if err != nil {
				return err
			}
			_, err = utils.SimpleGitExec("push " + b.upstreamPair[0] + " " + b.targetTag)
			if err != nil {
				return err
			}

			return nil
		},
	}

	if len(b.upstreamPair) == 0 {
		action.reason = "no git upstream"
		action.enabled = false
		return action
	}

	action.enabledText = fmt.Sprintf("Git push to %s", b.upstreamPair[0])

	return action
}

func (b *bumpRunner) gradleAction() *action {
	action := &action{
		enabled:      true,
		reason:       "",
		enabledText:  "Update gradle.properties",
		disabledText: "Skip updating gradle.properties",
		successText:  "Updated gradle.properties",
	}

	props, err := gradleCheck()
	if err != nil {
		action.enabled = false
		action.reason = err.Error()
		return action
	}

	action.enabledText = fmt.Sprintf(
		"Update the \"mod_version\" field in gradle.properties: %s → %s",
		props.GetString("mod_version", ""),
		b.targetVersion,
	)

	// looks good, set run function
	action.run = func() error {
		props.Set("mod_version", b.targetVersion)
		f, err := os.Create("./gradle.properties")
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = props.WriteComment(f, "# ", properties.UTF8)
		return err
	}

	return action
}

func gradleCheck() (*properties.Properties, error) {
	props, err := properties.LoadFile("./gradle.properties", properties.UTF8)
	if err != nil {
		return props, fmt.Errorf("gradle.properties does not exist " + err.Error())
	}

	// write mod_version in gradle.properties if its there
	if props.GetString("mod_version", "") == "" {
		return props, fmt.Errorf("gradle.properties does not have a \"mod_version\" field")
	}

	return props, nil
}
