package cmd

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/briandowns/spinner"
	"github.com/fiws/minepkg/internals/cmdlog"
	"github.com/fiws/minepkg/pkg/api"
	"github.com/fiws/minepkg/pkg/manifest"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

// doomRegex: (\d+\.\d+\.\d-)?(\d+\.\d+\.\d)(.+)?
var semverDoom = regexp.MustCompile(`(\d+\.\d+\.\d+-)?(\d+\.\d+\.\d+)(.+)?`)
var apiURL = "https://test-api.minepkg.io/v1"

var (
	dry            bool
	skipBuild      bool
	nonInteractive bool
	release        string
)

func init() {
	publishCmd.Flags().BoolVarP(&dry, "dry", "", false, "Dry run without publishing")
	publishCmd.Flags().BoolVarP(&skipBuild, "skip-build", "", false, "Skips building the package")
	publishCmd.Flags().BoolVarP(&nonInteractive, "non-interactive", "y", false, "Answers all interactive questions with the default")
	publishCmd.Flags().StringVarP(&release, "release", "r", "", "The release version number to publish")
	rootCmd.AddCommand(publishCmd)
}

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publishes the local package in the current directory",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {

		tasks := logger.NewTask(3)
		tasks.Step("📚", "Preparing Publish")

		tasks.Log("Checking minepkg.toml")
		minepkg, err := ioutil.ReadFile("./minepkg.toml")
		if err != nil {
			logger.Fail("Could not find a minepkg.toml in this directory")
		}

		m := manifest.Manifest{}
		_, err = toml.Decode(string(minepkg), &m)
		if err != nil {
			logger.Fail(err.Error())
		}

		switch {
		case m.Requirements.Minecraft == "":
			logger.Fail("Your minepkg.toml is missing a minecraft version under [requirements]")
		case m.Requirements.Forge == "" && m.Requirements.Fabric == "":
			logger.Fail("Your minepkg.toml is missing either forge or fabric in [requirements]")
		}

		tasks.Log("Checking Authentication")
		if apiClient.JWT == "" {
			logger.Warn("You need to login to minepkg.io first")
			minepkgLogin()
		}

		logger.Log("Checking access rights")
		timeout, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		project, err := apiClient.GetProject(timeout, m.Package.Name)

		if err == api.ErrorNotFound {
			if nonInteractive != true {
				create := boolPrompt(&promptui.Prompt{
					Label:     "Project " + m.Package.Name + " does not exists yet. Do you want to create it",
					Default:   "Y",
					IsConfirm: true,
				})
				if create != true {
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
				logger.Fail(err.Error())
			}
			logger.Info("Project " + project.Name + " created")
		} else if err != nil {
			logger.Fail(err.Error())
		}

		// TODO: reimplement!
		// if res.Header.Get("mpkg-write-access") == "" {
		// 	// TODO: check for other problems here!
		// 	logger.Fail("Do not have write access for " + m.Package.Name)
		// }

		tasks.Log("Determening version to publish")
		switch {
		case release != "":
			tasks.Log("Using supplied release version number")
			m.Package.Version = release
		case m.Package.Version != "":
			tasks.Log("Using version number in minepkg.toml")
		default:
			repo, err := git.PlainOpen("./")
			if err != nil {
				logger.Fail("Can't fallback to git tag version (not a valid git repo)")
			}
			iter, err := repo.Tags()

			// TODO: warn user about quirks match
			iter.ForEach(func(ref *plumbing.Reference) error {
				match := versionFromTag(ref)
				if match != "" {
					m.Package.Version = match
				}

				return nil
			})

			logger.Info("  Found version from git " + m.Package.Version)
		}

		if m.Package.Version == "" {
			logger.Fail("Could not determine version to publish")
		}

		// check if version exists
		logger.Log("Checking if release exists")

		// TODO: static fabric is bad!
		release, err := apiClient.GetRelease(context.TODO(), "fabric", m.Package.Name+"@"+m.Package.Version)

		switch {
		case err == nil && release.Meta.Published != false:
			logger.Fail("Release already published!")
		case err != nil && err != api.ErrorNotFound:
			// unknown error
			logger.Fail(err.Error())
		}

		// just publish the manifest for modpacks
		// TODO: check if anything needs publishing
		// if m.Package.Type == manifest.TypeModpack {
		// 	logger.Info("Creating release")
		// 	release, err = project.CreateRelease(context.TODO(), &m)
		// 	if err != nil {
		// 		logger.Fail(err.Error())
		// 	}
		// 	logger.Info(" ✓ Released " + release.Package.Version)
		// 	return
		// }

		tasks.Step("🏗", "Building")

		if m.Hooks.Build == "" && m.Package.Type == manifest.TypeModpack {
			skipBuild = true
		}

		if skipBuild != true {
			buildScript := "gradle --build-cache build"
			if m.Hooks.Build != "" {
				tasks.Log("Using custom build hook")
				tasks.Log(" running " + m.Hooks.Build)
				buildScript = m.Hooks.Build
			} else {
				tasks.Log("Using default build step (gradle --build-cache build)")
			}

			// TODO: I don't think this is multi platform
			build := exec.Command("sh", []string{"-c", buildScript}...)
			build.Env = os.Environ()

			if nonInteractive == true {
				terminalOutput(build)
			}

			var spinner func()
			if nonInteractive != true {
				spinner = spinnerOutput(build)
			}

			startTime := time.Now()
			build.Start()
			if nonInteractive != true {
				spinner()
			}

			err = build.Wait()
			if err != nil {
				// TODO: output logs or something
				fmt.Println(err)
				logger.Fail("Build step failed. Aborting")
			}

			logger.Info(" ✓ Finished build in " + time.Now().Sub(startTime).String())
		} else {
			logger.Info("Skipped build")
		}

		var artifact string

		if m.Package.Type == manifest.TypeMod {
			// find se jar
			tasks.Log("Finding jar file")
			artifact = findJar()
		} else {
			// find all modpack related files
			tasks.Log("Archiving modpack file")
			// TODO: can fail, better logging, allow modpacks without any files
			artifact = buildModpackZIP()
		}

		tasks.Step("☁", "Uploading package")

		if dry == true {
			logger.Info("Skipping upload because this is a dry run")
			if artifact != "" {
				logger.Info("Build package can be found here: " + artifact)
			}
			os.Exit(0)
		}

		if release == nil {
			logger.Info("Creating release")
			r := apiClient.NewUnpublishedRelease(&m)
			if artifact == "" {
				r = apiClient.NewRelease(&m)
			}
			release, err = project.CreateRelease(context.TODO(), r)
			if err != nil {
				if merr, ok := err.(*api.MinepkgError); ok {
					errorMsg := fmt.Sprintf("[%d] Api error: %s", merr.StatusCode, merr.Message)
					logger.Fail(errorMsg)
				} else {
					logger.Fail(err.Error() + ". Check internet connection or report bug!")
				}
			}
		}

		// upload tha file
		if artifact != "" {
			logger.Info("Uploading artifact (" + artifact + ")")
			file, err := os.Open(artifact)
			if release, err = release.Upload(file); err != nil {
				logger.Fail(err.Error())
			}
		} else if release.Meta.Published == false {
			logger.Fail("This release expects an artifact to upload but we have nothing to upload.\n Contact support to cleanup this package")
		}

		logger.Info(" ✓ Released " + release.Package.Version)
	},
}

