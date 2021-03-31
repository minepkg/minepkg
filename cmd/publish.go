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
	"strings"
	"time"

	"github.com/fiws/minepkg/internals/api"
	"github.com/fiws/minepkg/internals/commands"
	"github.com/fiws/minepkg/internals/globals"
	"github.com/fiws/minepkg/internals/instances"
	"github.com/fiws/minepkg/pkg/manifest"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	runner := &publishRunner{}
	cmd := commands.New(&cobra.Command{
		Use:     "publish",
		Short:   "Publishes the local package in the current directory",
		Aliases: []string{"run", "start", "play"},
		Args:    cobra.MaximumNArgs(1),
	}, runner)

	cmd.Flags().BoolVarP(&runner.dry, "dry", "", false, "Dry run without publishing")
	cmd.Flags().BoolVarP(&runner.skipBuild, "skip-build", "", false, "Skips building the package")
	cmd.Flags().StringVarP(&runner.release, "release", "r", "", "The release version number to publish")

	rootCmd.AddCommand(cmd.Command)
}

type publishRunner struct {
	dry       bool
	skipBuild bool
	release   string
	file      string
}

func (p *publishRunner) RunE(cmd *cobra.Command, args []string) error {
	apiClient := globals.ApiClient
	nonInteractive := viper.GetBool("nonInteractive")

	tasks := logger.NewTask(3)
	tasks.Step("📚", "Preparing Publish")

	tasks.Log("Checking minepkg.toml")
	instance, err := instances.NewInstanceFromWd()
	if err != nil {
		return err
	}

	m := instance.Manifest

	switch {
	case m.Requirements.Minecraft == "":
		logger.Fail("Your minepkg.toml is missing a minecraft version under [requirements]")
	case m.Requirements.Forge == "" && m.Requirements.Fabric == "":
		logger.Fail("Your minepkg.toml is missing either forge or fabric in [requirements]")
	}

	tasks.Log("Checking Authentication")
	if !globals.ApiClient.HasCredentials() {
		logger.Warn("You need to login to minepkg.io first")
		runner := &mpkgLoginRunner{}
		runner.RunE(cmd, args)
	}

	logger.Log("Checking access rights")
	timeout, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	project, err := apiClient.GetProject(timeout, m.Package.Name)

	if err == api.ErrorNotFound {
		if !nonInteractive {
			create := boolPrompt(&promptui.Prompt{
				Label:     "Project " + m.Package.Name + " does not exists yet. Do you want to create it",
				Default:   "Y",
				IsConfirm: true,
			})
			if !create {
				logger.Info("Aborting")
				os.Exit(0)
			}
		}
		readme, _ := getReadme()
		project, err = apiClient.CreateProject(&api.Project{
			Name:   m.Package.Name,
			Type:   m.Package.Type,
			Readme: readme,
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
	case p.release != "":
		tasks.Log("Using supplied release version number: " + p.release)
		m.Package.Version = p.release
	case m.Package.Version != "":
		tasks.Log("Using version number in minepkg.toml: " + m.Package.Version)
	default:
		logger.Fail("No version set in minepkg.toml and no release version passed. Please set one.")
	}

	if m.Package.Version == "" {
		logger.Fail("Could not determine version to publish")
	}

	// check if version exists
	logger.Log("Checking if release exists: ")

	// TODO: static fabric is bad!
	release, err := apiClient.GetRelease(context.TODO(), "fabric", m.Package.Name+"@"+m.Package.Version)

	switch {
	case err == nil && release.Meta.Published:
		logger.Fail("Release already published!")
	case err != nil && err != api.ErrorNotFound:
		// unknown error
		return err
	}

	tasks.Step("🏗", "Building")

	buildCmd := m.Dev.BuildCommand
	if (buildCmd == "" && m.Package.Type == manifest.TypeModpack) || p.file != "" {
		p.skipBuild = true
	}

	if !p.skipBuild {
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

	if release == nil {
		logger.Info("Creating release")
		r := apiClient.NewUnpublishedRelease(m)
		if artifact == "" {
			r = apiClient.NewRelease(m)
		}
		release, err = project.CreateRelease(context.TODO(), r)
		if err != nil {
			if merr, ok := err.(*api.MinepkgError); ok {
				return fmt.Errorf("[%d] Api error: %s", merr.StatusCode, merr.Message)
			} else {
				return fmt.Errorf("%w. Check internet connection or report bug", err)
			}
		}
	}

	// upload the file
	if artifact != "" {
		logger.Info("Uploading artifact (" + artifact + ")")
		file, err := os.Open(artifact)
		if err != nil {
			return err
		}
		if release, err = release.Upload(file); err != nil {
			return err
		}
	} else if !release.Meta.Published {
		logger.Fail("This release expects an artifact to upload but we have nothing to upload.\n Contact support to cleanup this package")
	}

	logger.Info(" ✓ Released " + release.Package.Version)
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

	return instance.FindModJar()
}
