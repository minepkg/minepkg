package curse

import (
	"fmt"
	"testing"
)

func TestFetchMod(t *testing.T) {
	_, err := FetchMod(231951)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchModFiles(t *testing.T) {
	mod, err := FetchMod(231951)
	if err != nil {
		t.Error(err)
	}

	files, err := FetchModFiles(mod)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(files)
}
