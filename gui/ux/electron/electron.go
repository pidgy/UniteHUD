package electron

import (
	"context"
	"fmt"
	"image"
	"path/filepath"
	"sync"
	"time"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"

	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/gui/ux/title"
	"github.com/pidgy/unitehud/system/win32"
)

//! Required: assets/electron/vendor/astilelectron/index.js
//!
//! function windowCreate(json) {
//!     ...
//!     if (typeof json.windowOptions.proxy !== "undefined") {
//!         elements[json.targetID].webContents.session.setProxy(json.windowOptions.proxy)
//!             .then(() => windowCreateFinish(json))
//!     } else {
//!         elements[json.targetID].setIgnoreMouseEvents(true)  <--- Custom option.
//!         windowCreateFinish(json)
//!     }
//! }

const (
	// name is the Electron window title.
	name = "UniteHUD Overlay"
)

// debugger adapts logging to astilectron's debugger interface.
type debugger struct {
	fmt,
	ftl func(format string, v ...any)
}

var (
	// app is the active astilectron application instance.
	app *astilectron.Astilectron
	// window is the overlay window instance.
	window *astilectron.Window
	// active reports whether the overlay app is running.
	active, hidden bool

	// html is the path to the overlay UI HTML file.
	html = filepath.Join(exe.Directory(), "www", "UniteHUD Client.html")
)

// Active reports whether the Electron overlay is running.
func Active() bool {
	return active
}

// Close shuts down the overlay window and app.
func Close() {
	if !active {
		notify.Warn("[Electron] <ini:f:close> (inactive)")
		return
	}
	active = false
	hidden = true

	err := closeWindow()
	if err != nil {
		notify.Error("[Electron] <ini:f:close> (%v)", err)
		return
	}

	err = closeApp()
	if err != nil {
		notify.Error("[Electron] <ini:f:close> (%v)", err)
		return
	}
}

var prev struct {
	rect   win32.Rect
	hidden bool
}

var lock = &sync.Mutex{}

// Follow positions the overlay over the given window and scales it to size.
func Follow(hwnd uintptr, size image.Point, force bool) {
	if !active || hwnd == 0 {
		notify.Debug("[Electron] Not following HWND:%d (active:%t) ", hwnd, active)
		return
	}

	// if !lock.TryLock() {
	// 	notify.Error("[Electron] Failed to follow window (busy)")
	// 	return
	// }
	// defer lock.Unlock()

	switch win32.Window(hwnd).Info().Status {
	case win32.WindowStatusNotVisible:
		if !force {
			Hide()
		}
	case win32.WindowStatusVisible:
		err := show()
		if err != nil {
			notify.Error("[Electron] <ini:f:to> show HUD (%v)", err)
		}

		next, err := win32.Window(hwnd).RectComplete()
		if err != nil {
			notify.Error("[Electron] Failed to find projector dimensions (%v)", err)
			return
		}

		insetY := (int(next.Bottom-next.Top) - size.Y)     // Video is always attached to top.
		insetX := (int(next.Right-next.Left) - size.X) / 2 // Video is evenly set between minX -> maxX.

		next.Bottom -= int32(insetY)
		next.Left += int32(insetX)
		next.Right -= int32(insetX)

		if next.Eq(prev.rect) {
			return
		}
		prev.rect = next

		// err = window.ExecuteJavaScript(`document.getElementById('hud').style.opacity = 0;`)
		// if err != nil {
		// 	notify.Warn("[Electron] <ini:f:set> HUD opacity (%v)", err)
		// }

		w := int(next.Right - next.Left)
		h := int(next.Bottom - next.Top)

		err = window.ExecuteJavaScript(fmt.Sprintf(`
			document.getElementById('hud').style.transform = 'scaleX(%f) scaleY(%f)';
			document.getElementById('hud').style.transformOrigin = '0 0';`,
			float64(w)/1920,
			float64(h)/1080,
		))
		if err != nil {
			notify.Warn("[Electron] <ini:f:set> HUD scale (%v)", err)
			break
		}

		err = trySetBounds(next, w, h, force)
		if err != nil {
			notify.Error("[Electron] <ini:f:set> HUD bounds (%v)", err)
			break
		}

		// err = window.ExecuteJavaScript(`document.getElementById('hud').style.opacity = 1;`)
		// if err != nil {
		// 	notify.Warn("[Electron] <ini:f:unset> HUD opacity (%v)", err)
		// }no

		err = window.MoveTop()
		if err != nil {
			notify.Error("[Electron] <ini:f:to> move HUD to top (%v)", err)
		}
	case win32.WindowStatusUnknown:
		notify.Error("[Electron] Unknown Window Info Status")
		return
	}

	prev.hidden = hidden
}

