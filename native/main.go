package main

import (
	_ "embed"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	"github.com/egoist/mygo/ui"
)

//go:embed assets/icon.png
var appIcon []byte

func main() {
	app := newApplication()
	mygo.App.SetName("Data COS Suite")
	mygo.App.SetLanguages("zh-Hans")
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	app.settingsPath = filepath.Join(configDir, "data-cos-suite-mygo", "settings.json")
	if path := os.Getenv("DATA_COS_SETTINGS_PATH"); path != "" {
		app.settingsPath = path
	}
	app.config, err = loadSettings(app.settingsPath)
	app.sidebarWidth = app.config.SidebarWidth
	app.syncCurrentMode()
	if err != nil {
		app.reportError(err.Error())
	}
	app.engine, app.engineError = findEngine()
	if app.engineError != nil {
		app.reportError(app.engineError.Error())
	}
	mygo.App.WhenReady(func() {
		if err := mygo.App.Dock.SetIcon(appIcon); err != nil {
			log.Printf("set application icon: %v", err)
		}
		if runtime.GOOS == "windows" {
			mygo.App.SetMenu(nil)
		} else {
			mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
				{Role: mygo.RoleAppMenu},
				{Role: mygo.RoleEditMenu},
				{Role: mygo.RoleWindowMenu},
			}))
		}
		app.window = mygo.NewWindow(mygo.WindowOptions{Title: "Data COS Suite · 数据提取", Width: 1440, Height: 900, MinWidth: 1000, MinHeight: 720, Maximized: true, Content: ui.View(app.view)})
		if err := app.window.SetIcon(appIcon); err != nil {
			log.Printf("set window icon: %v", err)
		}
		app.wake = app.window.Invalidate
		app.window.OnClosed(func() {
			if app.cancel != nil {
				app.cancel()
			}
			if err := saveSettings(app.settingsPath, app.config); err != nil {
				log.Printf("save settings: %v", err)
			}
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
