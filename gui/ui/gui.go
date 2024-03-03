package ui

import (
	"fmt"
	"image"
	"math"
	"time"
	"unsafe"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"

	"github.com/pidgy/unitehud/avi/video/fps"
	"github.com/pidgy/unitehud/avi/video/monitor"
	"github.com/pidgy/unitehud/avi/video/wapi"
	"github.com/pidgy/unitehud/core/fonts"
	"github.com/pidgy/unitehud/core/global"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/stats"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/ux/title"
	"github.com/pidgy/unitehud/system/process"
)

type Action string

type GUI struct {
	HWND uintptr

	window *app.Window
	nav    *title.Widget

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

	hz *fps.Hz
}

const (
	Start = Action("start")
	Stop  = Action("stop")
	// Record = Action("record")
	// Open    = Action("open")
	Closing = Action("closing")
	Refresh = Action("refresh")
	// Config  = Action("Config")
	// Log     = Action("Log")
)

var UI *GUI

func New() {
	err := wapi.SetProcessDPIAwareness(wapi.PerMonitorAware)
	if err != nil {
		notify.Warn("UI: Failed to set DPI awareness, %v", err)
	}

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

		hz: fps.NewHz(),

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

	UI.nav = title.New(
		global.Title,
		fonts.NewCollection(),
		UI.minimize,
		UI.resize,
		func() {
			UI.window.Perform(system.ActionClose)
		},
	)

	notify.System("UI: Using %dx%d resolution", max.X, max.Y)

	go UI.loading()

	// go UI.draw()
}

func (g *GUI) Close() {
	g.next(is.Closing)
}

func (g *GUI) Open() {
	g.next(is.MainMenu)

	// tray.Open(g.Close)
	// defer tray.Close()

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

	p, ok := g.nav.Dragging()
	if ok {
		g.setWindowPos(p)
		return
	}

	g.hz.Tick(gtx.Now)
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
	peak := struct{ cpu, ram float64 }{}

	for ; is.Now != is.Closing; time.Sleep(time.Second) {
		uptime := process.Uptime()
		g.performance.uptime = fmt.Sprintf("%02d:%02d:%02d", uptime.Hour(), uptime.Minute(), uptime.Second())

		ram := process.RAM()
		if ram > peak.ram+100 {
			peak.ram = ram
			notify.SystemWarn("UI: Consumed %.0fMB of RAM", peak.ram)
		}
		g.performance.ram = fmt.Sprintf("RAM %.0f%s", ram, "MB")
		go stats.RAM(ram)

		cpu, err := process.CPU()
		if err != nil {
			notify.Error("Process: Failed to monitor CPU/RAM (%v)", err)
			continue
		}
		if cpu > peak.cpu*2 {
			peak.cpu = cpu
			notify.SystemWarn("UI: Consumed %.1f%s CPU", peak.cpu, "%")
		}
		g.performance.cpu = fmt.Sprintf("CPU %.1f%s", cpu, "%")
		go stats.CPU(cpu)
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
