package tray

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/rupor-github/win-gpg-agent/systray"
	"github.com/skratchdot/open-golang/open"

	"github.com/pidgy/unitehud/avi/audio"
	"github.com/pidgy/unitehud/avi/img"
	"github.com/pidgy/unitehud/avi/video"
	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/system/process"
	"github.com/pidgy/unitehud/system/save"
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

	header    toggle
	logs      toggle
	website   toggle
	startstop toggle
	hide      toggle
	exit      toggle

	eventq     chan func()
	startstopq chan bool
}{
	eventq:     make(chan func(), 1024),
	startstopq: make(chan bool, 1024),
}

func Close() {
	if !menu.visible {
		return
	}
	menu.visible = false

	notify.Debug("[Tray] Closing")
	systray.Quit()
}

func Open(title, version string, onExit func()) error {
	notify.Debug("[Tray] Opening...")

	go func() {
		runtime.LockOSThread()

		for hwnd == 0 {
			time.Sleep(time.Second)
		}

		systray.Run(
			func() { // OnReady.
				menu.header = header(title, version)
				proc()
				configuration()
				menu.logs = logs()
				menu.hide = hide()
				menu.startstop = startstop()
				menu.website = website()
				menu.exit = exit()

				menu.visible = true

				notify.Debug("[Tray] Opened")

				for {
					select {
					case fn := <-menu.eventq:
						fn()
					case <-menu.header.ClickedCh:
						menu.header.event()
					case <-menu.website.ClickedCh:
						menu.website.event()
					case <-menu.startstop.ClickedCh:
						menu.startstop.event()
					case <-menu.hide.ClickedCh:
						menu.hide.event()
					case <-menu.exit.ClickedCh:
						menu.exit.event()
					}
				}
			},
			func() { // OnExit.
				notify.Warn("[Tray] Exiting...")
			},

			func(s systray.SessionEvent) { // OnSessionEvent.
				notify.System("[Tray] SessionEvent: %s", s)
			},
		)
	}()

	return nil
}

func SetHWND(h uintptr) {
	hwnd = uintptr(h)
}

func SetStartStopDisabled() {
	menu.eventq <- func() {
		menu.startstop.Disable()
	}
}

func SetStartStopEnabled() {
	menu.eventq <- func() {
		menu.startstop.Enable()
	}
}

func SetStartStopTitle(t string) {
	menu.eventq <- func() {
		menu.startstop.SetTitle(t)
	}
}

func StartStopEvent() bool {
	okq := make(chan bool)

	select {
	case menu.eventq <- func() {
		if !menu.visible {
			okq <- false
			return
		}

		select {
		case <-menu.startstopq:
			okq <- menu.startstopq != nil
		default:
			okq <- false
		}
	}:
	default:
		return false
	}

	t := time.NewTimer(time.Second)
	select {
	case ok := <-okq:
		if !t.Stop() {
			<-t.C
		}
		return ok
	case <-t.C:
		return false
	}
}

func configuration() {
	systray.AddSeparator()

	m := systray.AddMenuItem("Configuration", "View current UniteHUD configuration")
	m.AddSubMenuItem(fmt.Sprintf("Device > %s", strings.Title(config.Current.Gaming.Device)), "").Disable()

	vtitle := m.AddSubMenuItem("Video > Unknown", "")
	vtitle.Disable()
	// vname := m.AddSubMenuItem(" Unknown", "")
	// vname.Disable()

	atitle := m.AddSubMenuItem("Audio > Unknown", "")
	atitle.Disable()

	go func() {
		for ; ; time.Sleep(time.Second) {
			switch {
			case video.Active(video.Device, ""):
				vtitle.SetTitle(fmt.Sprintf("Video > %s", config.Current.Video.Capture.Device.Name))
			case video.Active(video.Monitor, ""), video.Active(video.Window, ""):
				vtitle.SetTitle(fmt.Sprintf("Video > %s", config.Current.Video.Capture.Window.Name))
			}

			atitle.SetTitle(fmt.Sprintf("Audio > %s", audio.Current))
		}
	}()

	conf := m.AddSubMenuItem("Open File (Read Only)", "View current configuration file")

	go func() {
		for range conf.ClickedCh {
			path, err := config.Current.SaveTemp()
			if err != nil {
				notify.Error("[UI] Failed to create \"%s\" (%v)", path, err)
				continue
			}

			err = exec.Command("C:\\Windows\\system32\\notepad.exe", path).Run()
			if err != nil {
				notify.Error("[UI] Failed to open \"%s\" (%v)", path, err)
				continue
			}

			err = os.Remove(path)
			if err != nil {
				notify.Error("[UI] Failed to delete \"%s\" (%v)", path, err)
			}
		}
	}()
}

