package instances

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fiws/minepkg/internals/minecraft"
)

func extractNative(jar string, target string) error {
	r, err := zip.OpenReader(jar)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		// skip META-INF dir
		if strings.HasPrefix(f.Name, "META-INF") {
			continue
		}

		if err := sanitizeExtractPath(f.Name, target); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		f, err := os.Create(filepath.Join(target, f.Name))
		if err != nil {
			return err
		}

		io.Copy(f, rc)
		rc.Close()
	}
	return nil
}

// v for variable
func v(s string) string {
	return "${" + s + "}"
}

// TODO: remove
func existOrDownload(lib minecraft.Lib) {
	home, _ := os.UserHomeDir()
	globalDir := filepath.Join(home, ".minepkg/libraries")
	path := filepath.Join(globalDir, lib.Filepath())
	url := lib.DownloadURL()
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "osx"
	}
	if lib.Natives[osName] != "" {
		nativeID, _ := lib.Natives[osName]
		native := lib.Downloads.Classifiers[nativeID]
		url = native.URL
		path = filepath.Join(globalDir, native.Path)
	}
	if _, err := os.Stat(path); err == nil {
		return
	}

	res, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	if res.StatusCode != http.StatusOK {
		panic(url + " did not return status code 200")
	}
	// create directory first
	os.MkdirAll(filepath.Dir(path), 0755)
	// file next
	target, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(target, res.Body)
	if err != nil {
		panic(err)
	}
}

func (i *Instance) ensureAssets(man *minecraft.LaunchManifest) error {

	missing, err := i.FindMissingAssets(man)
	if err != nil {
		return err
	}

	for _, asset := range missing {
		fileRes, err := http.Get(asset.DownloadURL())
		// TODO: check status code and all the things!
		os.MkdirAll(filepath.Join(i.GlobalDir, "assets/objects", asset.Hash[:2]), os.ModePerm)
		dest, err := os.Create(filepath.Join(i.GlobalDir, "assets/objects", asset.UnixPath()))
		if err != nil {
			return err
		}
		_, err = io.Copy(dest, fileRes.Body)
		if err != nil {
			return err
		}
	}

	return nil
}

// stolen from https://github.com/mholt/archiver/blob/e4ef56d48eb029648b0e895bb0b6a393ef0829c3/archiver.go#L110-L119
func sanitizeExtractPath(filePath string, destination string) error {
	// to avoid zip slip (writing outside of the destination), we resolve
	// the target path, and make sure it's nested in the intended
	// destination, or bail otherwise.
	destpath := filepath.Join(destination, filePath)
	if !strings.HasPrefix(destpath, destination) {
		return fmt.Errorf("%s: illegal file path", filePath)
	}
	return nil
}
