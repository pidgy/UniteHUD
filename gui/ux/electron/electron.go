package electron

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const.

import (
	"context"
	"fmt"
	"image"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"

	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/gui/ux/title"
	"github.com/pidgy/unitehud/system/wapi"
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
	name = "UniteHUD Overlay"
)

// debugger defines debugger behavior and state.
type debugger struct {
	fmt,
	ftl func(format string, v ...interface{})
}

var (
	app            *astilectron.Astilectron
	window         *astilectron.Window
	active, hidden bool

	html = filepath.Join(exe.Directory(), "www", "UniteHUD Client.html")
)

func Active() bool {
	return active
}

// Close closes the target.
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
	rect   wapi.Rect
	hidden bool
}

var lock = &sync.Mutex{}

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

	switch wapi.Window(hwnd).Info().Status {
	case wapi.WindowStatusNotVisible:
		if !force {
			Hide()
		}
	case wapi.WindowStatusVisible:
		err := show()
		if err != nil {
			notify.Error("[Electron] <ini:f:to> show HUD (%v)", err)
		}

		next := wapi.Rect{}
		_, _, err = wapi.GetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&next)))
		if err != syscall.Errno(0) {
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
	case wapi.WindowStatusUnknown:
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

// Open opens the target.
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

// closeApp closes the target.
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

// closeWindow closes the target.
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

// newDebugger returns a new Debugger.
func newDebugger(prefix string) *debugger {
	return &debugger{
		fmt: func(format string, v ...interface{}) { notify.Debug(prefix+" "+format, v...) },
		ftl: func(format string, v ...interface{}) { notify.Debug(prefix+" [Fatal] "+format, v...) },
	}
}

func (d *debugger) Fatal(v ...interface{}) { d.ftl("%s", fmt.Sprint(v...)) }

func (d *debugger) Fatalf(format string, v ...interface{}) { d.ftl(format, v...) }

func (d *debugger) Print(v ...interface{}) { d.fmt("%s", fmt.Sprint(v...)) }

func (d *debugger) Printf(format string, v ...interface{}) { d.fmt(format, v...) }

// openApp opens the target.
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

// openWindow opens the target.
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

func trySetBounds(next wapi.Rect, w, h int, force bool) error {
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
