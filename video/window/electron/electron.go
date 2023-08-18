package electron

import (
	"fmt"
	"time"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	"github.com/pidgy/unitehud/config"
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
		notify.System("Closing %s Controller", Title)

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

	notify.System("Opening %s", Title)

	err := openApp()
	if err != nil {
		return err
	}

	err = openWindow()
	if err != nil {
		return err
	}

	return nil
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
		url := `./www/UniteHUD Client.html`
		if config.Current.Profile == config.ProfileBroadcaster {
			url = "./www/UniteHUD Broadcaster.html"
		}

		area := monitor.MainResolution()
		w, h := area.Max.X, area.Max.X

		var err error

		window, err = app.NewWindow(url, &astilectron.WindowOptions{
			Width:     astikit.IntPtr(w),
			Height:    astikit.IntPtr(h),
			MaxWidth:  astikit.IntPtr(w),
			MaxHeight: astikit.IntPtr(h),
			MinWidth:  astikit.IntPtr(w),
			MinHeight: astikit.IntPtr(h),

			Fullscreen:  astikit.BoolPtr(true),
			Minimizable: astikit.BoolPtr(false),
			Resizable:   astikit.BoolPtr(true),
			Center:      astikit.BoolPtr(true),
			Closable:    astikit.BoolPtr(true),

			Transparent: astikit.BoolPtr(true),
			AlwaysOnTop: astikit.BoolPtr(true),

			EnableLargerThanScreen: astikit.BoolPtr(false),
			Focusable:              astikit.BoolPtr(false),
			Frame:                  astikit.BoolPtr(false),
			HasShadow:              astikit.BoolPtr(false),
			Icon:                   astikit.StrPtr("./assets/icon/icon_browser.png"),

			Show: astikit.BoolPtr(true),

			WebPreferences: &astilectron.WebPreferences{},
		})
		if err != nil {
			errq <- fmt.Errorf("Failed to create %s (%v)", Title, err)
			return
		}

		waitq := make(chan bool)
		app.On(astilectron.EventNameAppEventReady, func(e astilectron.Event) (deleteListener bool) {
			println("waiting")
			time.Sleep(time.Second * 3)
			return <-waitq
		})

		err = window.Create()
		if err != nil {
			errq <- fmt.Errorf("Failed to open %s (%v)", Title, err)
			return
		}

		close(errq)

		waitq <- true

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
