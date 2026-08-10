package main

import (
	"embed"
	"fmt"
	"time"

	"github.com/SafetyCulture/mitti-exporter-ui/internal/version"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	extLogger "github.com/SafetyCulture/mitti-exporter/pkg/logger"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {
	// Create an instance of the app structure
	app := NewApp()

	settingsDir, err := CreateSettingsDirectory()
	if err != nil {
		panic(fmt.Sprintf("failed to get settings directory: %s", err.Error()))
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:     "Mitti Exporter",
		Width:     1080,
		Height:    780,
		MinWidth:  1080,
		MinHeight: 780,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
		LogLevel: logger.INFO,
		Logger:   extLogger.GetExporterLogger(settingsDir),
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
		Mac: &mac.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			About: &mac.AboutInfo{
				Title:   "Mitti Exporter",
				Message: fmt.Sprintf("Version %v\n\nCopyright \u00a9 %v\nSafetyCulture Pty Ltd.\n\nTerms: https://safetyculture.com/legal/terms-and-conditions\nPrivacy: https://safetyculture.com/legal/privacy-policy\n\nSupport: https://help.safetyculture.com/", version.GetVersion(), time.Now().Year()),
				Icon:    icon,
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
