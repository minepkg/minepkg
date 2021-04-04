package cache

import (
	"os"
	"path/filepath"

	"github.com/minepkg/minepkg/internals/downloadmgr"
	"github.com/minepkg/minepkg/pkg/manifest"
)

// type Cacher interface {
// 	Set()
// }

// Cache is a helper to cache/store files locally
type Cache struct {
	location   string
	downloader *downloadmgr.DownloadManager
}

var defaultStorage = Cache{location: "/tmp", downloader: downloadmgr.New()}

func init() {
	home, _ := os.UserHomeDir()
	globalDir := filepath.Join(home, ".minepkg/cache-v2")
	defaultStorage.location = globalDir
}

// New returns a Storage object
func New(location string) Cache {
	return Cache{
		location:   location,
		downloader: downloadmgr.New(),
	}
}

func (c *Cache) Store(l *manifest.DependencyLock) {
	loc := filepath.Join(c.location, l.ID())
	c.downloader.Add(downloadmgr.NewHTTPItem(l.URL, loc))
}

func (c *Cache) GetPath(l *manifest.DependencyLock) string {
	loc := filepath.Join(c.location, l.ID())
	return loc
}
