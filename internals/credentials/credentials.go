package credentials

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

var NoKeyRingMode = false

// Store stores any credentials
type Store struct {
	globalDir string
	// Name is the name of this credentials store
	Name string
}

// New creates a new credentials store
func New(globalDir string, name string) *Store {
	store := &Store{globalDir: globalDir, Name: name}
	return store
}

// Get tries to find existing credentials
func (s *Store) Get(target interface{}) error {
	// find minepkg credentials
	minepkgAuth, err := keyring.Get("minepkg_auth_data", s.Name)
	switch err {
	case nil:
		return json.Unmarshal([]byte(minepkgAuth), target)
	case keyring.ErrNotFound:
		// empty result
		return nil
	default:
		log.Println("Could not use key store, will default to file store for secrets.", err)
		NoKeyRingMode = true
		return s.findFromFiles(target)
	}
}

func (s *Store) localFilename() string {
	return "minepkg-credentials-" + s.Name + ".json"
}

// findFromFiles is the same as Find but reads from plain files instead
func (s *Store) findFromFiles(target interface{}) error {
	return s.readCredentialFile(s.localFilename(), &target)
}

// Set sets the credentials and persists it to disk
func (s *Store) Set(data interface{}) error {
	// maybe set in struct?
	jsonBlob, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if NoKeyRingMode {
		return s.writeCredentialFile(s.localFilename(), jsonBlob)
	}
	return keyring.Set("minepkg_auth_data", s.Name, string(jsonBlob))
}

// readCredentialFile is a helper that reads a file from the minepkg config dir
func (s *Store) readCredentialFile(location string, v interface{}) error {
	file := filepath.Join(s.globalDir, location)
	rawCreds, err := ioutil.ReadFile(file)
	switch {
	case err == nil:
		// parse json
		if err := json.Unmarshal(rawCreds, &v); err != nil {
			// ignore error. this usually happens if the disk runs out of space
			// by ignoring it we can let the user login again after sufficient
			// space exists again
			fmt.Printf("WARNING: A credentials file was corrupted. ignoring")
			return nil
		}
		// parsed as expected
		return nil
	case os.IsNotExist(err):
		// no file is fine
		return nil
	default:
		// everything else is not
		return err
	}
}

// writeCredentialFile is a helper that writes a file to the minepkg config dir
func (s *Store) writeCredentialFile(location string, content []byte) error {
	os.MkdirAll(s.globalDir, os.ModePerm)
	credFile := filepath.Join(s.globalDir, location)
	return ioutil.WriteFile(credFile, content, 0700)
}
