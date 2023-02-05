package api_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/minepkg/minepkg/internals/api"
)

func TestProjectType(t *testing.T) {
	// ensure that a marshalled project does not include the `stats` field
	project := api.Project{
		Name: "test",
		Type: "mod",
	}
	data, err := json.Marshal(project)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "stats") {
		t.Fatal("stats field found in marshalled project")
	}
}