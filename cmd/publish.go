package cmd

import (
	"regexp"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4"
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/fiws/minepkg/pkg/manifest"

	"github.com/spf13/cobra"
)

// doomRegex: (\d+\.\d+\.\d-)?(\d+\.\d+\.\d)(.+)?
var semverDoom = regexp.MustCompile(`(\d+\.\d+\.\d-)?(\d+\.\d+\.\d)(.+)?`)
var apiURL = "https://test-api.minepkg.io/v1/"

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
		token := os.Getenv("MINEPKG_API_TOKEN")
		if token == "" {
			logger.Fail("Missing MINEPKG_API_TOKEN environment variable!")
		}

		client := &http.Client{}

		logger.Log("Checking Access rights")
		req, _ := http.NewRequest("GET", apiURL+"/projects/"+m.Package.Name, nil)
		req.Header.Add("Authorization", "Bearer "+token)
		res, err := client.Do(req)
		if err != nil {
			logger.Fail(err.Error())
		}

		if res.StatusCode != http.StatusOK {
			// TODO: check for other problems here!
			logger.Fail("Response not ok: " + res.Status)
		}

		if res.Header.Get("mpkg-write-access") == "" {
			// TODO: check for other problems here!
			logger.Fail("Do not have write access for " + m.Package.Name)
		}

		tasks.Log("Determening version to publish")
		if m.Package.Version == "" {
			version := m.Package.Version
			repo, err := git.PlainOpen("./")
			if err != nil {
				logger.Fail("Can't fallback to git tag version (not a valid git repo)")
			}
			iter, err := repo.Tags()

			// TODO: warn user about quirks match
			iter.ForEach(func (ref *plumbing.Reference) error {
				tagName := string(ref.Name())[10:]

				matches := semverDoom.FindStringSubmatch(tagName)
				switch len(matches){
				case 3:
					version = matches[2]
				case 4:
					version = matches[2] + "+" + matches[3][1:]
				}
				
				return nil
			})

			m.Package.Version = version
			logger.Info("Found version " + version)
		} else {
			tasks.Log("Using version number in minepkg.toml")
		}

		// check if version exists
		{
			req, _ := http.NewRequest("GET", apiURL+"/projects/"+m.Package.Name+"@"+m.Package.Version, nil)
			req.Header.Add("Authorization", "Bearer "+token)
			res, err := client.Do(req)
			if err != nil {
				logger.Fail(err.Error())
			}
			if res.StatusCode != http.StatusNotFound {
				// TODO: check for other problems here!
				logger.Fail("Release already exists!")
			}
		}

		tasks.Step("🏗", "Building")

		buildScript := "gradle --build-cache build"
		if m.Hooks.Build != "" {
			tasks.Log("Using custom build hook")
			tasks.Log("» " + m.Hooks.Build)
			buildScript = m.Hooks.Build
		} else {
			tasks.Log("Using default build step (gradle --build-cache build)")
		}

		// TODO: I don't think this i multi platform
		build := exec.Command("sh", []string{"-c", buildScript}...)
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		err = build.Run()
		if err != nil {
			tasks.Fail("Build step failed. Aborting")
		}

		tasks.Log("Finished custom build hook")
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

		file, err := os.Open("tmp-minepkg-package.jar")
		stat, _ := file.Stat()
		upload, _ := http.NewRequest("PUT", apiURL+"/projects/"+m.Package.Name+"@"+m.Package.Version, file)
		upload.Header.Add("Authorization", "Bearer "+token)
		upload.Header.Add("Content-Type", "application/java-archive")
		// next line is great
		upload.Header.Add("Content-Length", strconv.Itoa(int(stat.Size())))
		uploadRes, err := client.Do(upload)

		if err != nil {
			logger.Fail(err.Error())
		}

		if uploadRes.StatusCode != http.StatusCreated {
			// TODO: check for other problems here!
			logger.Fail("Response not ok: " + uploadRes.Status)
		}

		logger.Info("Release succesfully published")
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
