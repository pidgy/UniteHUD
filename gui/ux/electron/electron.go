package electron

import (
	"context"
	"fmt"
	"image"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	"github.com/pkg/errors"

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

func Close() {
	if !active {
		notify.Warn("[Electron] <ini:failed:close> (inactive)")
		return
	}
	active = false
	hidden = true

	err := closeWindow()
	if err != nil {
		notify.Error("[Electron] <ini:failed:close> (%v)", err)
		return
	}

	err = closeApp()
	if err != nil {
		notify.Error("[Electron] <ini:failed:close> (%v)", err)
		return
	}
}

var prev struct {
	rect   wapi.Rect
	hidden bool
}

func Follow(hwnd uintptr, size image.Point, force bool) {
	if !active || hwnd == 0 {
		notify.Debug("[Electron] Not following HWND:%d (active:%t) ", hwnd, active)
		return
	}

	switch wapi.Window(hwnd).InfoStatus() {
	case wapi.WindowInfoStatusNotVisible:
		if !force {
			Hide()
		}
	case wapi.WindowInfoStatusVisible:
		if prev.hidden {
			defer forceShow()
		}

		next := wapi.Rect{}
		_, _, err := wapi.GetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&next)))
		if err != syscall.Errno(0) {
			notify.Error("[Overlay Window] Failed to match overlay position (%v)", err)
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

		ctx, cfn := context.WithTimeout(context.Background(), time.Millisecond)
		go func(ctx context.Context, _ context.CancelFunc, rect wapi.Rect) {
			h := int(prev.rect.Bottom - prev.rect.Top)
			scaleY := float64(h) / 1080

			w := int(prev.rect.Right - prev.rect.Left)
			scaleX := float64(w) / 1920

			err = window.ExecuteJavaScript(fmt.Sprintf(`
				document.getElementById('hud').style.transform = 'scaleX(%f) scaleY(%f)';
		        document.getElementById('hud').style.transformOrigin = '0 0';
			`, scaleX, scaleY))
			if err != nil {
				notify.Warn("[Electron] <ini:failed:set> custom Javascript (%v)", err)
			}

			insetY := int32(0)
			if !force {
				insetY = int32(title.Height)
			}

			err := window.SetBounds(
				astilectron.RectangleOptions{
					PositionOptions: astilectron.PositionOptions{
						X: astikit.IntPtr(int(prev.rect.Left)),
						Y: astikit.IntPtr(int(prev.rect.Top + insetY)),
					},
					SizeOptions: astilectron.SizeOptions{
						Width:  astikit.IntPtr(w + title.Height),
						Height: astikit.IntPtr(h + title.Height),
					},
				},
			)
			if err != nil {
				notify.Error("[Electron] <ini:failed:set> bounds (%v)", err)
			}

		}(ctx, cfn, prev.rect)

		err = window.MoveTop()
		if err != nil {
			notify.Debug("[Electron] <ini:failed:to> move on top (%v)", err)
		}

		forceShow()
	case wapi.WindowInfoStatusUnknown:
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

	err := window.Hide()
	if err != nil {
		notify.Error("[Electron] Failed to hide overlay (%v)", err)
	}
}

func Open() error {
	if active {
		return fmt.Errorf("window is active")
	}
	hidden = false

	err := openApp()
	if err != nil {
		notify.Error("[Electron] <ini:failed:open> (%v)", err)
		return err
	}

	err = openWindow()
	if err != nil {
		notify.Error("[Electron] <ini:failed:open> (%v)", err)
		return err
	}

	active = true

	return nil
}

func Show() {
	if !hidden {
		return
	}

	forceShow()
}

func closeApp() error {
	notify.Debug("[Electron] Closing app...")
	defer notify.Debug("[Electron] Closed app")

	go app.Stop()
	go app.Close()

	err := app.Quit()
	if err != nil {
		return err
	}

	return nil
}

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

func newDebugger(prefix string) *debugger {
	return &debugger{
		fmt: func(format string, v ...interface{}) { notify.Debug(prefix+" "+format, v...) },
		ftl: func(format string, v ...interface{}) { notify.Debug(prefix+" [Fatal] "+format, v...) },
	}
}
func (d *debugger) Fatal(v ...interface{})                 { d.ftl("%s", fmt.Sprint(v...)) }
func (d *debugger) Fatalf(format string, v ...interface{}) { d.ftl(format, v...) }
func (d *debugger) Print(v ...interface{})                 { d.fmt("%s", fmt.Sprint(v...)) }
func (d *debugger) Printf(format string, v ...interface{}) { d.fmt(format, v...) }

func forceShow() {
	notify.Debug("[Electron] Showing overlay... (hidden:%t)", hidden)

	hidden = false

	err := window.Show()
	if err != nil {
		notify.Error("[Electron] Failed to show (%v)", err)
	}
}

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

func openWindow() error {
	notify.Debug("[Electron] Opening window...")
	defer notify.Debug("[Electron] Opened window")

	var err error

	window, err = app.NewWindow(
		html,
		&astilectron.WindowOptions{
			Title: astikit.StrPtr(name),
			Show:  astikit.BoolPtr(true),

			// Width:  astikit.IntPtr(1280),
			// Height: astikit.IntPtr(720),
			Width:  astikit.IntPtr(1920),
			Height: astikit.IntPtr(1080),

			Minimizable: astikit.BoolPtr(true),
			Resizable:   astikit.BoolPtr(false),
			Movable:     astikit.BoolPtr(true),
			Closable:    astikit.BoolPtr(true),
			Transparent: astikit.BoolPtr(true),
			AlwaysOnTop: astikit.BoolPtr(true),
			Focusable:   astikit.BoolPtr(false),
			Frame:       astikit.BoolPtr(false),

			// Fullscreen:  astikit.BoolPtr(true),
			// Center:      astikit.BoolPtr(true),
			// EnableLargerThanScreen: astikit.BoolPtr(false),
			// HasShadow:              astikit.BoolPtr(false),

			Icon: astikit.StrPtr(fmt.Sprintf("%s/icon/icon-browser.png", exe.AssetDirectory)),

			WebPreferences: &astilectron.WebPreferences{
				WebSecurity:             astikit.BoolPtr(false),
				DevTools:                astikit.BoolPtr(exe.Debug),
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
		notify.Error("[Electron] <ini:failed:open> (%v)", err)
		return err
	}

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
