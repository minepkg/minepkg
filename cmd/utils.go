package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/minepkg/minepkg/internals/commands"
	"github.com/minepkg/minepkg/internals/instances"
)

// MinepkgMapping is a server mapping (very unfinished)
type MinepkgMapping struct {
	Platform string `json:"platform"`
	Modpack  string `json:"modpack"`
}

// HumanUint32 returns the number in a human readable format
func HumanUint32(num uint32) string {
	switch {
	case num >= 1000000000:
		return fmt.Sprintf("%v B", num/1000000000)
	case num >= 1000000:
		return fmt.Sprintf("%v M", num/1000000)
	case num >= 1000:
		return fmt.Sprintf("%v K", num/1000)
	}
	return fmt.Sprintf("%v", num)
}

// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. Otherwise, attempt to create a hard link
// between the two files. If that fail, copy the file contents from src to dst.
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	if err = os.Link(src, dst); err == nil {
		return
	}
	err = copyFileContents(src, dst)
	return
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

func cmdTerminalOutput(b *exec.Cmd) {
	b.Stderr = os.Stderr
	b.Stdout = os.Stdout
}

func getJarFileForInstance(i *instances.Instance) (*instances.MatchedJar, error) {
	jars, err := i.FindModJar()
	if err != nil {
		return nil, err
	}

	if i.Manifest.Dev.Jar != "" {
		fmt.Printf("Searching according to your pattern \"%s\" to find jar file\n", i.Manifest.Dev.Jar)
	}
	if len(jars) > 1 && i.Manifest.Dev.Jar == "" {
		text := fmt.Sprintf("Found multiple jar files. Using %s", jars[0].Name())
		fmt.Println(commands.StyleWarnBox.Render(text))
		fmt.Println(" Checkout https://preview.minepkg.io/docs/manifest#devjar if you want to use a different file.")
	} else {
		text := fmt.Sprintf("Using jar: %s", jars[0].Path())
		fmt.Println(commands.StyleInfoBox.Render(text))
	}

	return &jars[0], nil
}
