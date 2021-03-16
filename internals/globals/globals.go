package globals

import (
	"github.com/fiws/minepkg/internals/api"
	"github.com/fiws/minepkg/internals/credentials"
	"github.com/fiws/minepkg/internals/mojang"
)

var (
	CredStore    *credentials.Store
	ApiClient    = api.New()
	MojangClient = mojang.New()
)
