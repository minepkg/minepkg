package instances

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CopyOverwrites copies everything from the instance dir (with a few exceptions) to the minecraft dir
// exceptions are: the minecraft folder itself and minepkg related files (manifest & lockfile)
func (i *Instance) CopyOverwrites() error {
	err := filepath.Walk(i.OverwritesDir(), func(fullPath string, info os.FileInfo, err error) error {
		// get a relative path
		path, err := filepath.Rel(i.OverwritesDir(), fullPath)
		if err != nil {
			return err
		}

		switch {
		case path == ".":
			// skip root
			return nil
		case strings.HasPrefix(path, "excluded") == true:
			// skip everything starting with "excluded"
			return filepath.SkipDir
		case path == "minecraft" || path == "saves":
			// skip the minecraft directory
			// and skip the saves dir. it is handled using `Instance.CopyLocalSaves`
			return filepath.SkipDir
		case info.IsDir() && strings.HasPrefix(path, "."):
			// skip any hidden dirs or files like ".git"
			return filepath.SkipDir

		case path == "minepkg.toml" || path == "minepkg-lock.toml" || strings.HasPrefix(strings.ToLower(path), "readme"):
			// do not skip the root directory, but skip those files
			return nil
		}

		destPath := filepath.Join(i.McDir(), path)

		// create directory
		if info.IsDir() {
			err = os.Mkdir(destPath, os.ModePerm)
			if err != nil && os.IsExist(err) {
				// existing dirs are fine
				return nil
			}
			// return to skip copy files logic, because this is a directory
			return err
		}

		// not a directory – copy file
		src, err := os.Open(fullPath)
		if err != nil {
			return err
		}
		dest, err := os.Create(destPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(dest, src)
		return err
	})
	if err != nil {
		return err
	}

	return nil
}

// CopyLocalSaves copies saves from the instance dir to the minecraft dir
// this should run BEFORE `Instance.LinkDependencies` because local saves should always
// have the highest priority
func (i *Instance) CopyLocalSaves() error {
	os.MkdirAll(filepath.Join(i.McDir(), "saves"), os.ModePerm)
	start := filepath.Join(i.Directory, "saves")
	err := filepath.Walk(start, func(aPath string, info os.FileInfo, err error) error {
		// get a relative path
		path, err := filepath.Rel(start, aPath)
		if err != nil {
			return err
		}

		// skip root
		if path == "." {
			return nil
		}

		destPath := filepath.Join(i.McDir(), "saves", path)

		// create directory
		if info.IsDir() {
			err = os.Mkdir(destPath, os.ModePerm)
			// very important: do not overwrites saves if any exist
			if err != nil && os.IsExist(err) {
				return filepath.SkipDir
			}
			// return to skip copy files logic, because this is a directory. (err should be nil)
			return err
		}

		// not a directory – copy file
		src, err := os.Open(aPath)
		if err != nil {
			return err
		}
		dest, err := os.Create(destPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(dest, src)
		return err
	})
	if err != nil {
		return err
	}

	return nil
}
