package downloadmgr

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// DownloadManager includes a queue to download
type DownloadManager struct {
	queue      []Downloader
	client     *http.Client
	OnProgress func(p int)
}

// Add adds a new item to the queue
func (d *DownloadManager) Add(i Downloader) {
	d.queue = append(d.queue, i)
}

// Start starts the download queue
func (d *DownloadManager) Start(ctx context.Context) error {
	sem := make(chan int, 16)
	errc := make(chan error)

	if d.queue == nil {
		return nil
	}

	go func() {
		for _, item := range d.queue {
			sem <- 1
			go func(item Downloader, err chan error) {
				err <- item.Download(ctx)
				<-sem
			}(item, errc)
		}
	}()
	// var maybeErr error
	for i := 0; i < len(d.queue); i++ {
		maybeErr := <-errc
		if maybeErr != nil {
			return maybeErr
		}
		if d.OnProgress != nil {
			d.OnProgress(int(float32(i) / float32(len(d.queue)) * 100))
		}
	}
	return nil
}

// Downloader allows downloadmgr to download the file
type Downloader interface {
	Download(ctx context.Context) error
}

// HTTPItem is a URL, target pair with optional properties that will be downloaded
// using http(s)
type HTTPItem struct {
	URL    string
	Target string
	Size   int
	Sha256 string
}

// Download downloads the item to the defined target using http
func (i *HTTPItem) Download(ctx context.Context) error {
	err := os.MkdirAll(filepath.Dir(i.Target), os.ModePerm)
	if err != nil {
		return err
	}
	fileRes, err := http.Get(i.URL)
	// TODO: check status code and all the things!
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
	return &HTTPItem{URL, Target, 0, ""}
}

// New creates a new downloadmgr
func New() *DownloadManager {
	return &DownloadManager{}
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

func checkSha256(sha string, srcPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
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
