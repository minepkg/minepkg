package minecraft_test

import (
	"fmt"

	"github.com/minepkg/minepkg/internals/minecraft"
)

func ExampleMergeManifests() {
	source := &minecraft.LaunchManifest{
		ID: "1.18.2",
		Libraries: []minecraft.Library{
			{Name: "commons-logging:commons-logging:1.2"},
		},
	}
	manifest2 := &minecraft.LaunchManifest{
		ID: "overwritten",
		Libraries: []minecraft.Library{
			{Name: "io.minepkg.test:lib:1.0.0"},
		},
	}
	// MergeManifest modifies the source manifest
	minecraft.MergeManifests(source, manifest2)

	// Print the modified source manifest
	fmt.Println("ID:", source.ID)
	fmt.Println("Libraries:")
	for _, arg := range source.Libraries {
		fmt.Println(" - ", arg.Name)
	}
	// Output:
	// ID: overwritten
	// Libraries:
	//  -  commons-logging:commons-logging:1.2
	//  -  io.minepkg.test:lib:1.0.0
}
