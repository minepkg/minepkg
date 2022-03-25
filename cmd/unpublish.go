package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/erikgeiser/promptkit/confirmation"
	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/internals/globals"
	"github.com/minepkg/minepkg/internals/instances"
	"github.com/minepkg/minepkg/pkg/manifest"
	"github.com/spf13/cobra"
)

func init() {
	runner := &unpublishRunner{}
	cmd := commands.New(&cobra.Command{
		Use:   "unpublish",
		Short: "Deletes a release from minepkg.io",
		Args:  cobra.MaximumNArgs(1),
	}, runner)

	rootCmd.AddCommand(cmd.Command)
}

type unpublishRunner struct {
	release *api.Release
}

func (p *unpublishRunner) RunE(cmd *cobra.Command, args []string) error {
	apiClient := globals.ApiClient
	// nonInteractive := viper.GetBool("nonInteractive")

	instance, err := instances.NewFromWd()
	if err != nil {
		return err
	}

	m := instance.Manifest

	logger.Log("Validating minepkg.toml")
	problems := m.Validate()
	fatal := false
	for _, problem := range problems {
		if problem.Level == manifest.ErrorLevelFatal {
			fmt.Printf(
				"%s ERROR: %s\n",
				commands.Emoji("❌"),
				problem.Error(),
			)
			fatal = true
		} else {
			fmt.Printf(
				"%s WARNING: %s\n",
				commands.Emoji("⚠️"),
				problem.Error(),
			)
		}
	}
	if fatal {
		return errors.New("validation of minepkg.toml failed")
	}

	if !globals.ApiClient.HasCredentials() {
		logger.Warn("You need to login to minepkg.io first")
		runner := &mpkgLoginRunner{}
		runner.RunE(cmd, args)
	}

	logger.Log("Checking access rights")
	timeout, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()
	_, err = apiClient.GetProject(timeout, m.Package.Name)
	if err != nil {
		return err
	}

	// check if version exists
	logger.Log("Checking if release exists: ")
	p.release, err = apiClient.GetRelease(context.TODO(), m.PlatformString(), m.Package.Name+"@"+m.Package.Version)
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
		logger.Fail("Release is older than 2 days and can not be deleted anymore.")
	}
	logger.Info(fmt.Sprintf("Release %s@%s can be deleted.", m.Package.Name, m.Package.Version))
	logger.Warn("People already using this release might get problems. Publishing a new version is less destructive.")

	// ask for confirmation
	input := confirmation.New("Delete the existing release anyways?", confirmation.No)
	overwrite, err := input.RunPrompt()
	if !overwrite || err != nil {
		logger.Info("Aborting")
		os.Exit(0)
	}

	logger.Info("Deleting release")
	timeout, cancel = context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()
	_, err = apiClient.DeleteRelease(timeout, m.PlatformString(), m.Package.Name+"@"+m.Package.Version)

	return err
}
