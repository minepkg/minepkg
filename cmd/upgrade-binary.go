package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

type minepkgClientVersions struct {
	Version  string `json:"version"`
	Binaries struct {
		Win   string `json:"win"`
		MacOS string `json:"macos"`
		Linux string `json:"linux"`
	} `json:"binaries"`
}

func (m *minepkgClientVersions) PlatformBinary() string {
	switch runtime.GOOS {
	case "linux":
		return m.Binaries.Linux
	case "macos":
		return m.Binaries.MacOS
	case "windows":
		return m.Binaries.Win
	default:
		panic("No binary available for your platform")
	}
}

var selfupdateCmd = &cobra.Command{
	Use:   "selfupdate",
	Short: "Updates minepkg to the latest version",
	Run: func(cmd *cobra.Command, args []string) {

		toUpdate, err := os.Executable()
		if err != nil {
			logger.Fail(err.Error())
		}

		toUpdate, err = filepath.EvalSymlinks(toUpdate)
		if err != nil {
			logger.Fail(err.Error())
		}

		fmt.Println("Checking for new version")
		res, err := http.Get("https://storage.googleapis.com/minepkg-client/latest-version.json")
		if err != nil {
			logger.Fail(err.Error())
		}

		buf, err := ioutil.ReadAll(res.Body)
		if err != nil {
			logger.Fail(err.Error())
		}
		parsed := minepkgClientVersions{}
		if err := json.Unmarshal(buf, &parsed); err != nil {
			logger.Fail(err.Error())
		}

		fmt.Println("Downloading new version")
		// TODO: if this version is newer
		newCli, err := ioutil.TempFile(filepath.Dir(toUpdate), parsed.Version)
		newCli.Chmod(0700)
		download, err := http.Get(parsed.PlatformBinary())
		io.Copy(newCli, download.Body)

		newCli.Close()

		fmt.Println("Testing new version")
		test := exec.Command(newCli.Name(), "selftest")
		out, err := test.Output()
		if err != nil {
			logger.Fail("Update aborted. Self test of new update failed:\n " + err.Error())
		}
		if string(out) != "Selftest OK\n" {
			logger.Fail("Update aborted. Self test of new update failed:\nInvalid output. Please open a bug report")
		}

		if err := os.Rename(newCli.Name(), toUpdate); err != nil {
			logger.Fail(err.Error())
		}
		fmt.Println("minepkg CLI was updated successfully")

	},
}

var selftestCmd = &cobra.Command{
	Use:    "selftest",
	Short:  "checks if this binary works",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Selftest OK")
	},
}