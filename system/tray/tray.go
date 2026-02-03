package tray

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const.

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rupor-github/win-gpg-agent/systray"
	"github.com/skratchdot/open-golang/open"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/media/audio"
	"github.com/pidgy/unitehud/media/img"
	"github.com/pidgy/unitehud/media/video"
	"github.com/pidgy/unitehud/media/video/device"
	"github.com/pidgy/unitehud/system/lang"
	"github.com/pidgy/unitehud/system/process"
	"github.com/pidgy/unitehud/system/save"
	"github.com/pidgy/unitehud/system/wapi"
)

type toggle struct {
	*systray.MenuItem
	event func()
}

var (
	WarnLog  = func(s string, a ...any) {}
	ErrorLog = func(s string, a ...any) {}
	InfoLog  = func(s string, a ...any) {}

	menu = struct {
		visible bool

		header    toggle
		logs      toggle
		website   toggle
		startstop toggle
		hideshow  toggle
		exit      toggle

		hwnd uintptr

		eventq     chan func()
		startstopq chan bool

		hidden bool
	}{
		eventq:     make(chan func(), 1024),
		startstopq: make(chan bool, 1024),
	}
)

func Close() {
	if !menu.visible {
		return
	}
	menu.visible = false

	notify.Debug("[Tray] Closing")
	systray.Quit()
}

func Open() error {
	notify.Debug("[Tray] Opening...")

	go systray.Run(
		func() { // OnReady.
			for menu.hwnd == 0 {
				time.Sleep(time.Second)
			}

			menu.header = header(exe.Title, exe.TitleAndVersion)
			proc()
			configuration()
			menu.logs = logs()
			menu.hideshow = hideshow()
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
					if menu.startstop.Checked() {
						menu.startstop.Uncheck()
					} else {
						menu.startstop.Check()
					}
				case <-menu.hideshow.ClickedCh:
					menu.hideshow.event()
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

	return nil
}

func SetHWND(h uintptr) {
	menu.hwnd = uintptr(h)
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

	gaming := m.AddSubMenuItem("Gaming", "")
	gdev := gaming.AddSubMenuItem("Device\tUnknown", "")
	gdev.Disable()

	vid := m.AddSubMenuItem("Video", "")
	vdev := vid.AddSubMenuItem("Device\tUnknown", "")
	vdev.Disable()
	api := vid.AddSubMenuItem("API\tUnknown", "")
	api.Disable()
	codec := vid.AddSubMenuItem("Codec\tUnknown", "")
	codec.Disable()
	fps := vid.AddSubMenuItem("FPS\t0", "")
	fps.Disable()

	aud := m.AddSubMenuItem("Audio", "")
	audIn := aud.AddSubMenuItem("Input\tUnknown", "")
	audIn.Disable()
	audOut := aud.AddSubMenuItem("Output\tUnknown", "")
	audOut.Disable()

	go func() {
		for ; ; time.Sleep(time.Second) {
			gdev.SetTitle(fmt.Sprintf("Device\t%s", lang.Title(config.Current.Gaming.Device)))

			vdev.SetTitle(fmt.Sprintf("%s\t%s", video.Current(), video.Name()))

			api.SetTitle(fmt.Sprintf("API\t%s", lang.Title(config.Current.Video.Capture.Device.API)))
			codec.SetTitle(fmt.Sprintf("Codec\t%s", config.Current.Video.Capture.Device.Codec))
			fps.SetTitle(fmt.Sprintf("FPS\t%.1f/%d", device.FPS(), config.Current.Video.Capture.Device.FPS))

			audIn.SetTitle(fmt.Sprintf("Input\t %s", audio.Current.Input))
			audOut.SetTitle(fmt.Sprintf("Output\t%s", audio.Current.Output))
		}
	}()

	conf := m.AddSubMenuItem("Open File\t(Read Only)", "View current configuration file")

	go func() {
		for range conf.ClickedCh {
			path, err := config.Current.SaveTemp()
			if err != nil {
				notify.Error("[UI] Failed to create \"%s\" (%v)", path, err)
				continue
			}

			err = exec.Command("C:\\Windows\\system32\\notepad.exe", path).Run()
			if err != nil {
				notify.Error("[UI] <ini:f:open> \"%s\" (%v)", path, err)
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
		event: func() {
			exe.Close()
		},
	}
}

func header(title, version string) toggle {
	notify.Debug("[Tray] Adding Title")

	icon := img.IconBytes("icon.ico")
	if icon != nil {
		systray.SetIcon(icon)
	}
	systray.SetTitle(title)
	systray.SetTooltip(version)

	m := systray.AddMenuItem(version, "Open UniteHUD")
	iconbg := img.IconBytes("icon-bg.ico")
	if iconbg != nil {
		m.SetIcon(iconbg)
	}

	return toggle{
		MenuItem: m,
		event: func() {
			notify.Debug("[Tray] Raising HWND: %d", menu.hwnd)

			if menu.hidden {
				menu.hideshow.event()
			} else {
				wapi.SetWindowPosNoSizeNoMoveShowWindow(menu.hwnd)
			}
		},
	}
}

func hideshow() toggle {
	systray.AddSeparator()

	return toggle{
		MenuItem: systray.AddMenuItem("Hide", "Hide UniteHUD in System Tray"),
		event: func() {
			notify.Debug("[Tray] Hiding/Showing HWND: %d", menu.hwnd)

			if menu.hidden {
				wapi.ShowWindowRestore(menu.hwnd)
				menu.hideshow.SetTitle("Hide")
				menu.hideshow.SetTooltip("Hide UniteHUD in System Tray")
			} else {
				wapi.ShowWindowHide(menu.hwnd)
				menu.hideshow.SetTitle("Show")
				menu.hideshow.SetTooltip("Restore UniteHUD on your Desktop")
			}

			menu.hidden = !menu.hidden
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
					notify.Error("[Tray] <ini:f:open> log directory (%v)", err)
				}
			case <-view.ClickedCh:
				err := save.OpenCurrentLog()
				if err != nil {
					notify.Error("[Tray] <ini:f:open> log directory (%v)", err)
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

	cpu := systray.AddMenuItem("CPU\t0%", "CPU")
	cpu.Disable()

	ram := systray.AddMenuItem("RAM\t0MB", "RAM")
	ram.Disable()

	threads := systray.AddMenuItem("Threads\t0", "Threads")
	threads.Disable()

	go func() {
		for ; ; time.Sleep(time.Second) {
			cpu.SetTitle(strings.Replace(process.Usage.CPU.String(), " ", "\t", 1))
			ram.SetTitle(strings.Replace(process.Usage.RAM.String(), " ", "\t", 1))
			threads.SetTitle(strings.Replace(process.Usage.Threads.String(), " ", "\t", 1))
			threads.Check()
			ram.Check()
			cpu.Check()
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
				notify.Error("[Tray] <ini:f:open> unitehud.dev (%v)", err)
			}
		},
	}
}