func spinnerOutput(build *exec.Cmd) func() {
	stdout, _ := build.StdoutPipe()
	scanner := bufio.NewScanner(stdout)
	// TODO: stderr!!

	return func() {
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond) // Build our new spinner
		s.Prefix = " "
		s.Start()
		s.Suffix = " [no build output yet]"

		maxTextWidth := terminalWidth() - 4 // spinner + spaces
		for scanner.Scan() {
			s.Suffix = " " + truncateString(scanner.Text(), maxTextWidth)
		}
		stdout.Close()
		s.Suffix = ""
		s.Stop()
	}
}

func terminalOutput(b *exec.Cmd) {
	b.Stderr = os.Stderr
	b.Stdout = os.Stdout
}

func injectManifest(r *zip.ReadCloser, m *manifest.Manifest) error {
	dest, err := os.Create("tmp-minepkg-package.jar")
	if err != nil {
		return err
	}
	// Create a new zip archive.
	w := zip.NewWriter(dest)

	// generate toml
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(m); err != nil {
		return err
	}

	f, err := w.Create("minepkg.toml")
	if err != nil {
		return err
	}
	f.Write(buf.Bytes())

	for _, f := range r.File {
		target, err := w.CreateHeader(&f.FileHeader)
		if err != nil {
			log.Fatal(err)
		}
		reader, err := f.Open()
		if err != nil {
			return err
		}
		_, err = io.Copy(target, reader)
		if err != nil {
			return err
		}
	}
	return w.Close()
}