func exit() toggle {
	notify.Debug("[Tray] Adding Exit")

	systray.AddSeparator()

	return toggle{
		MenuItem: systray.AddMenuItem("Exit UniteHUD", "Quit the program"),
		event:    func() { os.Exit(0) },
	}
}

func header(title, version string) toggle {
	notify.Debug("[Tray] Adding Title")

	systray.SetIcon(img.IconBytes("icon.ico"))
	systray.SetTitle(title)
	systray.SetTooltip(version)

	m := systray.AddMenuItem(version, "Open UniteHUD")
	m.SetIcon(img.IconBytes("icon-bg.ico"))

	return toggle{
		MenuItem: m,
		event: func() {
			notify.Debug("[Tray] Raising hwnd: %d", hwnd)

			wapi.ShowWindowMinimizedRestore(hwnd)

			menu.hide.SetTitle("Hide")
			menu.hide.Enable()
		},
	}
}

func hide() toggle {
	systray.AddSeparator()

	return toggle{
		MenuItem: systray.AddMenuItem("Hide", "Minimize to system tray"),
		event: func() {
			notify.Debug("[Tray] Hiding/Showing hwnd: %d", hwnd)

			wapi.ShowWindowHide(hwnd)
			menu.hide.Disable()
		},
	}
}

func logs() toggle {
	notify.Debug("[Tray] Adding Logs")

	systray.AddSeparator()

	mi := systray.AddMenuItem("Logs", "View present and historical logs")
	view := mi.AddSubMenuItem("Open File", "View active log file")
	open := mi.AddSubMenuItem("Open Directory", "View historical logs")

	go func() {
		for {
			select {
			case <-open.ClickedCh:
				err := save.Open()
				if err != nil {
					notify.Error("[Tray] Failed to open log directory (%v)", err)
				}
			case <-view.ClickedCh:
				err := save.OpenCurrentLog()
				if err != nil {
					notify.Error("[Tray] Failed to open log directory (%v)", err)
				}
			}
		}
	}()

	return toggle{
		MenuItem: mi,
		event:    func() {},
	}
}

func proc() {
	notify.Debug("[Tray] Adding CPU")

	systray.AddSeparator()

	cpu := systray.AddMenuItem("CPU 0%", "CPU")
	cpu.Disable()

	ram := systray.AddMenuItem("RAM 0MB", "RAM")
	ram.Disable()

	go func() {
		for ; ; time.Sleep(time.Second) {
			ram.SetTitle(process.RAM.String())
			cpu.SetTitle(process.CPU.String())
		}
	}()
}

func startstop() toggle {
	notify.Debug("[Tray] Adding Start/Stop")

	systray.AddSeparator()

	return toggle{
		MenuItem: systray.AddMenuItem("Start", "Start capturing events"),
		event:    func() { menu.startstopq <- true },
	}
}

func website() toggle {
	notify.Debug("[Tray] Adding Website")

	systray.AddSeparator()

	return toggle{
		MenuItem: systray.AddMenuItem("Website", "Browse to https://unitehud.dev"),
		event: func() {
			err := open.Run("https://unitehud.dev")
			if err != nil {
				notify.Error("[Tray] Failed to open unitehud.dev (%v)", err)
			}
		},
	}
}
