package electron

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"time"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"

	"github.com/pidgy/unitehud/fps"
	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/video/monitor"
)

const (
	Title = "UniteHUD Overlay"
)

var (
	Captureq = make(chan image.Image)

	app    *astilectron.Astilectron
	window *astilectron.Window

	opened = false
)

func Close() {
	defer func() {
		app, window = nil, nil
		opened = false
	}()

	if window != nil {
		notify.System("Overlay: Closing rendering window")

		err := window.Destroy()
		if err != nil {
			notify.Warn("Overlay: Failed to close (%v)", err)
		}
	}

	if app != nil {
		notify.Debug("Overlay: Closing rendering app")

		err := app.Quit()
		if err != nil {
			notify.Warn("Overlay: Failed to close rendering app(%v)", err)
		}
	}
}

func Open() error {
	if app != nil && window != nil {
		return nil
	}

	notify.System("Overlay: Opening...")

	err := openApp()
	if err != nil {
		return err
	}

	return openWindow()
}

func openApp() error {
	if opened {
		return nil
	}
	opened = true

	a, err := astilectron.New(nil, astilectron.Options{
		AppName:            Title,
		BaseDirectoryPath:  ".",
		DataDirectoryPath:  fmt.Sprintf("./%s/electron", global.AssetsFolder),
		AppIconDefaultPath: fmt.Sprintf("./%s/icon/icon.png", global.AssetsFolder),
		VersionElectron:    astilectron.DefaultVersionElectron,
		VersionAstilectron: astilectron.DefaultVersionAstilectron,
	})
	if err != nil {
		return fmt.Errorf("Failed to start app for %s (%v)", Title, err)
	}
	app = a

	app.HandleSignals()

	app.On(astilectron.EventNameAppCrash, onClose)
	app.On(astilectron.EventNameAppCmdQuit, onClose)
	app.On(astilectron.EventNameAppClose, onClose)

	return app.Start()
}

func openWindow() error {
	errq := make(chan error)

	app.On(astilectron.EventNameAppEventReady, func(e astilectron.Event) (deleteListener bool) {
		notify.System("Overlay: Started")
		return true
	})

	go func() {
		area := monitor.MainResolution
		w, h := area.Max.X, area.Max.Y

		var err error

		window, err = app.NewWindow(fmt.Sprintf(`%s/html/UniteHUD Client.html`, global.AssetsFolder),
			&astilectron.WindowOptions{
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
				Closable:    astikit.BoolPtr(false),

				Transparent: astikit.BoolPtr(true),
				AlwaysOnTop: astikit.BoolPtr(false),

				EnableLargerThanScreen: astikit.BoolPtr(false),
				Focusable:              astikit.BoolPtr(false),
				Frame:                  astikit.BoolPtr(false),
				HasShadow:              astikit.BoolPtr(false),
				Icon:                   astikit.StrPtr(fmt.Sprintf("%s/icon/icon-browser.png", global.AssetsFolder)),

				Show: astikit.BoolPtr(false),

				WebPreferences: &astilectron.WebPreferences{
					DevTools:                astikit.BoolPtr(global.DebugMode),
					Images:                  astikit.BoolPtr(true),
					Javascript:              astikit.BoolPtr(true),
					NodeIntegrationInWorker: astikit.BoolPtr(true),
				},
			},
		)
		if err != nil {
			notify.Error("Overlay: Failed to open (%v)", err)
			errq <- err
			return
		}

		err = window.Create()
		if err != nil {
			errq <- fmt.Errorf("Overlay: Failed to start (%v)", err)
			return
		}

		close(errq)

		defer fps.NewLoop(&fps.LoopOptions{
			Async: true,
			FPS:   1,
			Render: func(min, max, avg time.Duration) (close bool) {
				if window == nil {
					return true
				}

				err = window.SendMessage("screenshot", func(m *astilectron.EventMessage) {
					var response = struct {
						Type string `json:"type"`
						Data []byte `json:"data"`
					}{}

					err = m.Unmarshal(&response)
					if err != nil {
						notify.Error("Overlay: Failed to parse response (%v)", err)
						return
					}

					img, err := png.Decode(bytes.NewReader(response.Data))
					if err != nil {
						notify.Error("Overlay: Failed to decode response (%v)", err)
						return
					}

					Captureq <- img

					notify.Debug("Overlay: Received \"%s\" (%d bytes)", response.Type, len(response.Data))
				})
				if err != nil {
					notify.Error("Overlay: Failed to capture (%v)", err)
				}

				return err != nil
			},
		}).Stop()

		app.Wait()
	}()

	return <-errq
}

func onClose(e astilectron.Event) (deleteListener bool) {
	notify.Debug("Render: Closed (%s)", e.Name)
	return true
}
