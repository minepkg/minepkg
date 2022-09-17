package downloadmgr

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

// ErrInvalidStatusCode is returned if the status code was not 200
var ErrInvalidStatusCode = errors.New("status code was not 200")

type ErrFailedAttempt struct {
	err error
}

func (e *ErrFailedAttempt) Error() string {
	return e.err.Error()
}

// DownloadManager includes a queue to download
type DownloadManager struct {
	queue      []*Item
	OnProgress func(p int)
}

type Item struct {
	downloader  Downloader
	lastErr     error
	attempts    uint16
	maxAttempts uint16
}

// Add adds a new item to the queue
func (d *DownloadManager) Add(i Downloader) {
	d.queue = append(d.queue, &Item{
		downloader:  i,
		maxAttempts: 12,
	})
}

// Start starts the download queue
func (d *DownloadManager) Start(ctx context.Context) error {
	sem := make(chan int, 16)
	cErr := make(chan error)

	if d.queue == nil {
		return nil
	}

	go func() {
		for _, item := range d.queue {
			sem <- 1
			go func(item *Item, cErr chan error) {
				for {
					time.Sleep(time.Duration(item.attempts*item.attempts) * time.Second)
					err := item.downloader.Download(ctx)
					if err == nil {
						cErr <- nil
						break
					}
					item.lastErr = err

					item.attempts += 1
					if item.attempts >= item.maxAttempts {
						cErr <- err
						break
					} else {
						cErr <- &ErrFailedAttempt{err}
					}
				}
				<-sem
			}(item, cErr)
		}
	}()

	// var maybeErr error
	var attemptType *ErrFailedAttempt
	for i := 0; i < len(d.queue); i++ {
		maybeErr := <-cErr
		if maybeErr != nil {
			if errors.As(maybeErr, &attemptType) {
				i--
				fmt.Fprintf(os.Stderr, "! %s\n", maybeErr.Error())
			} else {
				return maybeErr
			}
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

// New creates a new downloadmgr
func New() *DownloadManager {
	return &DownloadManager{}
}
