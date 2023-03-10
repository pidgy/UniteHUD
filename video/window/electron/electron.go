package electron

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/notify"
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
		notify.System("Closing %s (%s)", config.BrowserWindow, config.Current.BrowserWindowURL)

		err := window.Destroy()
		if err != nil {
			notify.Warn("Failed to close %s (%v)", config.BrowserWindow, err)
		}
	}

	if app != nil {
		notify.System("Closing %s Controller (%s)", config.BrowserWindow, config.Current.BrowserWindowURL)

		err := app.Quit()
		if err != nil {
			notify.Warn("Failed to close %s Controller (%v)", config.BrowserWindow, err)
		}
	}
}

func Open() error {
	if app != nil && window != nil {
		return nil
	}

	if config.Current.BrowserWindowURL == "" {
		resetWindow(astilectron.Event{})
		return fmt.Errorf("Failed to open %s (no url provided)", config.BrowserWindow)
	}

	notify.System("Opening %s (%s)", config.BrowserWindow, config.Current.BrowserWindowURL)

	// Prevent main thread blocking errors in a goroutine.
	runtime.LockOSThread()

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
		AppName:            config.BrowserWindow,
		BaseDirectoryPath:  ".",
		DataDirectoryPath:  "./assets/electron",
		AppIconDefaultPath: "./assets/icon/icon.png",
	})
	if err != nil {
		return fmt.Errorf("Failed to start app for %s (%v)", config.BrowserWindow, err)
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
		hd, ok := embedded(config.Current.BrowserWindowURL)
		if !ok {
			notify.System("%s is optimized for youtube.com/embed URLs", config.BrowserWindow)
		}

		w, h := 1920, 1080

		var err error

		window, err = app.NewWindow(hd, &astilectron.WindowOptions{
			AlwaysOnTop:            astikit.BoolPtr(false),
			BackgroundColor:        astikit.StrPtr("#0a0814"),
			Center:                 astikit.BoolPtr(true),
			Closable:               astikit.BoolPtr(config.Current.DisableBrowserFormatting),
			EnableLargerThanScreen: astikit.BoolPtr(true),
			Focusable:              astikit.BoolPtr(config.Current.DisableBrowserFormatting),
			Frame:                  astikit.BoolPtr(false),
			HasShadow:              astikit.BoolPtr(true),
			Height:                 astikit.IntPtr(h),
			Icon:                   astikit.StrPtr("./assets/icon/icon_browser.png"),
			MaxWidth:               astikit.IntPtr(w),
			MaxHeight:              astikit.IntPtr(h),
			Minimizable:            astikit.BoolPtr(false),
			Resizable:              astikit.BoolPtr(false),
			Show:                   astikit.BoolPtr(true),
			Width:                  astikit.IntPtr(w),
			WebPreferences: &astilectron.WebPreferences{
				BackgroundThrottling: astikit.BoolPtr(false),
				Webaudio:             astikit.BoolPtr(false),
			},
		})
		if err != nil {
			errq <- fmt.Errorf("Failed to create %s (%v)", config.BrowserWindow, err)
			return
		}

		err = window.Create()
		if err != nil {
			errq <- fmt.Errorf("Failed to open %s (%v)", config.BrowserWindow, err)
			return
		}

		err = window.SetBounds(astilectron.RectangleOptions{
			SizeOptions: astilectron.SizeOptions{
				Height: astikit.IntPtr(h),
				Width:  astikit.IntPtr(w),
			},
			PositionOptions: astilectron.PositionOptions{
				X: astikit.IntPtr(0),
				Y: astikit.IntPtr(10),
			},
		})
		if err != nil {
			notify.Warn("Failed to set %s bounds (%v)", config.BrowserWindow, err)
		}

		if !config.Current.DisableBrowserFormatting {
			for _, script := range []string{"style.js", "video.js", "ads.js"} {
				buf, err := os.ReadFile(fmt.Sprintf("assets/js/%s", script))
				if err != nil {
					notify.Warn("Failed to find %s %s script (%v)", config.BrowserWindow, script, err)
					continue
				}

				err = window.ExecuteJavaScript(string(buf))
				if err != nil {
					notify.Warn("Failed to execute %s %s script (%v)", config.BrowserWindow, script, err)
					continue
				}
			}
		}

		close(errq)

		app.Wait()
	}()

	return <-errq
}

func resetWindow(e astilectron.Event) (deleteListener bool) {
	if config.Current.Window == config.MainDisplay {
		return true
	}

	notify.SystemWarn("%s closed by user, display has been set to %s", config.BrowserWindow, config.MainDisplay)
	config.Current.Window = config.MainDisplay

	return true
}

func embedded(url string) (string, bool) {
	if config.Current.DisableBrowserFormatting {
		return url, true
	}

	args := strings.Split(url, "youtube.com")
	if len(args) > 2 {
		return url, false
	}

	switch {
	case strings.Contains(url, "watch?v="):
		// youtube.com/watch?v=9WYKYLOm5zo
		// -> youtube.com/embed/9WYKYLOm5zo
		return fmt.Sprintf("%s?autoplay=1&vq=hd1080&mute=1", strings.ReplaceAll(url, "watch?v=", "embed/")), true
	case strings.Contains(url, "live/"):
		// youtube.com/live/9WYKYLOm5zo
		// -> youtube.com/embed/9WYKYLOm5zo
		return fmt.Sprintf("%s?autoplay=1&vq=hd1080&mute=1", strings.ReplaceAll(url, "live/", "embed/")), true
	default:
		return fmt.Sprintf("%s?autoplay=1&vq=hd1080&mute=1", url), true
	}
}
