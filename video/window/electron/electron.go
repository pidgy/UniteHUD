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
	overlayApp    *astilectron.Astilectron
	overlayWindow *astilectron.Window
)

func Close() {
	defer func() {
		overlayApp, overlayWindow = nil, nil
	}()

	if overlayWindow != nil {
		notify.System("Closing %s", Title)

		err := overlayWindow.Destroy()
		if err != nil {
			notify.Warn("Failed to close %s (%v)", Title, err)
		}
	}

	if overlayApp != nil {
		notify.Debug("Closing Astilectron")

		err := overlayApp.Quit()
		if err != nil {
			notify.Warn("Failed to close %s Controller (%v)", Title, err)
		}
	}
}

func IsOpen() bool {
	return overlayApp != nil && overlayWindow != nil
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

var opened = false

func openApp() error {
	if opened {
		return nil
	}
	opened = true

	a, err := astilectron.New(nil, astilectron.Options{
		AppName:            Title,
		BaseDirectoryPath:  ".",
		DataDirectoryPath:  "./assets/electron",
		AppIconDefaultPath: "./assets/icon/icon.png",
		VersionElectron:    astilectron.DefaultVersionElectron,
		VersionAstilectron: astilectron.DefaultVersionAstilectron,
	})
	if err != nil {
		return fmt.Errorf("Failed to start app for %s (%v)", Title, err)
	}
	overlayApp = a

	overlayApp.HandleSignals()

	overlayApp.On(astilectron.EventNameAppCrash, windowClosed)
	overlayApp.On(astilectron.EventNameAppCmdQuit, windowClosed)
	overlayApp.On(astilectron.EventNameAppClose, windowClosed)

	return overlayApp.Start()
}

func openWindow() error {
	errq := make(chan error)

	go func() {
		url := `www/UniteHUD Client.html`
		if config.Current.Profile == config.ProfileBroadcaster {
			url = "www/UniteHUD Broadcaster.html"
		}

		area := monitor.MainResolution
		w, h := area.Max.X, area.Max.Y

		var err error

		overlayWindow, err = overlayApp.NewWindow(url, &astilectron.WindowOptions{
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
			Icon:                   astikit.StrPtr("./assets/icon/icon-browser.png"),

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
		overlayApp.On(astilectron.EventNameAppEventReady, func(e astilectron.Event) (deleteListener bool) {
			return <-waitq
		})

		err = overlayWindow.Create()
		if err != nil {
			errq <- fmt.Errorf("Failed to open %s (%v)", Title, err)
			return
		}

		// window.OpenDevTools()

		close(errq)
		close(waitq)

		overlayApp.Wait()
	}()

	return <-errq
}

func windowClosed(e astilectron.Event) (deleteListener bool) {
	notify.SystemWarn("Window crashed (%s)", e.Code)
	return
}
