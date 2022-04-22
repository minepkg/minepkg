package instances

import (
	"log"
	"os"
	"path/filepath"
)

var excludeFromClean = []string{
	"banned-ips.json",
	"banned-players.json",
	"ops.json",
	"saves",
	"screenshots",
	"whitelist.json",
	"world",
}

func isExcluded(name string) bool {
	for _, exclude := range excludeFromClean {
		if name == exclude {
			return true
		}
	}
	return false
}

func (i *Instance) Clean() error {
	directory := i.McDir()

	// remove everything but the "savegames" directory
	dirs, err := os.ReadDir(directory)

	// dir does not exist. this is fine it is "clean" then
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		if isExcluded(dir.Name()) {
			continue
		}
		log.Println("Removing", dir.Name())
		err := os.RemoveAll(filepath.Join(directory, dir.Name()))
		if err != nil {
			return err
		}
	}

	log.Println("Removing manifest & lockfile")
	os.Remove(i.ManifestPath())
	os.Remove(i.LockfilePath())

	return nil
}
