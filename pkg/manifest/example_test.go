package manifest_test

import (
	"fmt"

	"github.com/minepkg/minepkg/pkg/manifest"
	"github.com/pelletier/go-toml"
)

// Unmarshal a toml to a manifest struct
func ExampleManifest_unmarshal() {
	raw := []byte(`
	manifestVersion = 0
	[package]
	name="test-utils"
	version="1.0.0"
`)
	var man manifest.Manifest
	toml.Unmarshal(raw, &man)
	fmt.Println(man.Package.Name)
	// Output:
	// test-utils
}

// Marshal a struct to toml
func ExampleManifest_marshal() {
	manifest := manifest.New()
	manifest.Package.Name = "test-mansion"

	fmt.Println(manifest.String()) // or manifest.Buffer() to get it as a buffer
}
