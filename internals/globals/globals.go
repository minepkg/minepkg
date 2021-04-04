package globals

import (
	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/credentials"
	"github.com/minepkg/minepkg/internals/mojang"
)

var (
	CredStore    *credentials.Store
	ApiClient    = api.New()
	MojangClient = mojang.New()
)
