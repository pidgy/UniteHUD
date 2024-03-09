package electron

import (
	"fmt"
	"math"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	"github.com/pkg/errors"

	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/global"
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
	title = "UniteHUD Overlay"
)

var (
	app            *astilectron.Astilectron
	window         *astilectron.Window
	active, hidden bool

	html = filepath.Join(global.WorkingDirectory(), "www", "UniteHUD Client.html")

	offscreen = astilectron.RectangleOptions{
		PositionOptions: astilectron.PositionOptions{
			X: astikit.IntPtr(-math.MaxInt32),
			Y: astikit.IntPtr(-math.MaxInt32),
		},
		SizeOptions: astilectron.SizeOptions{
			Height: astikit.IntPtr(0),
			Width:  astikit.IntPtr(0),
		},
	}
)

func Active() bool {
	return active
}

func Close() {
	if !active {
		notify.Warn("Overlay Window: Failed to close (inactive)")
		return
	}
	active = false
	hidden = true

	err := closeWindow()
	if err != nil {
		notify.Error("Overlay Window: Failed to close (%v)", err)
		return
	}

	err = closeApp()
	if err != nil {
		notify.Error("Overlay Engine: Failed to close (%v)", err)
		return
	}
}

func Follow(hwnd uintptr, parent bool) {
	if !active {
		return
	}

	b := offscreen
	if !parent {
		r := &wapi.Rect{}

		_, _, err := wapi.GetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(r)))
		if err != syscall.Errno(0) {
			notify.Error("Overlay Window: Failed to match overlay position (%v)", err)
			return
		}

		b.PositionOptions = astilectron.PositionOptions{
			X: astikit.IntPtr(int(r.Left)),
			Y: astikit.IntPtr(int(r.Top)),
		}
		b.SizeOptions = astilectron.SizeOptions{
			Width:  astikit.IntPtr(int(r.Right - r.Left)),
			Height: astikit.IntPtr(int(r.Bottom - r.Top)),
		}
	}

	err := window.SetBounds(b)
	if err != nil {
		if active {
			notify.Debug("Overlay Window: Failed to set bounds (%v)", err)
		}
	}

	if !hidden {
		err = window.MoveTop()
		if err != nil {
			notify.Debug("Overlay Window: Failed to move on top (%v)", err)
		}

		err = window.Show()
		if err != nil {
			notify.Debug("Overlay Window: Failed to show (%v)", err)
		}
	}
}

func Hide() {
	hidden = true

	err := window.Hide()
	if err != nil {
		notify.Error("Overlay Window: Failed to hide (%v)", err)
	}
}

func Open() error {
	if active {
		return fmt.Errorf("window is active")
	}
	active = true
	hidden = false

	err := openApp()
	if err != nil {
		notify.Error("Overlay Engine: Failed to open (%v)", err)
		return err
	}

	err = openWindow()
	if err != nil {
		notify.Error("Overlay Window: Failed to open (%v)", err)
		return err
	}

	active = true

	return nil
}

func Show() {
	hidden = false

	err := window.Show()
	if err != nil {
		notify.Error("Overlay Window: Failed to show (%v)", err)
	}
}

func closeApp() error {
	notify.Debug("Overlay Engine: Closing...")
	defer notify.Debug("Overlay Engine: Closed...")

	go app.Stop()
	go app.Close()

	err := app.Quit()
	if err != nil {
		return err
	}

	return nil
}

