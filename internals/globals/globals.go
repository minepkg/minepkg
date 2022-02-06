package globals

import (
	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/cmdlog"
)

var (
	ApiClient = api.New()
	Logger    = cmdlog.New()
)
