package gui

import (
	"context"
	"fmt"

	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/provider"
)

// App struct
type App struct {
	ctx           context.Context
	MinepkgAPI    *api.MinepkgClient
	ProviderStore *provider.Store
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) JoinServer(ip string) {
	fmt.Println("Joining server", ip)

}