func Hide() {
	if hidden {
		return
	}
	hidden = true

	notify.Debug("[Electron] Hiding overlay... (hidden:%t)", hidden)

	// err := window.Hide()
	// if err != nil {
	// 	notify.Error("[Electron] Failed to hide overlay (%v)", err)
	// }
	err := window.ExecuteJavaScript(`document.getElementById('hud').style.opacity = 0;`)
	if err != nil {
		notify.Warn("[Electron] <ini:f:set> HUD opacity (%v)", err)
	}
}

// Open starts the overlay app and window.
func Open(size image.Point) error {
	if active {
		return fmt.Errorf("window is active")
	}

	err := openApp()
	if err != nil {
		notify.Error("[Electron] <ini:f:open> (%v)", err)
		return err
	}

	err = openWindow(size)
	if err != nil {
		notify.Error("[Electron] <ini:f:open> (%v)", err)
		return err
	}

	active = true

	return nil
}

// closeApp terminates the astilectron app instance.
func closeApp() error {
	notify.Debug("[Electron] Closing app...")
	defer notify.Debug("[Electron] Closed app")

	err := app.Quit()
	if err != nil {
		return err
	}

	app.Stop()
	app.Close()

	return nil
}

// closeWindow destroys the overlay window instance.
func closeWindow() error {
	notify.Debug("[Electron] Closing window...")
	defer notify.Debug("[Electron] Closed window")

	err := window.UpdateCustomOptions(astilectron.WindowCustomOptions{MinimizeOnClose: astikit.BoolPtr(false)})
	if err != nil {
		return err
	}

	err = window.Destroy()
	if err != nil {
		return err
	}

	return nil
}

// newDebugger returns a logger wired to the notify system.
func newDebugger(prefix string) *debugger {
	return &debugger{
		fmt: func(format string, v ...any) { notify.Debug(prefix+" "+format, v...) },
		ftl: func(format string, v ...any) { notify.Debug(prefix+" [Fatal] "+format, v...) },
	}
}

// Fatal logs a fatal message.
func (d *debugger) Fatal(v ...any) { d.ftl("%s", fmt.Sprint(v...)) }

// Fatalf logs a formatted fatal message.
func (d *debugger) Fatalf(format string, v ...any) { d.ftl(format, v...) }

// Print logs a message.
func (d *debugger) Print(v ...any) { d.fmt("%s", fmt.Sprint(v...)) }

// Printf logs a formatted message.
func (d *debugger) Printf(format string, v ...any) { d.fmt(format, v...) }

// openApp starts the astilectron application.
func openApp() error {
	notify.Debug("[Electron] Opening app...")
	defer notify.Debug("[Electron] Opened app")

	var err error

	app, err = astilectron.New(
		newDebugger("[Electron]"),
		astilectron.Options{
			AppName:            name,
			CustomElectronPath: filepath.Join(exe.Directory(), exe.AssetDirectory, "electron", "vendor", "electron-windows-amd64", "UniteHUD Overlay.exe"),
			BaseDirectoryPath:  ".",
			DataDirectoryPath:  filepath.Join(exe.Directory(), exe.AssetDirectory, "electron"),
			AppIconDefaultPath: filepath.Join(exe.Directory(), exe.AssetDirectory, "icon", "icon.png"),
			VersionElectron:    astilectron.DefaultVersionElectron,
			VersionAstilectron: astilectron.DefaultVersionAstilectron,
			AcceptTCPTimeout:   time.Hour * 24,
			ElectronSwitches:   []string{"disable-http-cache"},
		},
	)
	if err != nil {
		return err
	}

	app.HandleSignals()

	app.On(astilectron.EventNameAppEventReady, func(e astilectron.Event) (deleteListener bool) {
		notify.Debug("[Electron] event, %s", e.Name)
		return true
	})

	err = app.Start()
	if err != nil {
		return err
	}

	go func() {
		defer notify.Debug("[Electron] Exiting main loop")
		app.Wait()
	}()

	return nil
}

