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
		enabledText:  "git commit changes",
		disabledText: "skip commiting",
		successText:  "git commit done",
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
		enabledText:  "create git tag " + b.targetTag,
		disabledText: "skip creating a git tag",
		successText:  "git tag created",
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
		enabledText:  "git push to",
		disabledText: "skip git push",
		successText:  "git push ok",
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

	action.enabledText = fmt.Sprintf("git push to %s", b.upstreamPair[0])

	return action
}

func (b *bumpRunner) gradleAction() *action {
	action := &action{
		enabled:      true,
		reason:       "",
		enabledText:  "update gradle.properties",
		disabledText: "skip updating gradle.properties",
		successText:  "updated gradle.properties",
	}

	props, err := gradleCheck()
	if err != nil {
		action.enabled = false
		action.reason = err.Error()
		return action
	}

	action.enabledText = fmt.Sprintf(
		"update mod_version field in gradle.properties: %s → %s",
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
