package cmd

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mholt/archiver"

	"github.com/briandowns/spinner"
	"github.com/fiws/minepkg/internals/downloadmgr"
	"github.com/fiws/minepkg/internals/instances"
	"github.com/spf13/cobra"
)

var (
	version    string
	serverMode bool
)

func init() {
	launchCmd.Flags().BoolVarP(&serverMode, "server", "s", false, "Start a server instead of a client")
}

var launchCmd = &cobra.Command{
	Use:     "launch",
	Short:   "Launch a minecraft instance",
	Long:    ``, // TODO
	Aliases: []string{"run", "start", "play"},
	Run: func(cmd *cobra.Command, args []string) {
		instance, err := instances.DetectInstance()
		if err != nil {
			logger.Fail("Instance problem: " + err.Error())
		}

		// launch instance
		fmt.Printf("Launching %s\n", instance.Desc())
		if loginData.Mojang == nil {
			logger.Info("You need to sign in with your mojang account to launch minecraft")
			login()
		}
		instance.MojangCredentials = loginData.Mojang

		// Prepare launch
		s := spinner.New(spinner.CharSets[9], 300*time.Millisecond) // Build our new spinner
		s.Prefix = " "
		s.Start()
		s.Suffix = " Preparing launch"

		java := javaBin(instance.Directory)
		if java == "" {
			s.Suffix = " Preparing launch – Downloading java"
			java, err = downloadJava(instance.Directory)
			if err != nil {
				logger.Fail(err.Error())
			}
		}

		// resolve requirements
		if instance.Lockfile == nil || instance.Lockfile.HasRequirements() == false {
			s.Suffix = " Preparing launch – Resolving Requirements"
			instance.ResolveRequirements(context.TODO())
			instance.SaveLockfile()
		}

		mgr := downloadmgr.New()
		mgr.OnProgress = func(p int) {
			s.Suffix = fmt.Sprintf(" Preparing launch – Downloading %v", p) + "%"
		}

		launchManifest, err := instance.GetLaunchManifest()
		if err != nil {
			logger.Fail(err.Error())
		}

		if serverMode != true {
			missingAssets, err := instance.FindMissingAssets(launchManifest)
			if err != nil {
				logger.Fail(err.Error())
			}

			for _, asset := range missingAssets {
				target := filepath.Join(instance.Directory, "assets/objects", asset.UnixPath())
				mgr.Add(downloadmgr.NewHTTPItem(asset.DownloadURL(), target))
			}
		}

		missingLibs, err := instance.FindMissingLibraries(launchManifest)
		if err != nil {
			logger.Fail(err.Error())
		}

		for _, lib := range missingLibs {
			target := filepath.Join(instance.Directory, "libraries", lib.Filepath())
			mgr.Add(downloadmgr.NewHTTPItem(lib.DownloadURL(), target))
		}

		if err = mgr.Start(context.TODO()); err != nil {
			logger.Fail(err.Error())
		}

		s.Suffix = " Downloading dependencies"
		if err := instance.EnsureDependencies(context.TODO()); err != nil {
			logger.Fail(err.Error())
		}

		s.Stop()

		// TODO: This is just a hack
		if serverMode == true {
			launchManifest.MainClass = strings.Replace(launchManifest.MainClass, "Client", "Server", -1)
		}

		fmt.Println("\nLaunching Minecraft …")
		opts := &instances.LaunchOptions{
			LaunchManifest: launchManifest,
			SkipDownload:   true,
			Java:           java,
			Server:         serverMode,
		}
		err = instance.Launch(opts)
		if err != nil {
			logger.Fail(err.Error())
		}
	},
}

func javaBin(dir string) string {
	localJava, err := ioutil.ReadDir(filepath.Join(dir, "java"))

	if err == nil && len(localJava) != 0 {
		return filepath.Join(dir, "java", localJava[0].Name(), "bin/java")
	}

	return ""
	// TODO: check if local java is installed
	// cmd := exec.Command("java", cmdArgs...)

	// // TODO: detatch from process
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	// err = cmd.Run()
}

func downloadJava(dir string) (string, error) {
	url := ""
	ext := ".tar.gz"

	localJava := filepath.Join(dir, "java")
	os.MkdirAll(localJava, os.ModePerm)
	switch runtime.GOOS {
	case "linux":
		url = "https://github.com/AdoptOpenJDK/openjdk8-binaries/releases/download/jdk8u212-b03/OpenJDK8U-jre_x64_linux_hotspot_8u212b03.tar.gz"
	case "windows":
		ext = ".zip"
		url = "https://github.com/AdoptOpenJDK/openjdk8-binaries/releases/download/jdk8u212-b03/OpenJDK8U-jre_x64_windows_hotspot_8u212b03.zip"
	case "osx":
		url = "https://github.com/AdoptOpenJDK/openjdk8-binaries/releases/download/jdk8u212-b03/OpenJDK8U-jre_x64_mac_hotspot_8u212b03.tar.gz"
	}
	res, err := http.Get(url)
	target, err := ioutil.TempFile("", "minepkg-java.*"+ext)

	if err != nil {
		return "", err
	}
	_, err = io.Copy(target, res.Body)
	if err != nil {
		return "", err
	}

	err = archiver.Unarchive(target.Name(), localJava)
	if err != nil {
		return "", err
	}

	return javaBin(dir), nil
}

func init() {
	// launchCmd.Flags().StringVarP(&version, "run-version", "r", "", "Version to start. Uses the latest compatible if not present")
}
