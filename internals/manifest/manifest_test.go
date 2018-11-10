package manifest

import (
	"io/ioutil"
	"testing"
)

func TestAddingDepdendencies(t *testing.T) {
	t.Log("init")
	manifest := New()

	mod := ResolvedMod{Slug: "test", DownloadURL: "https://minepkg.io/fiws/test"}

	t.Log("1")
	manifest.AddDependency(&mod)
	t.Log("2")

	t.Log(manifest)

	// t.Log(manifest.String())
	b := []byte(manifest.String())
	ioutil.WriteFile("minepkg.toml", b, 5550)
}
