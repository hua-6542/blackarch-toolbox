package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"blackarch-toolbox/internal/app"
	"blackarch-toolbox/internal/config"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "配置加载失败:", err)
		os.Exit(1)
	}
	dbPath := filepath.Join(dataHome(), "blackarch-toolbox", "toolbox.db")
	application, err := app.New(cfg, dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "初始化失败:", err)
		os.Exit(1)
	}
	defer application.DB.Close()
	err = wails.Run(&options.App{
		Title:            "BlackArch ToolBox",
		Width:            1280,
		Height:           800,
		MinWidth:         1000,
		MinHeight:        640,
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 24, A: 1},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: func(ctx context.Context) {
			application.SetEventEmitter(func(event string, data ...any) {
				runtime.EventsEmit(ctx, event, data...)
			})
		},
		Bind: []interface{}{application},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "运行失败:", err)
		os.Exit(1)
	}
}

func dataHome() string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share")
}
