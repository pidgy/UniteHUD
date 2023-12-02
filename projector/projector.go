package projector

import (
	"fmt"
	"time"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"

	"github.com/pidgy/unitehud/fps"
	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/video/monitor"
)

const title = "UniteHUD Projector"

func New() {
	a, err := astilectron.New(nil, astilectron.Options{
		AppName:            title,
		BaseDirectoryPath:  ".",
		DataDirectoryPath:  fmt.Sprintf("%s/electron", global.AssetDirectory),
		AppIconDefaultPath: fmt.Sprintf("%s/icon/icon.png", global.AssetDirectory),
		VersionElectron:    astilectron.DefaultVersionElectron,
		VersionAstilectron: astilectron.DefaultVersionAstilectron,
	})
	if err != nil {
		notify.Error("%s: Failed to create app (%v)", title, err)
		return
	}
	defer a.Close()

	a.On(astilectron.EventNameAppCrash, func(e astilectron.Event) (deleteListener bool) {
		notify.System("%s: Crashed (%s)", title, e.Name)
		return false
	})
	a.On(astilectron.EventNameAppCmdQuit, func(e astilectron.Event) (deleteListener bool) {
		notify.System("%s: Quit (%s)", title, e.Name)
		return false
	})
	a.On(astilectron.EventNameAppClose, func(e astilectron.Event) (deleteListener bool) {
		notify.System("%s: Close (%s)", title, e.Name)
		return false
	})
	a.On(astilectron.EventNameAppEventReady, func(e astilectron.Event) (deleteListener bool) {
		notify.System("%s: Ready", title)
		return false
	})

	err = a.Start()
	if err != nil {
		notify.Error("%s: Failed to start app (%v)", title, err)
		return
	}

	w, err := a.NewWindow(fmt.Sprintf(`./%s/html/projector.html`, global.AssetDirectory),
		&astilectron.WindowOptions{
			Title: astikit.StrPtr("UniteHUD Projector"),

			Width:     astikit.IntPtr(monitor.MainResolution.Max.X),
			Height:    astikit.IntPtr(monitor.MainResolution.Max.Y),
			MaxWidth:  astikit.IntPtr(monitor.MainResolution.Max.X),
			MaxHeight: astikit.IntPtr(monitor.MainResolution.Max.Y),
			MinWidth:  astikit.IntPtr(monitor.MainResolution.Max.X),
			MinHeight: astikit.IntPtr(monitor.MainResolution.Max.Y),

			Minimizable: astikit.BoolPtr(true),
			Resizable:   astikit.BoolPtr(false),
			Center:      astikit.BoolPtr(true),
			Closable:    astikit.BoolPtr(true),

			Transparent:     astikit.BoolPtr(false),
			BackgroundColor: astikit.StrPtr("black"),
			AlwaysOnTop:     astikit.BoolPtr(false),

			EnableLargerThanScreen: astikit.BoolPtr(true),
			Focusable:              astikit.BoolPtr(true),
			Frame:                  astikit.BoolPtr(false),
			HasShadow:              astikit.BoolPtr(false),
			Icon:                   astikit.StrPtr(fmt.Sprintf("./%s/icon/icon-browser.png", global.AssetDirectory)),

			Show: astikit.BoolPtr(true),

			WebPreferences: &astilectron.WebPreferences{
				DevTools:   astikit.BoolPtr(true),
				Images:     astikit.BoolPtr(true),
				Javascript: astikit.BoolPtr(true),
			},
		},
	)
	if err != nil {
		notify.Error("%s: Failed to create window (%v)", title, err)
		return
	}

	w.OnMessage(func(m *astilectron.EventMessage) interface{} {
		var s string
		err := m.Unmarshal(&s)
		if err != nil {
			notify.Error("%s: Failed to read message from window (%v)", title, err)
			return nil
		}

		notify.Debug("%s: Received \"%s\"", title, s)

		if s == "close" {
			defer a.Close()

			err := w.Close()
			if err != nil {
				notify.Error("%s: Failed to close window (%v)", title, err)
				return nil
			}

			return "bye bye"
		}

		return nil
	})

	err = w.Create()
	if err != nil {
		notify.Error("%s: Failed to open window (%v)", title, err)
		return
	}

	if global.DebugMode {
		err := w.OpenDevTools()
		if err != nil {
			notify.Warn("%s: Failed to open dev tools", title)
		}
	}

	defer fps.NewLoop(&fps.LoopOptions{
		Async: true,
		Render: func(min, max, avg time.Duration) (close bool) {
			err := w.Session.ClearCache()
			if err != nil {
				notify.Warn("%s: Failed to clear session cache (%v)", title, err)
				return true
			}
			return false
		},
	}).Stop()

	a.Wait()
}
