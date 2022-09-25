package modrinth_test

import (
	"context"
	"fmt"

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
