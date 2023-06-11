package gui

import (
	"embed"

	"github.com/minepkg/minepkg/internals/api"
	"github.com/minepkg/minepkg/internals/provider"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func Start(api *api.MinepkgClient, providerStore *provider.Store) {
	// Create an instance of the app structure
	app := NewApp()
	app.MinepkgAPI = api
	app.ProviderStore = providerStore

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "minepkg-wails",
		Width:     1024,
		Height:    768,
		MinWidth:  400,
		MinHeight: 300,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 0, B: 0, A: 1},
		OnStartup:        app.startup,
		// Frameless:        true,
		Bind: []interface{}{
			app,
		},
		Linux: &linux.Options{
			WindowIsTranslucent: true,
			WebviewGpuPolicy:    linux.WebviewGpuPolicyOnDemand,
		},
		Mac: &mac.Options{
			WindowIsTranslucent:  true,
			WebviewIsTransparent: true,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
