package electron

import (
	"fmt"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/video/monitor"
)

const (
	Title = "UniteHUD Overlay"
)

var (
	window *astilectron.Window
	app    *astilectron.Astilectron
)

func Close() {
	defer func() {
		app, window = nil, nil
	}()

	if window != nil {
		notify.System("Closing %s", Title)

		err := window.Destroy()
		if err != nil {
			notify.Warn("Failed to close %s (%v)", Title, err)
		}
	}

	if app != nil {
		notify.Debug("Closing Astilectron")

		err := app.Quit()
		if err != nil {
			notify.Warn("Failed to close %s Controller (%v)", Title, err)
		}
	}
}

func IsOpen() bool {
	return app != nil && window != nil
}

func Open() error {
	if IsOpen() {
		return nil
	}

	if !config.Current.HUDOverlay {
		return nil
	}

	notify.System("Opening %s", Title)

	err := openApp()
	if err != nil {
		return err
	}

	return openWindow()
}

func openApp() error {
	var err error

	app, err = astilectron.New(nil, astilectron.Options{
		AppName:            Title,
		BaseDirectoryPath:  ".",
		DataDirectoryPath:  "./assets/electron",
		AppIconDefaultPath: "./assets/icon/icon.png",
	})
	if err != nil {
		return fmt.Errorf("Failed to start app for %s (%v)", Title, err)
	}

	app.HandleSignals()

	app.On(astilectron.EventNameAppCrash, resetWindow)
	app.On(astilectron.EventNameAppCmdQuit, resetWindow)
	app.On(astilectron.EventNameAppClose, resetWindow)

	return app.Start()
}

func openWindow() error {
	errq := make(chan error)

	go func() {
		url := `www/UniteHUD Client.html`
		if config.Current.Profile == config.ProfileBroadcaster {
			url = "www/UniteHUD Broadcaster.html"
		}

		area := monitor.MainResolution()
		w, h := area.Max.X, area.Max.Y

		var err error

		window, err = app.NewWindow(url, &astilectron.WindowOptions{
			Title:     astikit.StrPtr(Title),
			Width:     astikit.IntPtr(w),
			Height:    astikit.IntPtr(h),
			MaxWidth:  astikit.IntPtr(w),
			MaxHeight: astikit.IntPtr(h),
			MinWidth:  astikit.IntPtr(w),
			MinHeight: astikit.IntPtr(h),

			Fullscreen:  astikit.BoolPtr(true),
			Minimizable: astikit.BoolPtr(false),
			Resizable:   astikit.BoolPtr(false),
			Center:      astikit.BoolPtr(true),
			Closable:    astikit.BoolPtr(true),

			Transparent: astikit.BoolPtr(true),
			AlwaysOnTop: astikit.BoolPtr(true),

			EnableLargerThanScreen: astikit.BoolPtr(false),
			Focusable:              astikit.BoolPtr(true),
			Frame:                  astikit.BoolPtr(false),
			HasShadow:              astikit.BoolPtr(true),
			Icon:                   astikit.StrPtr("./assets/icon/icon_browser.png"),

			Show: astikit.BoolPtr(true),

			WebPreferences: &astilectron.WebPreferences{
				DevTools:   astikit.BoolPtr(global.DebugMode),
				Images:     astikit.BoolPtr(true),
				Javascript: astikit.BoolPtr(true),
			},
		})
		if err != nil {
			notify.Error("Failed to open %s (%v)", Title, err)
			errq <- err
			return
		}

		waitq := make(chan bool)
		app.On(astilectron.EventNameAppEventReady, func(e astilectron.Event) (deleteListener bool) {
			return <-waitq
		})

		err = window.Create()
		if err != nil {
			errq <- fmt.Errorf("Failed to open %s (%v)", Title, err)
			return
		}

		// window.OpenDevTools()

		close(errq)
		close(waitq)

		app.Wait()
	}()

	return <-errq
}

func resetWindow(e astilectron.Event) (deleteListener bool) {
	if config.Current.Window == config.MainDisplay {
		return true
	}

	notify.SystemWarn("%s closed by user, display has been set to %s", Title, config.MainDisplay)
	config.Current.Window = config.MainDisplay

	return true
}
