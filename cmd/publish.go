package cmd

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/erikgeiser/promptkit/confirmation"
	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/internals/globals"
	"github.com/minepkg/minepkg/internals/instances"
	"github.com/minepkg/minepkg/pkg/manifest"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var semverMatch = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-(0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(\.(0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*)?(\+[0-9a-zA-Z-]+(\.[0-9a-zA-Z-]+)*)?$`)

func init() {
	runner := &publishRunner{}
	cmd := commands.New(&cobra.Command{
		Use:     "publish",
		Short:   "Publishes the local package in the current directory to minepkg.io",
		Aliases: []string{"ship", "push"},
		Args:    cobra.MaximumNArgs(1),
	}, runner)

	cmd.Flags().BoolVarP(&runner.unofficial, "unofficial", "", false, "Indicate that you are only maintaining this package and do not controll the source")
	cmd.Flags().BoolVarP(&runner.dry, "dry", "", false, "Dry run without publishing")
	cmd.Flags().BoolVarP(&runner.noBuild, "no-build", "", false, "Skips building the package")
	cmd.Flags().StringVarP(&runner.versionName, "release", "r", "", "Release version number to publish (overwrites version in manifest)")

	rootCmd.AddCommand(cmd.Command)
}

type publishRunner struct {
	unofficial  bool
	dry         bool
	noBuild     bool
	versionName string
	file        string

	release *api.Release
}

func (p *publishRunner) RunE(cmd *cobra.Command, args []string) error {
	apiClient := globals.ApiClient
	nonInteractive := viper.GetBool("nonInteractive")

	tasks := logger.NewTask(3)
	tasks.Step("📚", "Preparing Publish")

	tasks.Log("Checking minepkg.toml")
	instance, err := instances.NewFromWd()
	if err != nil {
		return err
	}

	m := instance.Manifest
	// we validate the local manifest
	if err := root.validateManifest(m); err != nil {
		return err
	}

	tasks.Log("Checking Authentication")
	if !globals.ApiClient.HasCredentials() {
		logger.Warn("You need to login to minepkg.io first")
		runner := &mpkgLoginRunner{}
		runner.RunE(cmd, args)
	}

	logger.Log("Checking access rights")
	timeout, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()
	project, err := apiClient.GetProject(timeout, m.Package.Name)

	if err == api.ErrNotFound {
		if !nonInteractive {
			createText := "Project " + m.Package.Name + " does not exist. Do you want to create it?"
			if p.unofficial {
				createText += " (as unofficial)"
			}
			input := confirmation.New(createText, confirmation.Yes)
			create, err := input.RunPrompt()
			if !create || err != nil {
				logger.Info("Aborting")
				os.Exit(0)
			}
		}
		readme, _ := getReadme()
		project, err = apiClient.CreateProject(&api.Project{
			Name:        m.Package.Name,
			Type:        m.Package.Type,
			Readme:      readme,
			Description: m.Package.Description,
			Unofficial:  p.unofficial,
			Links: struct {
				Source   string "json:\"source,omitempty\""
				Homepage string "json:\"homepage,omitempty\""
			}{
				Source:   m.Package.Source,
				Homepage: m.Package.Homepage,
			},
		})
		if err != nil {
			return err
		}
		logger.Info("Project " + project.Name + " created")
	} else if err != nil {
		return err
	}

	// TODO: reimplement!
	// if res.Header.Get("mpkg-write-access") == "" {
	// 	// TODO: check for other problems here!
	// 	logger.Fail("Do not have write access for " + m.Package.Name)
	// }

	tasks.Log("Determening version to publish")
	switch {
	case p.versionName != "":
		tasks.Log("Using supplied release version number: " + p.versionName)
		m.Package.Version = strings.TrimPrefix(p.versionName, "v")
	case m.Package.Version != "":
		tasks.Log("Using version number in minepkg.toml: " + m.Package.Version)
	default:
		logger.Fail("No version set in minepkg.toml and no release version passed. Please set one.")
	}

	if m.Package.Version == "" {
		logger.Fail("Could not determine version to publish")
	}

	if validSemver := semverMatch.MatchString(m.Package.Version); !validSemver {
		return &commands.CliError{
			Text: "release version is not valid semver",
			Suggestions: []string{
				"Check that your supplied version is valid semver",
				"Semver versions always look like major.minor.patch (eg. 2.1.0)",
				"You can also use a prerelease semver version or add build info",
				"See https://semver.org/ for more info",
			},
		}
	}

	// check if version exists
	logger.Log("Checking if release exists: ")
	p.release, err = apiClient.GetRelease(context.TODO(), m.PlatformString(), m.Package.Name+"@"+m.Package.Version)

	switch {
	case err == nil && p.release.Meta.Published:
		if time.Since(*p.release.Meta.CreatedAt) > time.Hour*24*2 {
			logger.Fail("Release already published!")
		}
		logger.Info("Release already published but can be overwritten.")
		logger.Warn("Overwriting might take some time to fully apply everywhere. (~30 minutes)")
		input := confirmation.New("Delete & overwrite the existing release?", confirmation.Yes)
		overwrite, err := input.RunPrompt()
		if !overwrite {
			logger.Info("Aborting")
			os.Exit(0)
		}
		_, err = apiClient.DeleteRelease(context.TODO(), m.PlatformString(), m.Package.Name+"@"+m.Package.Version)
		if err != nil {
			return fmt.Errorf("could not delete old release: %w", err)
		}
		// no release anymore, make sure it gets created
		p.release = nil
		// no error, continue normally
	case err != nil && err != api.ErrNotFound:
		// unknown error
		return err
	}

	tasks.Step("🏗", "Building")

	buildCmd := m.Dev.BuildCommand
	if (buildCmd == "" && m.Package.Type == manifest.TypeModpack) || p.file != "" {
		p.noBuild = true
	}

	if !p.noBuild {
		build := instance.BuildMod()
		cmdTerminalOutput(build)
		build.Start()
		err := build.Wait()
		if err != nil {
			// TODO: output logs or something
			fmt.Println(err)
			logger.Fail("Build step failed. Aborting")
		}
	} else {
		logger.Info("Skipped build")
	}

	var artifact string

	if m.Package.Type == manifest.TypeMod {
		// find se jar
		tasks.Log("Finding jar file")
		artifact, err = p.findJar(instance)
		if err != nil {
			return err
		}
	} else {
		// find all modpack related files
		tasks.Log("Archiving modpack file")
		// TODO: can fail, better logging, allow modpacks without any files
		artifact = buildModpackZIP()
	}

	tasks.Step("☁", "Uploading package")

	if p.dry {
		logger.Info("Skipping upload because this is a dry run")
		if artifact != "" {
			logger.Info("Build package can be found here: " + artifact)
		}
		os.Exit(0)
	}

	if !nonInteractive {
		input := confirmation.New("Do you want to publish this now?", confirmation.Yes)
		overwrite, err := input.RunPrompt()
		if !overwrite || err != nil {
			logger.Info("Aborting")
			os.Exit(0)
		}
	}

	if p.release == nil {
		logger.Info("Creating release")
		r := apiClient.NewUnpublishedRelease(m)
		if artifact == "" {
			r = apiClient.NewRelease(m)
		}
		p.release, err = project.CreateRelease(context.TODO(), r)
		if err != nil {
			if merr, ok := err.(*api.MinepkgError); ok {
				return fmt.Errorf("[%d] Api error: %s", merr.StatusCode, merr.Message)
			} else {
				return fmt.Errorf("%w. Check internet connection or report bug", err)
			}
		}
	}

	// upload the file
	if err := p.uploadArtifact(artifact); err != nil {
		logger.Warn("Upload failed. Removing release")
		if _, err := apiClient.DeleteRelease(context.TODO(), p.release.Package.Platform, p.release.Identifier()); err != nil {
			return fmt.Errorf("error-inception: could not delete release after error. %w", err)
		}
		return fmt.Errorf("upload failed: %w", err)
	}

	logger.Info(" ✓ Released " + p.release.Package.Version)
	return nil
}

func (p *publishRunner) uploadArtifact(artifact string) error {
	if artifact != "" {
		logger.Info("Uploading artifact (" + artifact + ")")
		file, err := os.Open(artifact)
		if err != nil {
			return err
		}
		defer file.Close()
		fsStat, err := file.Stat()
		if err != nil {
			return err
		}
		_, err = p.release.Upload(file, fsStat.Size())
		return err
	} else if !p.release.Meta.Published {
		return fmt.Errorf("release expects an artifact to upload but we have nothing to upload")
	}
	return nil
}

func getReadme() (string, error) {
	files, err := ioutil.ReadDir(".")
	// something is wrong
	if err != nil {
		return "", err
	}

	readme := ""
	for _, file := range files {
		if strings.HasPrefix(strings.ToLower(file.Name()), "readme") {
			readme = filepath.Join("./", file.Name())
		}
	}

	if readme == "" {
		return "", errors.New("could not find any readme file")
	}

	file, err := ioutil.ReadFile(readme)
	if err != nil {
		return "", err
	}
	return string(file), nil

}

func buildModpackZIP() string {
	tmpZip, err := ioutil.TempFile("", "modpack-*.zip")
	if err != nil {
		panic(err)
	}
	archive := zip.NewWriter(tmpZip)
	fileCount := 0

	// TODO: custom ignore list
	c, err := addToZip(archive, "./overwrites", defaultFilter)
	if err != nil {
		log.Println(err)
	}
	fileCount += c

	c, err = addToZip(archive, "./overwrites/saves", savesFilter)
	if err != nil {
		log.Println(err)
	}
	fileCount += c

	if fileCount == 0 {
		os.Remove(tmpZip.Name())
		return ""
	}

	archive.Close()

	return tmpZip.Name()
}

type filter func(string) bool

func addToZip(archive *zip.Writer, path string, filter ...filter) (int, error) {
	addCount := 0
	filepath.Walk(path, func(origPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// get paths as if "overwites" is the root
		path, err = filepath.Rel("overwrites", origPath)
		if err != nil {
			return err
		}

		source, err := os.Open(origPath)
		if err != nil {
			return err
		}
		defer source.Close()

		if info.IsDir() {
			return nil
		}

		// skip filtered paths
		for _, f := range filter {
			if !f(path) {
				return nil
			}
		}

		target, err := archive.Create(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(target, source)
		if err != nil {
			return err
		}
		addCount++
		return nil
	})
	return addCount, nil
}

func savesFilter(path string) bool {
	no := []string{"advancements", "playerdata", "stats", "session.lock"}
	parts := strings.Split(path, string(filepath.Separator))

	if len(parts) > 1 {
		for _, disallowed := range no {
			// check the third part for filtered paths (eg. find stats in `saves/save-name/stats`)
			if strings.HasPrefix(parts[2], disallowed) {
				return false
			}
		}
	}

	return true
}

func defaultFilter(path string) bool {
	no := []string{"minecraft", ".git", "minepkg.toml", "minepkg-lock.toml", ".", "saves"}

	for _, disallowed := range no {
		if strings.HasPrefix(path, disallowed) {
			return false
		}
	}

	return true
}

func (p *publishRunner) findJar(instance *instances.Instance) (string, error) {
	if p.file != "" {
		abs, err := filepath.Abs(p.file)
		if err != nil {
			return "", fmt.Errorf("provided --file path '%s' is invalid:\n  %w", p.file, err)
		}
		return abs, nil
	}

	match, err := getJarFileForInstance(instance)
	if err != nil {
		return "", err
	}
	return match.Path(), nil
}