func closeWindow() error {
	notify.Debug("Overlay Window: Closing...")
	defer notify.Debug("Overlay Window: Closed...")

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

func openApp() error {
	notify.Debug("Overlay Engine: Opening...")

	var err error

	app, err = astilectron.New(
		notify.Debugger("Overlay Engine: "),
		astilectron.Options{
			AppName:            title,
			CustomElectronPath: filepath.Join(global.WorkingDirectory(), global.AssetDirectory, "electron", "vendor", "electron-windows-amd64", "UniteHUD Overlay.exe"),
			BaseDirectoryPath:  ".",
			DataDirectoryPath:  filepath.Join(global.WorkingDirectory(), global.AssetDirectory, "electron"),
			AppIconDefaultPath: filepath.Join(global.WorkingDirectory(), global.AssetDirectory, "icon", "icon.png"),
			VersionElectron:    astilectron.DefaultVersionElectron,
			VersionAstilectron: astilectron.DefaultVersionAstilectron,
			AcceptTCPTimeout:   time.Hour * 24,
		},
	)
	if err != nil {
		return err
	}

	app.HandleSignals()

	// closed := func(e astilectron.Event) (deleteListener bool) {
	// 	notify.Debug("Overlay Engine: %s", e.Name)
	// 	return false
	// }

	// app.On(astilectron.EventNameAppCrash, closed)
	// app.On(astilectron.EventNameAppCmdQuit, closed)
	// app.On(astilectron.EventNameAppClose, closed)
	app.On(astilectron.EventNameAppEventReady, func(e astilectron.Event) (deleteListener bool) {
		notify.Debug("Overlay Engine: event, %s", e.Name)
		return false
	})

	err = app.Start()
	if err != nil {
		return err
	}

	go func() {
		defer notify.Debug("Overlay App: Exiting main loop")
		app.Wait()
	}()

	return nil
}

func openWindow() error {
	notify.Debug("Overlay Window: Opening window...")
	defer notify.Debug("Overlay Window: Opened window")

	var err error

	window, err = app.NewWindow(html,
		&astilectron.WindowOptions{
			Title: astikit.StrPtr(title),
			Show:  astikit.BoolPtr(true),

			Width:  astikit.IntPtr(1280),
			Height: astikit.IntPtr(720),

			// Fullscreen:  astikit.BoolPtr(true),
			Minimizable: astikit.BoolPtr(true),
			Resizable:   astikit.BoolPtr(false),
			Movable:     astikit.BoolPtr(true),
			// Center:      astikit.BoolPtr(true),
			Closable: astikit.BoolPtr(true),

			Transparent: astikit.BoolPtr(true),
			AlwaysOnTop: astikit.BoolPtr(true),

			// EnableLargerThanScreen: astikit.BoolPtr(false),
			Focusable: astikit.BoolPtr(false),
			Frame:     astikit.BoolPtr(false),
			// HasShadow:              astikit.BoolPtr(false),

			Icon: astikit.StrPtr(fmt.Sprintf("%s/icon/icon-browser.png", global.AssetDirectory)),

			WebPreferences: &astilectron.WebPreferences{
				WebSecurity:             astikit.BoolPtr(false),
				DevTools:                astikit.BoolPtr(global.DebugMode),
				Images:                  astikit.BoolPtr(true),
				Javascript:              astikit.BoolPtr(true),
				NodeIntegrationInWorker: astikit.BoolPtr(true),
			},

			Custom: &astilectron.WindowCustomOptions{
				MinimizeOnClose: astikit.BoolPtr(true),
			},
		},
	)
	if err != nil {
		notify.Error("Overlay Window: Failed to open (%v)", err)
		return err
	}

	// ev := func(e astilectron.Event, b bool) (deleteListener bool) {
	// 	notify.Debug("Overlay Window: %s", e.Name)
	// 	active.window = b
	// 	return false
	// }
	// opened := func(e astilectron.Event) (deleteListener bool) { return ev(e, true) }
	// // closed := func(e astilectron.Event) (deleteListener bool) { return ev(e, false) }

	// window.On(astilectron.EventNameWindowEventDidFinishLoad, opened)
	// window.On(astilectron.EventNameWindowEventShow, opened)
	// // window.On(astilectron.EventNameWindowEventClosed, closed)
	// // window.On(astilectron.EventNameWindowEventHide, closed)
	// // window.On(astilectron.EventNameWindowEventMinimize, closed)

	errq := make(chan error)

	go func() {
		err := window.Create()
		if err != nil {
			errq <- errors.Wrap(err, "overlay window")
			return
		}
		errq <- errors.Wrap(window.Show(), "overlay window")
	}()

	err = <-errq
	if err != nil {
		return err
	}

	return nil
}
