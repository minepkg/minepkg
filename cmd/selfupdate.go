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

	"github.com/charmbracelet/lipgloss"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var styleWarnBox = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#b37400")).
	Foreground(lipgloss.Color("orange")).
	Padding(0, 1)

func init() {
	runner := &selfupdateRunner{}

	cmd := commands.New(&cobra.Command{
		Use:     "selfupdate",
		Aliases: []string{"self-update"},
		Short:   "Updates minepkg to the latest version",
		Args:    cobra.ExactArgs(0),
	}, runner)

	cmd.Flags().BoolVar(&runner.force, "force", false, "Force update")

	rootCmd.AddCommand(cmd.Command)
	rootCmd.AddCommand(selftestCmd)
}

type minepkgClientVersions struct {
	Version  string `json:"version"`
	Channel  string `json:"channel"`
	Info     string `json:"info"`
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
	case "darwin": // macOS
		return m.Binaries.MacOS
	case "windows":
		return m.Binaries.Win
	default:
		panic("No binary available for your platform")
	}
}

type selfupdateRunner struct {
	force bool
}

func (s *selfupdateRunner) RunE(cmd *cobra.Command, args []string) error {
	toUpdate, err := os.Executable()
	if err != nil {
		return err
	}

	toUpdate, err = filepath.EvalSymlinks(toUpdate)
	if err != nil {
		return err
	}

	fmt.Println("Checking for new version")
	parsed, err := s.fetchVersionInfo()
	if err != nil {
		return err
	}

	if parsed.Version == rootCmd.Version && !s.force {
		fmt.Println("Already up to date! :)")
		os.Exit(0)
	}

	if parsed.Info != "" {
		fmt.Println(styleWarnBox.Render(parsed.Info))
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	// TODO: if this version is newer
	newCli, err := ioutil.TempFile(cacheDir, parsed.Version)
	if err != nil {
		return err
	}
	newCli.Chmod(0700)
	download, err := http.Get(parsed.PlatformBinary())
	if err != nil {
		return err
	}
	_, err = io.Copy(newCli, download.Body)
	if err != nil {
		return err
	}

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

	if runtime.GOOS == "windows" {
		if err := os.Rename(toUpdate, toUpdate+".old"); err != nil {
			return err
		}
	}

	if err := os.Rename(newCli.Name(), toUpdate); err != nil {
		if runtime.GOOS == "windows" {
			// revert to old version
			if err := os.Rename(toUpdate+".old", toUpdate); err != nil {
				panic("This is bad... You might have to install minepkg manually again. Sorry")
			}
			logger.Fail("Upgrade failed. Reverted to old version. Please open a bug report")
		}
		return err
	}
	fmt.Println("minepkg CLI was updated successfully")

	return err
}

func (s *selfupdateRunner) fetchVersionInfo() (*minepkgClientVersions, error) {
	updateChannel := viper.GetString("updateChannel")
	pathPrefix := ""
	switch updateChannel {
	case "":
		fallthrough
	case "stable":
		fmt.Println("Using stable update channel")
	case "dev":
		fmt.Println("Using dev update channel")
		pathPrefix = "dev/"
	default:
		fmt.Printf("Unsupported update channel \"%s\". Falling back to stable\n", updateChannel)
	}

	res, err := http.Get(fmt.Sprintf("https://get.minepkg.io/%slatest-version.json", pathPrefix))
	if err != nil {
		return nil, err
	}

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	parsed := minepkgClientVersions{}
	if err := json.Unmarshal(buf, &parsed); err != nil {
		return nil, err
	}

	return &parsed, nil
}

var selftestCmd = &cobra.Command{
	Use:    "selftest",
	Short:  "checks if this binary works",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Selftest OK")
	},
}
