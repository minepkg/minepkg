package pack

import "github.com/fiws/minepkg/pkg/manifest"

// RepackageFile takes the given zip file, recreates it without compression
// and optionally injects the manifest specified.
// The compression is removed to help with deduplication – especially on IPFS.
// The compression also is almost negabile for mod jars
func RepackageFile(file string, manifest *manifest.Manifest) {
	panic("not implemented")
}
