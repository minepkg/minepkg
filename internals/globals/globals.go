package globals

import (
	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/auth"
	"github.com/minepkg/minepkg/internals/cmdlog"
	"github.com/minepkg/minepkg/internals/ownhttp"
)

var (
	GlobalDir  string
	Auth       auth.AuthProvider
	HTTPClient = ownhttp.New()
	ApiClient  = api.New()
	Logger     = cmdlog.New()
)
