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
	"gioui.org/unit"

	"github.com/pidgy/unitehud/avi/video/fps"
	"github.com/pidgy/unitehud/avi/video/monitor"
	"github.com/pidgy/unitehud/core/fonts"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/stats"
	app1 "github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/ux/title"
	"github.com/pidgy/unitehud/system/process"
	"github.com/pidgy/unitehud/system/wapi"
)

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
	Running bool

	onClose func()

	dimensions struct {
		min,
		max,
		size,
		shift image.Point

		smoothing int // Redraw every other frame to reduce shakiness.
		threshold int // Redraw every other frame to reduce shakiness.

		fullscreen,
		resizing bool
	}

	performance struct {
		uptime string
		eco    bool
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

	hz *fps.Hz
}

var UI *GUI

func New() *GUI {
	err := wapi.SetProcessDPIAwareness(wapi.PerMonitorAware)
	if err != nil {
		notify.Warn("[UI] Failed to set DPI awareness, %v", err)
	}

	min := image.Pt(1100, 700)
	max := monitor.Resolution().Max.Sub(image.Pt(0, monitor.TaskbarHeight()))

	is.Now = is.Loading

	notify.System("[UI] Generating")

	notify.Announce("Taskbar Height: %d", monitor.TaskbarHeight())

	UI = &GUI{
		window: app.NewWindow(app.Title(app1.Title), app.Decorated(false)),

		HWND: 0,

		dimensions: struct {
			min,
			max,
			size,
			shift image.Point

			smoothing int
			threshold int

			fullscreen,
			resizing bool
		}{
			min,
			max,
			min,
			image.Pt(0, 0),
			0,
			2,
			false,
			false,
		},

		Preview: true,

		hz: fps.NewHz(),

		performance: struct {
			uptime string

			eco bool
		}{
			uptime: "00:00",

			eco: true,
		},
	}

	UI.nav = title.New(
		app1.Title,
		fonts.NewCollection(),
		UI.minimize,
		UI.resize,
		func() {
			UI.window.Perform(system.ActionClose)
		},
	)

	notify.System("[UI] Using %dx%d resolution", max.X, max.Y)

	go UI.loading()

	return UI
}

func Close() {
	if UI != nil {
		UI.Close()
	}
}

func (g *GUI) OnClose(fn func()) *GUI {
	g.onClose = fn
	return g
}

func (g *GUI) Close() {
	g.next(is.Closing)
}

func (g *GUI) Open() {
	g.next(is.MainMenu)

	go func() {
		g.open = true

		defer g.onClose()

		for is.Now != is.Closing {
			switch is.Now {
			case is.Loading:
				notify.Debug("[UI] Loading...")
			case is.MainMenu:
				g.main()
			case is.Configuring:
				g.configure()
			default:
				g.ToastError(fmt.Errorf("Unexpected configuration... shutting down"))
				return
			}
		}
	}()

	go g.proc()

	if app1.Debug {
		// go g.debug()
	}

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
		g.moveWindow(p)
		return
	}

	g.hz.Tick(gtx.Now)
}

func (g *GUI) maximize() {
	size := g.squeeze()

	g.previous.position = g.position()
	g.previous.size = g.dimensions.size

	g.dimensions.fullscreen = true
	g.nav.NoDrag = true

	wapi.SetWindowPosShow(
		g.HWND,
		image.Pt(0, 0).Add(image.Pt(g.inset.left, 0)),
		size,
	)
}

func (g *GUI) minimize() {
	g.window.Perform(system.ActionMinimize)
}

func (g *GUI) moveWindow(shift image.Point) {
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

		if g.dimensions.smoothing == g.dimensions.threshold {
			wapi.MoveWindowNoSize(g.HWND, pos)

			g.dimensions.smoothing = 0
		}
		g.dimensions.smoothing++
	}()
}

func (g *GUI) position() image.Point {
	r := &wapi.Rect{}
	wapi.GetWindowRect.Call(g.HWND, uintptr(unsafe.Pointer(r)))
	return image.Pt(int(r.Left), int(r.Top))
}

func (g *GUI) next(i is.What) {
	notify.Debug("[UI] Next state set to \"%s\"", i)
	is.Now = i
}

func (g *GUI) proc() {
	peak := struct{ cpu, ram float64 }{}

	for ; is.Now != is.Closing; time.Sleep(time.Second) {
		g.performance.uptime = process.Uptime()

		if process.RAM.Float64() > peak.ram+100 {
			peak.ram = process.RAM.Float64()
			notify.Replace("[UI] RAM", notify.Warn, "[UI] RAM Usage: %.0fMB", peak.ram)
		}
		go stats.RAM(process.RAM.Float64())

		if process.CPU.Float64() > peak.cpu+10 {
			peak.cpu = process.CPU.Float64()
			notify.Replace("[UI] CPU Usage", notify.Warn, "[UI] CPU Usage: %.1f%s", peak.cpu, "%")
		}
		go stats.CPU(process.CPU.Float64())
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
		g.squeeze()
		return
	}

	// Move right when space is not available for the inset.
	pos := g.position()

	if pos.X < g.inset.left {
		pos.X += g.inset.left - pos.X
	}

	if g.dimensions.smoothing == g.dimensions.threshold {
		wapi.MoveWindowNoSize(g.HWND, pos)
	}
}

func (g *GUI) setInsetRight(right int) {
	g.inset.right += right

	if g.dimensions.fullscreen {
		g.squeeze()
		return
	}

	// Move left when new size exceeds max boundaries.
	pos := g.position()
	size := pos.X + g.dimensions.size.X + g.inset.right

	if size > g.dimensions.max.X {
		pos.X -= size - g.dimensions.max.X
	}

	if g.dimensions.smoothing == g.dimensions.threshold {
		wapi.MoveWindowNoSize(g.HWND, pos)
	}
}

func (g *GUI) squeeze() image.Point {
	size := g.dimensions.max.Sub(image.Pt(g.inset.left+g.inset.right+1, 0))
	g.window.Option(app.MinSize(unit.Dp(size.X), unit.Dp(size.Y)))

	if g.dimensions.fullscreen {
		wapi.SetWindowPosShow(
			g.HWND,
			image.Pt(0, 0).Add(image.Pt(g.inset.left, 0)),
			size,
		)
	}

	return size
}

func (g *GUI) unmaximize() {
	g.window.Option(app.MinSize(unit.Dp(g.dimensions.min.X), unit.Dp(g.dimensions.min.Y)))
	g.dimensions.fullscreen = false
	g.nav.NoDrag = false

	wapi.SetWindowPosShow(
		g.HWND,
		g.previous.position,
		g.previous.size,
	)
}

func (g *GUI) unsetInsetLeft(left int) {
	g.inset.left -= left

	if g.dimensions.fullscreen {
		g.squeeze()
		return
	}

	if g.dimensions.smoothing == g.dimensions.threshold {
		wapi.MoveWindowNoSize(g.HWND, g.position())
	}
}

func (g *GUI) unsetInsetRight(right int) {
	g.inset.right -= right

	if g.dimensions.fullscreen {
		g.squeeze()
		return
	}

	if g.dimensions.smoothing == g.dimensions.threshold {
		wapi.MoveWindowNoSize(g.HWND, g.position())
	}
}

func max(i, j int) int {
	return int(math.Max(float64(i), float64(j)))
}
