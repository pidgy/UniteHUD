//go:build !lite

package gui

import (
	"fmt"
	"image"
	"math"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"

	"github.com/pidgy/unitehud/fonts"
	"github.com/pidgy/unitehud/fps"
	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/tray"
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/stats"
	"github.com/pidgy/unitehud/video/monitor"
	"github.com/pidgy/unitehud/video/wapi"
)

type Action string

type GUI struct {
	HWND uintptr

	window *app.Window
	header *title.Widget

	inset struct {
		left,
		right int
	}

	Preview bool
	open    bool
	Actions chan Action
	Running bool

	dimensions struct {
		min,
		max,
		size,
		shift image.Point

		smoothing int // Redraw every other frame to reduce shakiness.

		fullscreen,
		resizing bool
	}

	performance struct {
		cpu, ram, uptime string
		eco              bool
	}

	previous struct {
		position,
		size image.Point

		toast struct {
			active bool
			time   time.Time
			err    error
		}
	}

	resizeToMax,
	firstOpen bool

	fps *fps.FPS
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

var UI *GUI

func New() {
	min := image.Pt(1100, 700)
	max := monitor.MainResolution.Max

	is.Now = is.Loading

	notify.System("UI: Generating")

	UI = &GUI{
		window: app.NewWindow(app.Title(global.Title), app.Decorated(false)),

		HWND: 0,

		dimensions: struct {
			min,
			max,
			size,
			shift image.Point

			smoothing int

			fullscreen,
			resizing bool
		}{
			min,
			max,
			min,
			image.Pt(0, 0),
			0,
			false,
			false,
		},

		Preview: true,
		Actions: make(chan Action, 1024),

		fps: fps.New(),

		performance: struct {
			cpu,
			ram,
			uptime string

			eco bool
		}{
			cpu:    "0%",
			ram:    "0MB",
			uptime: "00:00",

			eco: true,
		},
	}

	UI.header = title.New(
		global.Title,
		fonts.NewCollection(),
		UI.minimize,
		UI.resize,
		func() {
			UI.window.Perform(system.ActionClose)
		},
	)

	notify.System("UI: Default dimensions detected (%s)", max.String())

	go UI.loading()

	// go UI.draw()
}

func (g *GUI) Close() {
	g.next(is.Closing)
}

func (g *GUI) Open() {
	g.next(is.MainMenu)

	tray.Open(g.Close)
	defer tray.Close()

	go func() {
		g.open = true

		g.Actions <- Refresh
		defer func() {
			g.Actions <- Closing
		}()

		for is.Now != is.Closing {
			switch is.Now {
			case is.Loading:
				notify.Debug("UI: Loading...")
			case is.MainMenu:
				g.main()
			case is.Projecting:
				g.projector()
			default:
				g.ToastError(fmt.Errorf("Unexpected configuration... shutting down"))
				return
			}
		}
	}()

	go g.proc()

	if is.Now != is.Closing {
		app.Main()
	}
}

func (g *GUI) attachWindowLeft(hwnd uintptr, width int) {
	if hwnd == 0 {
		return
	}

	pos := g.position()

	x := pos.X - width
	if x < 0 {
		x = 0
	}
	y := pos.Y
	if y < 0 {
		y += title.Height
	}

	wapi.SetWindowPosNone(hwnd, image.Pt(x, y), image.Pt(width, g.dimensions.size.Y))
}

func (g *GUI) attachWindowRight(hwnd uintptr, width int) bool {
	if hwnd == 0 {
		return false
	}

	pos := g.position()

	attached := pos.Add(image.Pt(g.dimensions.size.X, 0))
	if attached.Y < 0 {
		attached.Y += title.Height
	}

	wapi.SetWindowPosNone(hwnd, attached, image.Pt(width, g.dimensions.size.Y))

	return true
}

func (g *GUI) frame(gtx layout.Context, e system.FrameEvent) {
	e.Frame(gtx.Ops)

	p, ok := g.header.Dragging()
	if ok {
		g.setWindowPos(p)
		return
	}

	g.fps.Tick(gtx.Now)
}

func (g *GUI) maximize() {
	g.previous.position = g.position()
	g.previous.size = g.dimensions.size

	left := image.Pt(0, 0).Add(image.Pt(g.inset.left, 0))
	right := g.dimensions.max.Sub(image.Pt(g.inset.left+g.inset.right+1, 0))
	wapi.SetWindowPosShow(g.HWND, left, right)

	//g.window.Perform(system.ActionCenter)

	g.dimensions.fullscreen = true
}

func (g *GUI) minimize() {
	g.window.Perform(system.ActionMinimize)
}

func (g *GUI) next(i is.What) {
	notify.Debug("UI: Next state set to \"%s\"", i)
	is.Now = i
}

func (g *GUI) position() image.Point {
	r := &wapi.Rect{}
	wapi.GetWindowRect.Call(g.HWND, uintptr(unsafe.Pointer(r)))
	return image.Pt(int(r.Left), int(r.Top))
}

func (g *GUI) proc() {
	handle, err := syscall.GetCurrentProcess()
	if err != nil {
		notify.Error("UI: Failed to monitor usage: (%v)", err)
		return
	}

	var ctime, etime, ktime, utime syscall.Filetime
	err = syscall.GetProcessTimes(handle, &ctime, &etime, &ktime, &utime)
	if err != nil {
		notify.Error("UI: Failed to monitor CPU/RAM (%v)", err)
		return
	}

	prev := ctime.Nanoseconds()
	usage := ktime.Nanoseconds() + utime.Nanoseconds() // Always overflows.

	cpus := float64(runtime.NumCPU()) - 2

	peakCPU := 0.0
	peakRAM := 0.0

	for is.Now != is.Closing {
		time.Sleep(time.Second)

		err := syscall.GetProcessTimes(handle, &ctime, &etime, &ktime, &utime)
		if err != nil {
			notify.Error("UI: Failed to monitor CPU/RAM (%v)", err)
			continue
		}

		now := time.Now().UnixNano()
		diff := now - prev
		current := ktime.Nanoseconds() + utime.Nanoseconds()
		diff2 := current - usage
		prev = now
		usage = current

		cpu := (100 * float64(diff2) / float64(diff)) / cpus
		if cpu > peakCPU*2 {
			peakCPU = cpu
			notify.SystemWarn("UI: Consumed %.1f%s CPU", peakCPU, "%")
		}

		g.performance.cpu = fmt.Sprintf("CPU %.1f%s", cpu, "%")

		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		ram := (float64(m.Sys) / 1024 / 1024)
		if ram > peakRAM+100 {
			peakRAM = ram
			notify.SystemWarn("UI: Consumed %.0f%s of RAM", peakRAM, "MB")
		}

		g.performance.ram = fmt.Sprintf("RAM %.0f%s", ram, "MB")

		run := time.Time{}.Add(time.Since(global.Uptime))
		if run.Hour() > 0 {
			g.performance.uptime = fmt.Sprintf("%1d:%02d:%02d", run.Hour(), run.Minute(), run.Second())
		} else {
			g.performance.uptime = fmt.Sprintf("%02d:%02d", run.Minute(), run.Second())
		}

		go stats.CPU(cpu)
		go stats.RAM(ram)
	}
}

func (g *GUI) resize() {
	if g.dimensions.fullscreen {
		g.unmaximize()
	} else {
		g.maximize()
	}
}

func (g *GUI) setInsetLeft(left int) {
	g.inset.left += left

	if g.dimensions.fullscreen {
		g.maximize()
		return
	}

	// Move right when space is not available for the inset.
	pos := g.position()

	if pos.X < g.inset.left {
		pos.X += g.inset.left - pos.X
	}

	wapi.SetWindowPosShow(g.HWND, pos, g.dimensions.size)
}

func (g *GUI) setInsetRight(right int) {
	g.inset.right += right

	if g.dimensions.fullscreen {
		g.maximize()
		return
	}

	// Move left when new size exceeds max boundaries.
	pos := g.position()
	size := pos.X + g.dimensions.size.X + g.inset.right

	if size > g.dimensions.max.X {
		pos.X -= size - g.dimensions.max.X
	}

	wapi.SetWindowPosShow(g.HWND, pos, g.dimensions.size)
}

func (g *GUI) setWindowPos(shift image.Point) {
	if g.dimensions.fullscreen || g.HWND == 0 || g.dimensions.resizing {
		return
	}
	g.dimensions.resizing = true

	go func() {
		defer func() { g.dimensions.resizing = false }()

		if shift.Eq(g.dimensions.shift) {
			return
		}

		pos := g.position().Add(shift)
		if !pos.In(image.Rectangle{Min: image.Pt(0, 0).Sub(g.dimensions.size), Max: g.dimensions.max.Add(g.dimensions.size)}) {
			return
		}

		g.dimensions.shift = shift

		if g.dimensions.smoothing == 2 {
			wapi.SetWindowPosNoSize(g.HWND, pos)
			g.dimensions.smoothing = 0
		}
		g.dimensions.smoothing++
	}()
}

func (g *GUI) unmaximize() {
	wapi.SetWindowPosShow(g.HWND, g.previous.position, g.previous.size)
	g.dimensions.fullscreen = false
}

func (g *GUI) unsetInsetLeft(left int) {
	g.inset.left -= left

	if g.dimensions.fullscreen {
		g.maximize()
		return
	}

	wapi.SetWindowPosShow(g.HWND, g.position(), g.dimensions.size)
}

func (g *GUI) unsetInsetRight(right int) {
	g.inset.right -= right

	if g.dimensions.fullscreen {
		g.maximize()
		return
	}

	wapi.SetWindowPosShow(g.HWND, g.position(), g.dimensions.size)
}

func max(i, j int) int {
	return int(math.Max(float64(i), float64(j)))
}
