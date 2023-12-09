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

	"github.com/pidgy/unitehud/core/global"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/media/video/wapi"
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
	app    *astilectron.Astilectron
	window *astilectron.Window
	active = struct{ app, window bool }{}

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

func Open() {
	notify.Debug("Electron: Opening...")
	defer notify.Debug("Electron: Closed")

	var err error

	app, err = astilectron.New(
		notify.Debugger("Electron: "),
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
		notify.Error("Electron: Failed to create app (%v)", err)
		return
	}

	app.HandleSignals()
	app.On(astilectron.EventNameAppCrash, onClose)
	app.On(astilectron.EventNameAppCmdQuit, onClose)
	app.On(astilectron.EventNameAppClose, onClose)
	app.On(astilectron.EventNameAppEventReady, func(e astilectron.Event) (deleteListener bool) {
		notify.Debug("Electron: event, %s", e.Name)
		active.app = true
		return false
	})

	err = app.Start()
	if err != nil {
		notify.Error("Electron: Failed to start app (%v)", err)
		return
	}

	go app.Wait()
}

func Close() {
	notify.Debug("Electron: Closing app...")
	defer notify.Debug("Electron: Closed app...")

	app.Close()

	active.window = false
	active.app = false
}

func CloseWindow() {
	notify.Debug("Electron: Closing window...")
	defer notify.Debug("Electron: Closed window...")

	err := window.Close()
	if err != nil {
		notify.Error("Electron: Failed to close window (%v)", err)
	}
}

func Follow(hwnd uintptr, hidden bool) {
	b := offscreen
	if !hidden {
		r := &wapi.Rect{}

		_, _, err := wapi.GetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(r)))
		if err != syscall.Errno(0) {
			notify.Error("Electron: Failed to follow Client window (%v)", err)
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

	for _, err := range []error{
		window.SetBounds(b),
		window.MoveTop(),
		window.Show(),
	} {
		if err != nil {
			notify.Debug("Electron: Failed to render (%v)", err)
		}
	}
}

func OpenWindow() (err error) {
	if active.window {
		return window.Show()
	}

	notify.Debug("Electron: Opening window...")
	defer notify.Debug("Electron: Opened window")

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
		notify.Error("Electron: Failed to open (%v)", err)
		return
	}

	window.On(astilectron.EventNameWindowEventDidFinishLoad, func(e astilectron.Event) (deleteListener bool) {
		notify.Debug("Electron: event, %s", e.Name)
		active.window = true
		return false
	})

	errq := make(chan error)

	go func() {
		err := window.Create()
		if err != nil {
			errq <- errors.Wrap(window.Show(), "Overlay:")
			return
		}

		// if global.DebugMode {
		// err := window.OpenDevTools()
		// if err != nil {
		// 	notify.Warn("Electron: Failed to open dev tools")
		// }
		// }

		errq <- errors.Wrap(window.Show(), "Overlay:")
	}()

	return <-errq
}

func onClose(e astilectron.Event) (deleteListener bool) {
	notify.Debug("Electron: app event %s", e.Name)
	return false
}
