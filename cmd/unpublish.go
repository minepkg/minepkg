package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/erikgeiser/promptkit/confirmation"
	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/internals/globals"
	"github.com/minepkg/minepkg/internals/instances"
	"github.com/minepkg/minepkg/internals/pkgid"
	"github.com/spf13/cobra"
)

func init() {
	runner := &unpublishRunner{}
	cmd := commands.New(&cobra.Command{
		Use:   "unpublish [version]",
		Short: "Deletes a release from minepkg.io",
		Long: `
"unpublish" deletes a release from minepkg.io.
Uses the version specified in the minepkg.toml file if not set.
		`,
		Example: strings.Join([]string{
			"minepkg unpublish",
			"minepkg unpublish 1.0.0",
			"minepkg unpublish demo-mod@2.1.0",
		}, "\n"),
		Args: cobra.MaximumNArgs(1),
	}, runner)

	rootCmd.AddCommand(cmd.Command)
}

type unpublishRunner struct {
	release *api.Release
}

func (p *unpublishRunner) RunE(cmd *cobra.Command, args []string) error {
	apiClient := globals.ApiClient
	// nonInteractive := viper.GetBool("nonInteractive")

	var mID *pkgid.ID
	var err error

	instance, instanceErr := instances.NewFromWd()

	if len(args) == 0 {
		if instanceErr != nil {
			return err
		}

		mani := instance.Manifest
		// we validate the local manifest
		if err := root.validateManifest(mani); err != nil {
			return err
		}
		mID = pkgid.NewFromManifest(mani)
	} else {
		mID = pkgid.ParseLikeVersion(args[0])

		if mID.Version == "" && instanceErr == nil && instance.Manifest.Package.Version != "" {
			mID.Version = instance.Manifest.Package.Version
		}

		// use manifest platform, but only if name is not set
		// if the user sets the name they probably don't want to inherit the platform from the manifest
		if mID.Platform == "" && mID.Name == "" && instanceErr == nil && instance.Manifest.Package.Platform != "" {
			mID.Platform = instance.Manifest.Package.Platform
		}

		if mID.Name == "" && instanceErr == nil && instance.Manifest.Package.Name != "" {
			mID.Name = instance.Manifest.Package.Name
		}
	}

	// no version passed or found in the manifest
	if mID.Version == "" {
		suggestions := []string{`Use a full ID like this: "minepkg unpublish fabric/my-modpack@0.5.1"`}
		if instanceErr == nil {
			suggestions = append(
				suggestions,
				`Pass the version like this: "minepkg unpublish 2.1.0"`,
				`Make sure the minepkg.toml file has package.version set`,
			)
		} else {
			suggestions = append(suggestions, `Move into a directory with a minepkg.toml file`)
		}
		return &commands.CliError{
			Text:        "You did not specify a version",
			Suggestions: suggestions,
		}
	}

	// no platform passed or found in the manifest
	if mID.Platform == "" {
		suggestions := []string{`Use a full ID like this: "minepkg unpublish fabric/my-modpack@0.5.1"`}
		if instanceErr == nil {
			suggestions = append(suggestions, `Set the "package.platform" field in your minepkg.toml`)
		}
		return &commands.CliError{
			Text:        "You did not specify a platform",
			Suggestions: suggestions,
		}
	}

	if !globals.ApiClient.HasCredentials() {
		logger.Warn("You need to login to minepkg.io first")
		runner := &mpkgLoginRunner{}
		runner.RunE(cmd, args)
	}

	// check if release exists
	timeout, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()
	logger.Log(fmt.Sprintf("Checking if release %s exists", mID.LegacyID()))
	p.release, err = apiClient.GetRelease(
		timeout,
		mID.Platform,
		mID.LegacyID(),
	)
	if err != nil {
		// typo or already unpublished
		if err == api.ErrNotFound {
			logger.Warn("Release not found")
			return nil
		}

		// unexpected error
		return err
	}

	if time.Since(*p.release.Meta.CreatedAt) > time.Hour*24*2 {
		logger.Fail("Release is older than 2 days and can not be deleted. Try publishing a higher version.")
	}
	logger.Info(fmt.Sprintf("Release %s@%s can be deleted.", mID.Name, mID.Version))
	logger.Warn("People already using this release might get problems.")
	logger.Info("Publishing a new version instead often is less destructive.\n")

	// ask for confirmation
	input := confirmation.New("Delete the existing release anyway?", confirmation.No)
	overwrite, err := input.RunPrompt()
	if !overwrite || err != nil {
		logger.Info("Aborting")
		os.Exit(0)
	}

	logger.Info("Deleting release " + mID.LegacyID())
	timeout, cancel = context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()
	_, err = apiClient.DeleteRelease(timeout, mID.Platform, mID.LegacyID())
	if err != nil {
		return err
	}

	logger.Info("Release deleted")
	return nil
}
