package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/internals/github"
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

type minepkgRelease struct {
	*github.Release
}

// Version returns the version of the release (it strips the v prefix)
func (m *minepkgRelease) Version() string {
	return m.Release.Name[1:]
}

func (m *minepkgRelease) Extension() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func (m *minepkgRelease) ArchiveName() string {
	return fmt.Sprintf(
		"minepkg_%s_%s_%s%s",
		m.Version(),
		runtime.GOOS,
		runtime.GOARCH,
		m.Extension(),
	)
}

func (m *minepkgRelease) ArchiveURL() string {
	for _, asset := range m.Release.Assets {
		if strings.EqualFold(asset.Name, m.ArchiveName()) {
			return asset.BrowserDownloadURL
		}
	}
	panic("No archive found for your platform")
}

func (m *minepkgRelease) BinName() string {
	switch runtime.GOOS {
	case "linux":
		return "minepkg"
	case "darwin": // macOS
		return "minepkg"
	case "windows":
		return "minepkg.exe"
	default:
		panic("No binary available for your platform (how did you even..?)")
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
	latestRaw, err := s.fetchVersionInfo()
	if err != nil {
		return err
	}

	latest := &minepkgRelease{latestRaw}

	if latest.Version() == rootCmd.Version && !s.force {
		fmt.Println("Already up to date! :)")
		os.Exit(0)
	}

	fmt.Println("Downloading new version")
	fmt.Println("  " + latest.ArchiveURL())

	// TODO: reimplement this
	// if parsed.Info != "" {
	// 	fmt.Println(styleWarnBox.Render(parsed.Info))
	// }

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	minepkgCacheDir := filepath.Join(cacheDir, "minepkg")

	wd, err := os.MkdirTemp(minepkgCacheDir, latest.Version())
	if err != nil {
		return err
	}
	defer os.RemoveAll(wd)
	// change working directory to temp dir
	if err := os.Chdir(wd); err != nil {
		return err
	}

	// TODO: if this version is newer
	binaryTargetPath := filepath.Join(wd, latest.BinName())
	binary, err := os.Create(binaryTargetPath)
	if err != nil {
		return err
	}

	binDownload, err := http.Get(latest.ArchiveURL())
	if err != nil {
		return err
	}
	_, err = io.Copy(binary, binDownload.Body)
	if err != nil {
		return err
	}

	fmt.Println("Downloaded to", binaryTargetPath)

	binary.Chmod(0700)
	binary.Close()

	// newCli.Chmod(0700)
	fmt.Println("Testing new version")
	test := exec.Command(binaryTargetPath, "selftest")
	out, err := test.Output()
	if err != nil {
		logger.Fail("Update aborted. Self test of new update failed:\n " + err.Error())
	}
	if string(out) != "Selftest OK\n" {
		logger.Fail("Update aborted. Self test of new update failed:\nInvalid output. Please open a bug report")
	}

	fmt.Println("Updating minepkg")

	if runtime.GOOS == "windows" {
		if err := os.Rename(toUpdate, toUpdate+".old"); err != nil {
			return err
		}
	}

	if err := os.Rename(binary.Name(), toUpdate); err != nil {
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

func (s *selfupdateRunner) fetchVersionInfo() (*github.Release, error) {
	updateChannel := viper.GetString("updateChannel")
	switch updateChannel {
	case "":
		fallthrough
	case "stable":
		fmt.Println("Using stable update channel")
	case "dev":
		fmt.Println("Dev update channel is currently unsupported, using stable")
	default:
		fmt.Printf("Unsupported update channel \"%s\". Falling back to stable\n", updateChannel)
	}

	release, err := github.GetLatestRelease(context.TODO(), "minepkg/minepkg")
	if err != nil {
		return nil, err
	}

	return release, nil
}

var selftestCmd = &cobra.Command{
	Use:    "selftest",
	Short:  "checks if this binary works",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Selftest OK")
	},
}
