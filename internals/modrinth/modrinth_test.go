package modrinth_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/minepkg/minepkg/internals/modrinth"
)

// Basic usage
func Example() {
	client := modrinth.New(nil)

	// Get the latest version of the fabric mod.
	versions, err := client.ListProjectVersion(context.Background(), "fabric-api", nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(versions[0].ProjectID)
	// Output: P7dR8mSH
}

func ExampleClient_GetVersion() {
	client := modrinth.New(nil)

	// get one version by its id
	version, err := client.GetVersion(context.Background(), "4XRtXhtL")
	if err != nil {
		panic(err)
	}
	fmt.Println(version.ProjectID)
	// Output: P7dR8mSH
}

func ExampleClient_GetVersionFile() {
	client := modrinth.New(nil)

	// get version with file hash
	version, err := client.GetVersionFile(
		context.Background(),
		"b9ab9ab267f8cdff525f9a8edb26435d3e2455f6",
	)

	if err != nil {
		panic(err)
	}

	fmt.Println(version.ID)
	// Output: 4XRtXhtL
}

// Test project 404
func TestNotFounds(t *testing.T) {
	client := modrinth.New(nil)

	t.Run("project", func(t *testing.T) {
		t.Parallel()
		_, err := client.GetProject(context.Background(), "test-case-should-never-exist-404-404-404")
		if err != modrinth.ErrProjectNotFound {
			t.Fatalf("expected ErrProjectNotFound, got %v", err)
		}
	})

	t.Run("version", func(t *testing.T) {
		t.Parallel()
		_, err := client.GetVersion(context.Background(), "test-case-should-never-exist-404-404-404")
		if err != modrinth.ErrVersionNotFound {
			t.Fatalf("expected ErrVersionNotFound, got %v", err)
		}
	})

	t.Run("file", func(t *testing.T) {
		t.Parallel()
		_, err := client.GetVersionFile(context.Background(), "test-case-should-never-exist-404-404-404")
		if err != modrinth.ErrResourceNotFound {
			t.Fatalf("expected ErrFileNotFound, got %v", err)
		}

		// hash of main.go
		_, err = client.GetVersionFile(context.Background(), "460e9c526fdf8dce833c560e56405c8ec253401b")
		if err != modrinth.ErrResourceNotFound {
			t.Fatalf("expected ErrFileNotFound, got %v", err)
		}
	})
}

// Test ListProjectVersionQuery String()
func TestQueryString(t *testing.T) {
	t.Parallel()
	query := modrinth.ListProjectVersionQuery{
		Loaders:      []string{"fabric"},
		GameVersions: []string{"1"},
	}

	result := query.String()
	expected := "game_versions=%5B%221%22%5D&loaders=%5B%22fabric%22%5D"

	if result != expected {
		t.Fatalf("expected %s, got %s", expected, result)
	}
}
