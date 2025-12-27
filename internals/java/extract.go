package java

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// extractArchive extracts a zip or tar.gz archive to the destination directory.
// It returns the name of the root directory inside the archive, if one exists.
func extractArchive(src, dest string) (string, error) {
	if strings.HasSuffix(src, ".zip") {
		return extractZip(src, dest)
	}
	// assume tar.gz for everything else for now, as that is what java.go was doing implicitly
	return extractTarGz(src, dest)
}

func extractZip(src, dest string) (string, error) {
	r, err := zip.OpenReader(src)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var rootDir string

	for _, f := range r.File {
		// Detect root dir (first directory found)
		if rootDir == "" && f.FileInfo().IsDir() {
			rootDir = f.Name
		} else if rootDir == "" && strings.Contains(f.Name, "/") {
			// complex case: file in root dir without explicit dir entry
			parts := strings.Split(f.Name, "/")
			if len(parts) > 1 {
				rootDir = parts[0]
			}
		}

		rc, err := f.Open()
		if err != nil {
			return "", err
		}

		path := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			rc.Close()
			return "", fmt.Errorf("%s: illegal file path", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), 0755) // Ensure parent dir exists
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				rc.Close()
				return "", err
			}
			_, err = io.Copy(f, rc)
			f.Close()
			if err != nil {
				rc.Close()
				return "", err
			}
		}
		rc.Close()
	}
	return strings.TrimSuffix(strings.TrimSuffix(rootDir, "/"), "\\"), nil
}

func extractTarGz(src, dest string) (string, error) {
	f, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var rootDir string

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// Detect root dir
		if rootDir == "" && header.Typeflag == tar.TypeDir {
			rootDir = header.Name
		} else if rootDir == "" && strings.Contains(header.Name, "/") {
			parts := strings.Split(header.Name, "/")
			if len(parts) > 1 {
				rootDir = parts[0]
			}
		}

		target := filepath.Join(dest, header.Name)
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return "", fmt.Errorf("%s: illegal file path", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return "", err
			}
		case tar.TypeReg:
			// ensure parent dir exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return "", err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return "", err
			}
			f.Close()
		}
	}
	return strings.TrimSuffix(strings.TrimSuffix(rootDir, "/"), "\\"), nil
}
