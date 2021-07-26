package java

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	archiver "github.com/mholt/archiver/v3"
)

type Java struct {
	dir              string
	asset            *AdoptAsset
	needsDownloading bool
}

func (j *Java) Bin() string {
	var bin string
	switch runtime.GOOS {
	case "windows":
		bin = "bin/java.exe"
	case "darwin": // macOS
		bin = "Contents/Home/bin/java"
	default:
		bin = "bin/java"
	}

	return filepath.Join(j.dir, bin)
}

func (j *Java) NeedsDownloading() bool {
	return j.needsDownloading
}

// Update downloads or updates this java version
func (j *Java) Update(ctx context.Context) error {
	// remove everything
	if err := os.RemoveAll(j.dir); err != nil {
		return err
	}
	os.RemoveAll(j.dir + ".tmp")

	// download archive
	archive, err := j.download(ctx)
	if err != nil {
		return err
	}
	defer os.Remove(archive.Name()) // remove temporary download

	// ugly hack to get root directory. it's something like "jdk8u292-b10-jre"
	rootDirName := ""
	err = archiver.Walk(archive.Name(), func(f archiver.File) error {
		if f.IsDir() {
			rootDirName = f.Name()
			return archiver.ErrStopWalk
		}
		return nil
	})
	if err != nil {
		return err
	}

	// extract the whole archive. avoids https://github.com/mholt/archiver/issues/289
	if err := archiver.Unarchive(archive.Name(), j.dir+".tmp"); err != nil {
		return err
	}
	// another ugly hack because archiver can not extract without creating a directory
	// and doing in manually with archiver.Walk is a pain. PRs welcome …
	if err := os.Rename(filepath.Join(j.dir+".tmp", rootDirName), j.dir); err != nil {
		return err
	}

	// we moved the rootDir "jdk8u292-b10-jre" to j.dir ("8-jre-openj9")
	// but the leftover .tmp dir is still here. should be empty, but it's not
	// for macos archives
	if err := os.RemoveAll(j.dir + ".tmp"); err != nil {
		return err
	}

	// finally write the asset file
	asset, err := os.Create(filepath.Join(j.dir, "asset.json"))
	if err != nil {
		return err
	}

	if err := json.NewEncoder(asset).Encode(j.asset); err != nil {
		return err
	}

	j.needsDownloading = false
	return nil
}

func (j *Java) download(ctx context.Context) (*os.File, error) {
	url := j.downloadURL()
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	ext := ".tar.gz"
	if !strings.HasSuffix(url, ".tar.gz") {
		ext = filepath.Ext(url)
	}
	archive, err := ioutil.TempFile("", "minepkg-java.*"+ext)
	if err != nil {
		return nil, err
	}
	defer archive.Close()

	_, err = io.Copy(archive, res.Body)
	if err != nil {
		return nil, err
	}

	return archive, nil
}

func (j *Java) downloadURL() string {
	return j.asset.Binaries[0].Package.Link
}
