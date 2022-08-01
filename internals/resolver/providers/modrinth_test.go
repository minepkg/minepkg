package providers

import (
	"context"
	"testing"

	"github.com/minepkg/minepkg/internals/modrinth"
	"github.com/minepkg/minepkg/pkg/manifest"
)

func TestModrinthProvider_Resolve(t *testing.T) {
	provider := ModrinthProvider{modrinth.New()}
	res, err := provider.Resolve(context.Background(), &Request{
		Dependency: &manifest.InterpretedDependency{
			Source: "fabric-api@4XRtXhtL",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Log(res.Lock().Name)
}
