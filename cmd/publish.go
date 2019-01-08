package cmd

import (
	"context"
	"github.com/manifoldco/promptui"
	"golang.org/x/crypto/ssh/terminal"
	"github.com/fiws/minepkg/pkg/api"
	"github.com/briandowns/spinner"
	"time"
	"bufio"
	"regexp"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4"
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/fiws/minepkg/pkg/manifest"

	"github.com/spf13/cobra"
)

// doomRegex: (\d+\.\d+\.\d-)?(\d+\.\d+\.\d)(.+)?
var semverDoom = regexp.MustCompile(`(\d+\.\d+\.\d-)?(\d+\.\d+\.\d)(.+)?`)
var apiURL = "https://test-api.minepkg.io/v1"

var (
	dry bool
	skipBuild bool
	nonInteractive bool
)

func init() {
	publishCmd.Flags().BoolVarP(&dry, "dry", "", false, "Dry run without publishing")
	publishCmd.Flags().BoolVarP(&skipBuild, "skip-build", "", false, "Skips building the package")
	publishCmd.Flags().BoolVarP(&nonInteractive, "non-interactive", "y", false, "Answers all interactive questions with the default")
}

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publishes a local mod in the current directory",
	Run: func(cmd *cobra.Command, args []string) {
		
		// overwrite api
		// TODO: don't do that here
		if customAPI := os.Getenv("MINEPKG_API"); customAPI != "" {
			apiURL = customAPI
		}
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

		if m.Package.Type != manifest.TypeMod {
			logger.Fail("Only mod can be published (for now)")
		}

		tasks.Log("Checking Authentication")
		// token := os.Getenv("MINEPKG_API_TOKEN")
		// if token == "" {
		// 	logger.Fail("Missing MINEPKG_API_TOKEN environment variable!")
		// }
		if apiClient.JWT == "" {
			logger.Warn("You need to login first")
			login()
		}

		logger.Log("Checking access rights")
		timeout, cancel := context.WithTimeout(context.Background(), time.Second * 5)
		defer cancel()
		_, err = apiClient.GetProject(timeout, m.Package.Name)

		if err == api.ErrorNotFound {
			if nonInteractive != true {
				create := boolPrompt(&promptui.Prompt {
					Label: "Project does not exists yet. Do you want to create it",
					Default: "Y",
					IsConfirm: true,
				})
				if create != true {
					logger.Info("Aborting")
					os.Exit(0)
				}
			}
			project, err := apiClient.CreateProject(&api.Project{
				Name: m.Package.Name,
				Type: m.Package.Type,
			})
			if err != nil {
				logger.Fail(err.Error())
			}
			logger.Info("Project "+project.Name+ " created")
		} else if (err != nil) {
			logger.Fail(err.Error())
		}

		// TODO: reimplement!
		// if res.Header.Get("mpkg-write-access") == "" {
		// 	// TODO: check for other problems here!
		// 	logger.Fail("Do not have write access for " + m.Package.Name)
		// }

		tasks.Log("Determening version to publish")
		switch {
		case m.Package.Version != "":
			tasks.Log("Using version number in minepkg.toml")
		default:
			repo, err := git.PlainOpen("./")
			if err != nil {
				logger.Fail("Can't fallback to git tag version (not a valid git repo)")
			}
			iter, err := repo.Tags()

			// TODO: warn user about quirks match
			iter.ForEach(func (ref *plumbing.Reference) error {
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
		{
			_, err := apiClient.GetRelease(context.TODO(), m.Package.Name, m.Package.Version)

			switch {
			case err == nil:
				logger.Fail("Release already exists!")
			case err != nil && err != api.ErrorNotFound:
				// unknown error
				logger.Fail(err.Error())
			}
		}

		tasks.Step("🏗", "Building")

		if skipBuild != true {
			buildScript := "gradle --build-cache build"
			if m.Hooks.Build != "" {
				tasks.Log("Using custom build hook")
				tasks.Log(" running " + m.Hooks.Build)
				buildScript = m.Hooks.Build
			} else {
				tasks.Log("Using default build step (gradle --build-cache build)")
			}
	
			// TODO: I don't think this i multi platform
			build := exec.Command("sh", []string{"-c", buildScript}...)
			// TODO: stderr !!!
			bStdout, _ := build.StdoutPipe()
			scanner := bufio.NewScanner(bStdout)
	
			s := spinner.New(spinner.CharSets[9], 100*time.Millisecond) // Build our new spinner
			s.Prefix = " "
			s.Start()
			s.Suffix = " [no build output yet]"
	
			startTime := time.Now()
			build.Start()
	
			maxTextWidth := terminalWidth() - 4 // spinner + spaces
			for scanner.Scan() {
				s.Suffix = " " + truncateString(scanner.Text(), maxTextWidth)
			}
			bStdout.Close()
			err = build.Wait()
			if err != nil {
				// TODO: output logs or something
				logger.Fail("Build step failed. Aborting")
			}
			s.Suffix = ""
			s.Stop()
	
			logger.Info(" ✓ Finished build in " + time.Now().Sub(startTime).String())
		} else {
			logger.Info("Skipped build")
		}

		// find se jar
		tasks.Log("Finding jar file")
		jar := findJar()

		logger.Info("Using " + jar)
		logger.Log("Checking for embedded minepkg.toml")
		r, err := zip.OpenReader(jar)
		if err != nil {
			logger.Fail("Broken jar file: " + err.Error())
		}
		defer r.Close()

		// Iterate through the files in the archive,
		hasManifest := false
		for _, f := range r.File {
			if f.Name == "minepkg.toml" {
				hasManifest = true
				break
			}
		}

		if hasManifest != true {
			logger.Info("package is missing minepkg.toml. Injecting it")
			err := injectManifest(r, &m)
			if err != nil {
				logger.Fail("Inject failed: " + err.Error())
			}
		}

		tasks.Step("☁", "Uploading package")

		if dry == true {
			logger.Info("Skipping upload because this is a dry run")
			os.Exit(0)
		}

		// upload tha file
		file, err := os.Open("tmp-minepkg-package.jar")	
		if _, err = apiClient.PutRelease(m.Package.Name, m.Package.Version, file); err != nil {
			logger.Fail(err.Error())
		}

		logger.Info(" ✓ Released " + m.Package.Version)
	},
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
	switch len(matches){
	case 3:
		version = matches[2]
	case 4:
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

func findJar() string {
	files, err := ioutil.ReadDir("./build/libs")
	if err != nil {
		logger.Fail(err.Error())
	}
	if len(files) == 0 {
		logger.Fail("No build files found in ./build/libs")
	}

	shortest := files[0]
	for _, file := range files[1:] {
		if len(file.Name()) < len(shortest.Name()) {
			shortest = file
		}
	}

	return filepath.Join("./build/libs", shortest.Name())

}
