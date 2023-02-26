package autocomplete

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/minepkg/minepkg/internals/api"
)

func TestAutocomplete(t *testing.T) {
	// create temporary cache dir
	dir := t.TempDir()

	// minepkg api client
	client := api.New()

	// create a new AutoCompleter
	completer := AutoCompleter{
		CacheDir: dir,
		Client:   client,
	}

	// fetch the projects
	projects, err := completer.GetProjects(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// should be at least 1 project
	if len(projects) == 0 {
		t.Fatal("no projects found")
	}

	// check if cache file was created
	_, err = os.Stat(completer.cacheFile())
	if err != nil {
		t.Fatal(err)
	}

	// ensure last fetch was set
	if completer.storage.LastFetch.IsZero() {
		t.Fatal("last fetch was not set")
	}

	// set last fetch to now, so we can test that it doesn't fetch again
	now := time.Now()
	completer.storage.LastFetch = now

	// get projects again
	projects, err = completer.GetProjects(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// should be at least 1 project
	if len(projects) == 0 {
		t.Fatal("no projects found")
	}

	// should not have fetched from the api
	if completer.storage.LastFetch != now {
		t.Fatal("should have fetched from the api")
	}

	// put LastFetch 20 seconds in the past to test if it fetches again
	completer.storage.LastFetch = now.Add(-20 * time.Second)

	// get projects again
	projects, err = completer.GetProjects(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// should be at least 1 project
	if len(projects) == 0 {
		t.Fatal("no projects found")
	}

	// should have fetched from the api
	if completer.storage.LastFetch == now {
		t.Fatal("should have fetched from the api")
	}
}

func TestAutocompleteCold(t *testing.T) {
	// create temporary cache dir
	dir := t.TempDir()

	// minepkg api client
	client := api.New()

	newCompleter := func() *AutoCompleter {

		// create a new AutoCompleter
		completer := AutoCompleter{
			CacheDir: dir,
			Client:   client,
		}
		return &completer
	}

	completer1 := newCompleter()
	// fetch the projects
	projects, err := completer1.GetProjects(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// should be at least 1 project
	if len(projects) == 0 {
		t.Fatal("no projects found")
	}

	// check if cache file was created
	c1Stats, err := os.Stat(completer1.cacheFile())
	if err != nil {
		t.Fatal(err)
	}

	// ensure last fetch was set
	if completer1.storage.LastFetch.IsZero() {
		t.Fatal("last fetch was not set")
	}

	// get projects again with a new completer
	completer2 := newCompleter()

	// get projects again
	projects, err = completer2.GetProjects(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// should be at least 1 project
	if len(projects) == 0 {
		t.Fatal("no projects found")
	}

	c2stats, err := os.Stat(completer2.cacheFile())
	if err != nil {
		t.Fatal(err)
	}

	// check that the cache file was not recreated
	if c1Stats.ModTime() != c2stats.ModTime() {
		t.Log(c1Stats.ModTime())
		t.Log(c2stats.ModTime())
		t.Fatal("cache file was recreated")
	}
}
