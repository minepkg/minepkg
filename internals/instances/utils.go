package instances

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func extractNative(jar string, target string) error {
	r, err := zip.OpenReader(jar)
	if err != nil {
		return err
	}

	log.Printf("Extracting natives from %s to %s\n", jar, target)
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
		defer rc.Close()
		f, err := os.Create(filepath.Join(target, f.Name))
		if err != nil {
			return err
		}
		defer f.Close()

		io.Copy(f, rc)
	}
	return nil
}

// stolen from https://github.com/mholt/archiver/v3/blob/e4ef56d48eb029648b0e895bb0b6a393ef0829c3/archiver.go#L110-L119
func sanitizeExtractPath(filePath string, destination string) error {
	// to avoid zip slip (writing outside of the destination), we resolve
	// the target path, and make sure it's nested in the intended
	// destination, or bail otherwise.
	destPath := filepath.Join(destination, filePath)
	if !strings.HasPrefix(destPath, destination) {
		return fmt.Errorf("%s: illegal file path", filePath)
	}
	return nil
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
		cErr := out.Close()
		if err == nil {
			err = cErr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}
