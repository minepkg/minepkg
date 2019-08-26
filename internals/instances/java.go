package instances

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mholt/archiver"
)

// HasJava returns true if the internal java installation has been detected
// for this instance
func (i *Instance) HasJava() bool {
	return i.javaBin() != ""
}

// UseSystemJava sets the instance to use the system java
// instead of the internal installation
func (i *Instance) UseSystemJava() {
	i.javaBinary = "java"
}

// UpdateJava updates the local java installation
func (i *Instance) UpdateJava() error {
	java, err := i.downloadJava()
	if err != nil {
		return err
	}
	i.javaBinary = java
	return nil
}

// javaBin returns the internal java binary
// it caches the path if it finds a java installation
func (i *Instance) javaBin() string {
	if i.javaBinary != "" {
		return i.javaBinary
	}
	javaPath := filepath.Join(i.GlobalDir, "java")
	localJava, err := ioutil.ReadDir(javaPath)

	if err == nil && len(localJava) != 0 {
		bin := "bin/java" // linux. somehow also works with windows
		switch runtime.GOOS {
		case "windows":
			bin = "bin/java.exe"
		case "darwin": // macOS
			bin = "Contents/Home/bin/java"
		}
		i.javaBinary = filepath.Join(javaPath, localJava[0].Name(), bin)
		return i.javaBinary
	}

	return ""
	// TODO: check if local java is installed
	// cmd := exec.Command("java", cmdArgs...)

	// // TODO: detatch from process
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	// err = cmd.Run()
}

// downloadJava downloads the internal java binary
// TODO: version should not be static!
func (i *Instance) downloadJava() (string, error) {
	url := ""
	ext := ".tar.gz"

	localJava := filepath.Join(i.GlobalDir, "java")
	os.MkdirAll(localJava, os.ModePerm)
	switch runtime.GOOS {
	case "linux":
		url = "https://github.com/AdoptOpenJDK/openjdk8-binaries/releases/download/jdk8u212-b03/OpenJDK8U-jre_x64_linux_hotspot_8u212b03.tar.gz"
	case "windows":
		ext = ".zip"
		url = "https://github.com/AdoptOpenJDK/openjdk8-binaries/releases/download/jdk8u212-b03/OpenJDK8U-jre_x64_windows_hotspot_8u212b03.zip"
	case "darwin": // macOS
		url = "https://github.com/AdoptOpenJDK/openjdk8-binaries/releases/download/jdk8u212-b03/OpenJDK8U-jre_x64_mac_hotspot_8u212b03.tar.gz"
	default:
		return "", errors.New("Unknown operating system. Can't download java for it")
	}
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}

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

	// macos tar contains some uneeded stuff that we need to remove
	if runtime.GOOS == "darwin" {
		os.Remove(filepath.Join(localJava, "._jdk8u212-b03-jre"))
	}

	return i.javaBin(), nil
}
