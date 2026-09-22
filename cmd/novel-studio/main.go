package main

import (
	"github.com/voocel/ainovel-cli/desktop/frontend"
	"github.com/voocel/ainovel-cli/internal/studio/bridge"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"log"
)

func main() {
	app := &bridge.App{}
	if err := wails.Run(&options.App{
		Title: "Novel Studio", Width: 1440, Height: 900, MinWidth: 1000, MinHeight: 650,
		BackgroundColour: &options.RGBA{R: 15, G: 19, B: 28, A: 255},
		AssetServer:      &assetserver.Options{Assets: frontend.Assets},
		OnStartup:        app.Startup, Bind: []interface{}{app},
	}); err != nil {
		log.Fatal(err)
	}
}
