package downloadmgr

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var defaultClient = http.Client{
	Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout:   20 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

// HTTPItem is a URL, target pair with optional properties that will be downloaded
// using http(s)
type HTTPItem struct {
	Client *http.Client
	URL    string
	Target string
	Size   int
	Sha256 string
}

// ErrInvalidSha is returned when the downloaded file's sha256 sum does not match the given sha1
type ErrInvalidSha struct {
	FileName    string
	ExpectedSha string
	ActualSha   string
}

func (e *ErrInvalidSha) Error() string {
	return fmt.Sprintf(
		"File corrupted: %s sha256 is invalid.\n\texpected to be \"%s\"\n\tbut actually is \"%s\"\n",
		e.FileName,
		e.ExpectedSha,
		e.ActualSha,
	)
}

// Download downloads the item to the defined target using http
func (i *HTTPItem) Download(ctx context.Context) error {
	err := os.MkdirAll(filepath.Dir(i.Target), os.ModePerm)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", i.URL, nil)
	if err != nil {
		return err
	}

	client := i.Client
	if client == nil {
		client = &defaultClient
	}

	fileRes, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Error while fetching %s: %w", i.URL, err)
	}

	if fileRes.StatusCode != 200 {
		return fmt.Errorf("invalid status code: %s from %s", fileRes.Status, fileRes.Request.URL)
	}

	dest, err := os.Create(i.Target)
	if err != nil {
		return err
	}
	_, err = io.Copy(dest, fileRes.Body)
	if err != nil {
		return err
	}
	if err := dest.Sync(); err != nil {
		return err
	}

	// check sha if there is one set
	if i.Sha256 != "" {
		if err := checkSha256(i.Sha256, dest.Name()); err != nil {
			return err
		}
	}
	return nil
}

// NewHTTPItem creates a Item to be queued that will download the file using HTTP(S)
func NewHTTPItem(URL string, Target string) *HTTPItem {
	if URL == "" {
		panic("Download URL can not be empty")
	}
	if Target == "" {
		panic("Target can not be empty")
	}
	return &HTTPItem{&defaultClient, URL, Target, 0, ""}
}

func checkSha256(sha string, srcPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	hasher := sha256.New()
	_, err = io.Copy(hasher, src)
	// probably io error during hashing
	if err != nil {
		return err
	}
	actualSha := fmt.Sprintf("%x", hasher.Sum(nil))
	if actualSha != sha {
		// TODO: this can fail! move file to tmp storage first
		os.Remove(src.Name())
		return &ErrInvalidSha{src.Name(), sha, actualSha}
	}
	return nil
}