// openWindow creates the transparent overlay window.
func openWindow(size image.Point) error {
	notify.Debug("[Electron] Opening window...")
	defer notify.Debug("[Electron] Opened window")

	var err error

	window, err = app.NewWindow(
		html,
		&astilectron.WindowOptions{
			Title: astikit.StrPtr(name),
			Show:  astikit.BoolPtr(true),

			Width:  astikit.IntPtr(size.X),
			Height: astikit.IntPtr(size.Y),

			Minimizable: astikit.BoolPtr(true),
			Resizable:   astikit.BoolPtr(true),
			Movable:     astikit.BoolPtr(true),
			Closable:    astikit.BoolPtr(true),
			Transparent: astikit.BoolPtr(true),
			AlwaysOnTop: astikit.BoolPtr(true),
			Focusable:   astikit.BoolPtr(false),
			Frame:       astikit.BoolPtr(false),

			// Fullscreen:  astikit.BoolPtr(true),
			// Center:      astikit.BoolPtr(true),
			// EnableLargerThanScreen: astikit.BoolPtr(false),
			HasShadow: astikit.BoolPtr(false),

			Icon: astikit.StrPtr(fmt.Sprintf("%s/icon/icon-browser.png", exe.AssetDirectory)),

			WebPreferences: &astilectron.WebPreferences{
				WebSecurity: astikit.BoolPtr(false),
				// DevTools:                astikit.BoolPtr(exe.Debug),
				Images:                  astikit.BoolPtr(true),
				Javascript:              astikit.BoolPtr(true),
				NodeIntegrationInWorker: astikit.BoolPtr(true),
			},

			Custom: &astilectron.WindowCustomOptions{
				MinimizeOnClose: astikit.BoolPtr(false),
			},
		},
	)
	if err != nil {
		notify.Error("[Electron] <ini:f:open> (%v)", err)
		return err
	}

	errq := make(chan error)

	go func() {
		err := window.Create()
		if err != nil {
			errq <- err
			return
		}
		errq <- show()
	}()

	err = <-errq
	if err != nil {
		return err
	}

	app.On(astilectron.EventNameWindowEventShow, func(e astilectron.Event) (deleteListener bool) {
		notify.Debug("[Electron] event, %s", e.Name)
		return true
	})

	return nil
}

// show makes the overlay visible.
func show() error {
	if !hidden {
		return nil
	}
	hidden = false

	notify.Debug("[Electron] Showing overlay...")

	// err :=
	return window.ExecuteJavaScript(`document.getElementById('hud').style.opacity = 1;`)
	// if err != nil {
	// notify.Warn("[Electron] <ini:f:set> HUD opacity (%v)", err)
	// }

	// return window.Show()
}

// trySetBounds updates the overlay window bounds with a timeout.
func trySetBounds(next win32.Rect, w, h int, force bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	errq := make(chan error)

	go func(errq chan error) {
		htmlInsetY := int32(0)
		if !force {
			htmlInsetY = int32(title.Height)
		}

		err := window.SetBounds(
			astilectron.RectangleOptions{
				PositionOptions: astilectron.PositionOptions{
					X: astikit.IntPtr(int(next.Left)),
					Y: astikit.IntPtr(int(next.Top + htmlInsetY)),
				},
				SizeOptions: astilectron.SizeOptions{
					Width:  astikit.IntPtr(w + title.Height),
					Height: astikit.IntPtr(h + title.Height),
				},
			},
		)
		if err != nil {
			errq <- err
			return
		}

		close(errq)
	}(errq)

	select {
	case err := <-errq:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
