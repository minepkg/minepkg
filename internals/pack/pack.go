package pack

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/fiws/minepkg/pkg/manifest"
)

// Reader for a zip (or jar) file that may contain a minepkg.toml
type Reader struct {
	zipReader *zip.Reader
}

// Manifest returns the mod manifest if any
func (p *Reader) Manifest() *manifest.Manifest {
	var manFile *zip.File
	for _, file := range p.Files() {
		if file.Name == "minepkg.toml" {
			manFile = file
			break
		}
	}
	manReader, err := manFile.Open()
	if err != nil {
		panic(err)
	}
	manBuf, err := ioutil.ReadAll(manReader)
	if err != nil {
		panic(err)
	}

	var parsedManifest manifest.Manifest
	toml.Unmarshal(manBuf, &parsedManifest)
	return &parsedManifest
}

// Files returns all contained files of the underlying zip/jar file
func (p *Reader) Files() []*zip.File {
	return p.zipReader.File
}

// ExtractModpack will extract everything in this zipfile to `dest` but will
// NOT overwrite existing savefiles
func (p *Reader) ExtractModpack(dest string) error {
	zipReader := p.zipReader

	skipPrefixes := []string{}
	createdDirs := make(map[string]interface{})

outer:
	for _, f := range zipReader.File {

		// make sure zip only contains valid paths
		if err := sanitizeExtractPath(f.Name, dest); err != nil {
			return err
		}

		// get a relative path – used for name matching and stuff
		relative, err := filepath.Rel(dest, filepath.Join(dest, f.Name))
		if err != nil {
			return err
		}

		// skipping already created save directories
		for _, skip := range skipPrefixes {
			if strings.HasPrefix(relative, skip) {
				continue outer
			}
		}

		// not sure if this is optimal...
		if f.FileInfo().IsDir() {
			continue outer
		}

		relativeDir := filepath.Dir(relative)
		// TODO: is this also / on windows?
		dirs := strings.Split(relativeDir, string(filepath.Separator))

		for n := range dirs {
			// this gets us `saves`, `saves/test-world`, `saves/test-world/DIM1` etc.
			dir := strings.Join(dirs[0:n+1], string(filepath.Separator))

			// see if we already created that dir. skip creating in that case
			if _, ok := createdDirs[dir]; ok {
				continue
			}

			err := os.Mkdir(filepath.Join(dest, dir), os.ModePerm)
			createdDirs[dir] = nil

			switch {
			case err != nil && !os.IsExist(err):
				// unknown error, return it
				return err
			case err != nil && strings.HasPrefix(dir, "saves") && dir != "saves":
				// we tried to create a save dir (eg, `saves/test-world`) and it already exists, exclude it
				skipPrefixes = append(skipPrefixes, dir)
				continue outer
			}
		}

		// all directories for this file are here, we can finally copy the file
		rc, err := f.Open()
		if err != nil {
			return err
		}
		target, err := os.Create(filepath.Join(dest, f.Name))
		if err != nil {
			return err
		}

		_, err = io.Copy(target, rc)
		if err != nil {
			return err
		}

		rc.Close()
	}
	return nil
}

// NewReader returns a Package from a `io.ReaderAt`
func NewReader(reader io.ReaderAt, size int64) *Reader {
	zipReader, err := zip.NewReader(reader, size)
	if err != nil {
		panic(err)
	}
	return &Reader{zipReader}
}

// PackageFile is a local zip (or jar) file that may contain a minepkg.toml
type PackageFile struct {
	*os.File
	*Reader
}

// Open will open the package (zip or jar) file specified by name and return a PackageFile.
func Open(filePath string) (*PackageFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	fStats, err := file.Stat()
	if err != nil {
		return nil, err
	}

	return &PackageFile{
		file,
		NewReader(file, fStats.Size()),
	}, nil
}

// stolen from https://github.com/mholt/archiver/v3/blob/e4ef56d48eb029648b0e895bb0b6a393ef0829c3/archiver.go#L110-L119
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
