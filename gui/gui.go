package gui

import (
	"fmt"
	"image"
	"runtime"
	"syscall"
	"time"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/screen"
	"github.com/pidgy/unitehud/gui/visual/split"
	"github.com/pidgy/unitehud/gui/visual/textblock"
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/stats"
	"github.com/pidgy/unitehud/video"
)

type Action string

type GUI struct {
	*app.Window
	*title.Bar
	*screen.Screen

	Preview bool
	open    bool

	resize  bool
	resized bool

	Actions chan Action

	Running bool

	cpu, ram, uptime string
	time             time.Time

	cascadia, normal *material.Theme

	toastActive    bool
	lastToastError error
	lastToastTime  time.Time

	ecoMode bool

	readyq chan bool

	resizeToMax bool
}

const (
	Start   = Action("start")
	Stats   = Action("stats")
	Stop    = Action("stop")
	Record  = Action("record")
	Open    = Action("open")
	Closing = Action("closing")
	Refresh = Action("refresh")
	Debug   = Action("debug")
	Idle    = Action("idle")
	Config  = Action("Config")
	Log     = Action("Log")
)

var Window *GUI

func New() {
	cas, err := textblock.NewCascadiaCode()
	if err != nil {
		notify.Warn("Failed to create CPU/RAM graph (%v)", err)
	}
	_ = cas

	Window = &GUI{
		Window: app.NewWindow(
			app.Title("UniteHUD"),
			app.Decorated(false),
			app.Size(
				unit.Dp(1080),
				unit.Dp(648+split.DefaultBarSizeAdjustable),
			),
			app.MinSize(
				unit.Dp(1080),
				unit.Dp(648+split.DefaultBarSizeAdjustable),
			),
		),

		Preview: true,
		Actions: make(chan Action, 1024),
		resize:  true,
		ecoMode: true,
		readyq:  make(chan bool),

		// cascadia = material.NewTheme(cas.Collection())
		cascadia: material.NewTheme(gofont.Collection()),
		normal:   material.NewTheme(gofont.Collection()),

		Bar: title.New(
			title.Default,
			material.NewTheme(gofont.Collection()),
			func() {
				Window.Perform(system.ActionMinimize)
			},
			func() {
				Window.resizeToMax = !Window.resizeToMax
				if Window.resizeToMax {
					Window.Perform(system.ActionMaximize)
					return
				}
				Window.Perform(system.ActionUnmaximize)
			},
			func() {
				Window.Perform(system.ActionClose)
			},
		),
	}

	go Window.loading()
}

func (g *GUI) Open() {
	g.readyq <- true

	go func() {
		g.open = true

		g.Actions <- Refresh
		defer func() {
			g.Actions <- Closing
		}()

		go g.preview()

		var err error

		next := "main"

		for next != "" {
			switch next {
			case "main":
				g.resize = false

				next, err = g.main()
				if err != nil {
					g.ToastError(err)
				}
			case "configure":
				g.resize = true

				next, err = g.configure()
				if err != nil {
					g.ToastError(err)
				}
			default:
				return
			}
		}
	}()

	go g.proc()

	//g.Window.Option(app.Fullscreen.Option())
	app.Main()
}

func (g *GUI) Title(t string) string {
	t = fmt.Sprintf("%s %s", title.Default, t)
	g.Bar.Title = t
	return t
}

func (g *GUI) proc() {
	handle, err := syscall.GetCurrentProcess()
	if err != nil {
		notify.Error("Failed to monitor usage: (%v)", err)
		return
	}

	var ctime, etime, ktime, utime syscall.Filetime
	err = syscall.GetProcessTimes(handle, &ctime, &etime, &ktime, &utime)
	if err != nil {
		notify.Error("Failed to monitor CPU/RAM (%v)", err)
		return
	}

	prev := ctime.Nanoseconds()
	usage := ktime.Nanoseconds() + utime.Nanoseconds() // Always overflows.

	g.time = time.Now()

	cpus := float64(runtime.NumCPU()) - 2

	peakCPU := 0.0
	peakRAM := 0.0

	for {
		time.Sleep(time.Second)

		err := syscall.GetProcessTimes(handle, &ctime, &etime, &ktime, &utime)
		if err != nil {
			notify.Error("Failed to monitor CPU/RAM (%v)", err)
			continue
		}

		now := time.Now().UnixNano()
		diff := now - prev
		current := ktime.Nanoseconds() + utime.Nanoseconds()
		diff2 := current - usage
		prev = now
		usage = current

		cpu := (100 * float64(diff2) / float64(diff)) / cpus
		if cpu > peakCPU+10 {
			peakCPU = cpu
			notify.SystemWarn("Consumed %.1f%s CPU", peakCPU, "%")
		}

		g.cpu = fmt.Sprintf("CPU %.1f%s", cpu, "%")

		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		ram := (float64(m.Sys) / 1024 / 1024)
		if ram > peakRAM+10 {
			peakRAM = ram
			notify.SystemWarn("Consumed %.0f%s of RAM", peakRAM, "MB")
		}

		g.ram = fmt.Sprintf("RAM %.0f%s", ram, "MB")

		run := time.Time{}.Add(time.Since(g.time))
		if run.Hour() > 0 {
			g.uptime = fmt.Sprintf("%1d:%02d:%02d", run.Hour(), run.Minute(), run.Second())
		} else {
			g.uptime = fmt.Sprintf("%02d:%02d", run.Minute(), run.Second())
		}

		go stats.CPU(cpu)
		go stats.RAM(ram)
	}
}

func (g *GUI) display(src image.Image) {
	g.Screen = &screen.Screen{
		Image:         src,
		VerticalScale: true,
		Splash:        true,
	}

	if g.open {
		// Prevent capturing once the window has been resized.
		if !g.resized {
			g.resize = false
			g.resized = true
			g.Preview = false
		}
	}
}

func (g *GUI) preview() {
	for ; ; time.Sleep(time.Millisecond * 50) {
		if g.Preview {
			img, err := video.Capture()
			if err != nil {
				g.ToastError(err)
				continue
			}

			g.display(img)
		}

		// Redraw the image.
		g.Invalidate()
	}
}

// buttonSpam ensures we only execute a config reload once before cooling down.
func (g *GUI) buttonSpam(b *button.Button) {
	b.LastPressed = time.Now()

	time.AfterFunc(time.Second, func() {
		if time.Since(b.LastPressed) >= time.Second {
			config.Current.Reload()
			g.Preview = true
		}
	},
	)
}