func versionFromTag(ref *plumbing.Reference) string {
	tagName := string(ref.Name())[10:]

	var version string
	matches := semverDoom.FindStringSubmatch(tagName)

	switch {
	case len(matches) == 0:
		return version
	case matches[2] != "" && matches[3] == "":
		version = matches[2]
	case matches[2] != "" && matches[3] != "":
		version = matches[2] + "+" + matches[3][1:]
	}
	return version
}

func terminalWidth() int {
	fd := int(os.Stdout.Fd())
	termWidth, _, _ := terminal.GetSize(fd)
	return termWidth
}

func truncateString(str string, num int) string {
	bnoden := str
	if len(str) > num {
		if num > 3 {
			num -= 3
		}
		bnoden = str[0:num] + "..."
	}
	return bnoden
}

func getReadme() (string, error) {
	files, err := ioutil.ReadDir(".")
	// something is wrong
	if err != nil {
		logger.Fail(err.Error())
	}

	readme := ""
	for _, file := range files {
		if strings.HasPrefix(strings.ToLower(file.Name()), "readme") {
			readme = filepath.Join("./", file.Name())
		}
	}

	if readme == "" {
		return "", errors.New("Could not find any readme file")
	}

	file, err := ioutil.ReadFile(readme)
	if err != nil {
		logger.Fail(err.Error())
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
			if f(path) == false {
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

func findJar() string {
	files, err := ioutil.ReadDir("./build/libs")
	if err != nil {
		logger.Fail(err.Error())
	}
	if len(files) == 0 {
		logger.Fail("No build files found in ./build/libs")
	}

	chosen := files[0]

search:
	for _, file := range files[1:] {
		name := file.Name()
		base := filepath.Base(name)

		// filter out dev and sources jars
		switch {
		case strings.HasSuffix(base, "dev.jar"):
			continue
		case strings.HasSuffix(base, "sources.jar"):
			continue
		// worldedit uses dist for the runnable jars. lets hope this
		// does not break any other mods
		case strings.HasSuffix(base, "dist.jar"):
			// we choose this file and stop
			chosen = file
			break search
		}
		if len(file.Name()) < len(chosen.Name()) {
			chosen = file
		}
	}

	return filepath.Join("./build/libs", chosen.Name())
}

func buildMod(m *manifest.Manifest, tasks *cmdlog.Task) {
	buildScript := "gradle --build-cache build"
	if m.Hooks.Build != "" {
		tasks.Log("Using custom build hook")
		tasks.Log(" running " + m.Hooks.Build)
		buildScript = m.Hooks.Build
	} else {
		tasks.Log("Using default build step (gradle --build-cache build)")
	}

	// TODO: I don't think this is multi platform
	build := exec.Command("sh", []string{"-c", buildScript}...)
	build.Env = os.Environ()

	if nonInteractive == true {
		terminalOutput(build)
	}

	var spinner func()
	if nonInteractive != true {
		spinner = spinnerOutput(build)
	}

	startTime := time.Now()
	build.Start()
	if nonInteractive != true {
		spinner()
	}

	err := build.Wait()
	if err != nil {
		// TODO: output logs or something
		fmt.Println(err)
		logger.Fail("Build step failed. Aborting")
	}

	logger.Info(" ✓ Finished build in " + time.Now().Sub(startTime).String())
}
