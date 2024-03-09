package tray

import (
	"os"

	"github.com/rupor-github/win-gpg-agent/systray"
	"github.com/skratchdot/open-golang/open"

	"github.com/pidgy/unitehud/avi/img"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/system/wapi"
)

type toggle struct {
	*systray.MenuItem
	event func()
}

var (
	hwnd = uintptr(0)
)

var menu = struct {
	visible bool

	header,
	website,
	startstop,
	quit toggle

	startstopq chan bool
	errorq     chan error
}{
	startstopq: make(chan bool, 1),
	errorq:     make(chan error),
}

func Close() {
	if !menu.visible {
		return
	}
	menu.visible = false

	notify.Debug("Tray: Closing")
	systray.Quit()
}

func Open(title, version string, exit func()) error {
	notify.Debug("Tray: Opening...")
	defer notify.Debug("Tray: Opened")

	go systray.Run(func() {
		menu.header = header(title, version)
		menu.website = website()
		menu.startstop = startstop()
		menu.quit = quit()

		menu.errorq <- nil

		menu.visible = true

		for {
			select {
			case <-menu.header.ClickedCh:
				menu.header.event()
			case <-menu.website.ClickedCh:
				menu.website.event()
			case <-menu.startstop.ClickedCh:
				menu.startstop.event()
			case <-menu.quit.ClickedCh:
				menu.quit.event()
			}
		}
	}, nil, func(s systray.SessionEvent) { notify.System("Tray: Session Event \"%s\"", s) })

	return <-menu.errorq
}

func SetHWND(h uintptr) {
	hwnd = uintptr(h)
}

func SetStartStopDisabled()      { menu.startstop.Disable() }
func SetStartStopEnabled()       { menu.startstop.Enable() }
func SetStartStopTitle(t string) { menu.startstop.SetTitle(t) }
func StartStopEvent() bool {
	if !menu.visible {
		return false
	}

	select {
	case <-menu.startstopq:
		return menu.startstopq != nil
	default:
		return false
	}
}

func quit() toggle {
	notify.Debug("Tray: Adding Quit")

	systray.AddSeparator()

	return toggle{
		MenuItem: systray.AddMenuItem("Quit UniteHUD", "Close UniteHUD"),
		event:    func() { os.Exit(0) },
	}
}

func startstop() toggle {
	notify.Debug("Tray: Adding Start/Stop")

	systray.AddSeparator()

	return toggle{
		MenuItem: systray.AddMenuItem("Start", "Start capturing events"),
		event:    func() { menu.startstopq <- true },
	}
}

func header(title, version string) toggle {
	notify.Debug("Tray: Adding Title")

	systray.SetIcon(img.IconBytes("icon.ico"))
	systray.SetTitle(title)
	systray.SetTooltip(version)

	t := systray.AddMenuItem(version, "Open UniteHUD")
	t.SetIcon(img.IconBytes("icon-bg.ico"))

	return toggle{
		MenuItem: t,
		event: func() {
			notify.Debug("Tray: Raising hwnd: %d", hwnd)
			wapi.RaiseWindow(hwnd)
		},
	}
}

func website() toggle {
	notify.Debug("Tray: Adding Website")

	systray.AddSeparator()

	return toggle{
		MenuItem: systray.AddMenuItem("Website", "Browse to https://unitehud.dev"),
		event: func() {
			err := open.Run("https://unitehud.dev")
			if err != nil {
				notify.Error("Tray: Failed to open unitehud.dev (%v)", err)
			}
		},
	}
}
