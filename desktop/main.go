package main

import (
	"embed"
	"log"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var frontendAssets embed.FS

func main() {
	desktopService := NewApplicationService()
	app := application.New(application.Options{
		Name:        "CARVION Studio",
		Description: "Desktop workspace for synthetic vehicle multimedia analysis",
		Services: []application.Service{
			application.NewService(desktopService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(frontendAssets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "CARVION Studio",
		Width:            1440,
		Height:           940,
		MinWidth:         1120,
		MinHeight:        720,
		BackgroundColour: application.NewRGB(9, 13, 16),
		URL:              "/",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 54,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
	})

	if err := app.Run(); err != nil {
		log.Printf("CARVION Studio stopped: %v", err)
		os.Exit(1)
	}
}
