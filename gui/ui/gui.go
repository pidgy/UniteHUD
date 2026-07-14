package ui

import (
	"image"
	"runtime"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/stats"
	"github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/ux/title"
	"github.com/pidgy/unitehud/media/video/fps"
	"github.com/pidgy/unitehud/media/video/monitor"
	"github.com/pidgy/unitehud/system/process"
	"github.com/pidgy/unitehud/system/win32"
)

// GUI manages the main application window, state, and UI lifecycle.
type GUI struct {
	HWND uintptr

	window *app.Window
	nav    *title.Widget

	Preview bool
	open    bool
	Running bool

	onClose func()

	dimensions dimensions

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

	hz *fps.Hz
}

type dimensions struct {
	size,
	shift image.Point

	smoothing int // Redraw every other frame to reduce shakiness.
	threshold int // Redraw every other frame to reduce shakiness.

	fullscreen,
	resizing bool

	inset struct {
		left,
		right int
	}
}

func (d *dimensions) max() image.Point {
	return monitor.Bounds().Size().Sub(image.Pt(d.inset.left+d.inset.right, 0))
}

// UI holds the global GUI instance.
var UI *GUI

// New initializes and returns a new GUI instance.
func New() *GUI {
	err := win32.SetProcessDPIAwareness(win32.PerMonitorAware)
	if err != nil {
		notify.Warn("[UI] <ini:f:set> DPI awareness, %v", err)
	}

	d := dimensions{threshold: 2}

	notify.System("[UI] Generating")

	notify.Debug("[UI] Taskbar Height: %d", monitor.TaskbarHeight())

	UI = &GUI{
		window: app.NewWindow(
			app.Title(exe.Title),
			app.Decorated(false),
			app.WindowMode.Option(app.Windowed),
		),

		HWND: 0,

		dimensions: d,

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
		exe.Title,
		UI.minimize,
		UI.resize,
		UI.Close,
		// func() {UI.window.Perform(system.ActionClose)},
	)

	go UI.loading()

	return UI
}

// Close closes the global GUI instance if it exists.
func Close() {
	if UI != nil {
		UI.Close()
	}
}

// Close transitions the GUI to the closing state.
func (g *GUI) Close() {
	is.Next(is.Closing)
}

// OnClose sets a callback to run when the GUI closes.
func (g *GUI) OnClose(fn func()) *GUI {
	g.onClose = fn
	return g
}

// Open starts the GUI event loop and background processing.
func (g *GUI) Open() {
	is.Next(is.MainMenu)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		g.open = true
		defer g.onClose()

		for {
			switch {
			case is.Currently(is.Closing):
				return
			case is.Currently(is.Loading):
				notify.Debug("[UI] Loading...")
			case is.Currently(is.MainMenu):
				g.main()
			case is.Currently(is.Configuring):
				g.configure()
			default:
				g.ToastErrorf("Unexpected configuration, shutting down")
				return
			}
		}
	}()

	go g.proc()

	if !is.Currently(is.Closing) {
		app.Main()
	}
}

// attachWindowLeft positions a window to the left of the main window.
func (g *GUI) attachWindowLeft(hwnd uintptr, width int) {
	if hwnd == 0 {
		return
	}

	g.dimensions.inset.left = width

	pos := g.position()

	x := max(pos.X-width, 0)
	y := pos.Y
	if y < 0 {
		y += title.Height
	}

	// Set the follower window.
	win32.SetWindowPosShow(hwnd, image.Pt(x, y), image.Pt(width, g.dimensions.size.Y))

	g.squeeze()
}

func (g *GUI) deattachWindowLeft() {
	g.dimensions.inset.left = 0

	if g.dimensions.fullscreen {
		g.unmaximize()
		g.maximize()
	}
}

// attachWindowRight positions a window to the right of the main window.
func (g *GUI) attachWindowRight(hwnd uintptr, width int) {
	if hwnd == 0 {
		return
	}

	g.dimensions.inset.right = width

	pos := g.position()

	attached := pos.Add(image.Pt(g.dimensions.size.X, 0))
	if attached.Y < 0 {
		attached.Y += title.Height
	}

	// Set the follower window.
	win32.SetWindowPosShow(hwnd, attached, image.Pt(width, g.dimensions.size.Y))

	g.squeeze()
}

func (g *GUI) deattachWindowRight() {
	g.dimensions.inset.right = 0

	if g.dimensions.fullscreen {
		g.unmaximize()
		g.maximize()
	}
}

// frame renders a single GUI frame and handles dragging.
func (g *GUI) frame(gtx layout.Context, e system.FrameEvent) {
	g.window.Invalidate()

	e.Frame(gtx.Ops)

	p, ok := g.nav.Dragging()
	if ok {
		g.moveWindow(p)
		return
	}

	g.hz.Tick(gtx.Now)
}

// maximize makes the window fullscreen and disables dragging.
func (g *GUI) maximize() {
	g.dimensions.fullscreen = true
	g.nav.NoDrag = true
	g.previous.size = g.dimensions.size
	g.dimensions.size = g.dimensions.max()
	g.window.Option(app.Size(unit.Dp(g.dimensions.size.X), unit.Dp(g.dimensions.size.Y)))
	g.window.Option(app.MaxSize(unit.Dp(g.dimensions.size.X), unit.Dp(g.dimensions.size.Y)))

	g.window.Perform(system.ActionMaximize)
}

// minimize minimizes the window.
func (g *GUI) minimize() {
	g.window.Perform(system.ActionMinimize)
}

// moveWindow applies a drag shift to reposition the window.
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

		g.dimensions.shift = shift

		if g.dimensions.smoothing == g.dimensions.threshold {
			win32.MoveWindowNoSize(g.HWND, pos)
			g.dimensions.smoothing = 0
		}
		g.dimensions.smoothing++
	}()
}

// position returns the current window position.
func (g *GUI) position() image.Point {
	next, err := win32.Window(g.HWND).RectComplete()
	if err != nil {
		return image.Point{}
	}
	return next.Image().Min
}

// proc tracks and reports process performance over time.
func (g *GUI) proc() {
	peak := struct{ cpu, ram float64 }{}

	for ; !is.Currently(is.Closing); time.Sleep(time.Second) {
		g.performance.uptime = process.Uptime()

		if process.Usage.RAM.Float64() > peak.ram+100 {
			peak.ram = process.Usage.RAM.Float64()
			notify.Replace("[UI] RAM", notify.Warn, "[UI] RAM Usage: %.0fMB", peak.ram)
		}

		if process.Usage.CPU.Float64() > peak.cpu+10 {
			peak.cpu = process.Usage.CPU.Float64()
			notify.Replace("[UI] CPU Usage", notify.Warn, "[UI] CPU Usage: %.1f%s", peak.cpu, "%")
		}

		go stats.Procs(process.Usage.CPU.Float64(), process.Usage.RAM.Float64(), process.Usage.Threads.Float64())
	}
}

// resize toggles between maximized and normal states.
func (g *GUI) resize() {
	if g.dimensions.fullscreen {
		g.unmaximize()
	} else {
		g.maximize()
	}
}

// squeeze shrinks the window to fit current insets.
func (g *GUI) squeeze() {
	if g.dimensions.fullscreen {
		size := g.dimensions.max()
		g.window.Option(app.Size(unit.Dp(size.X), unit.Dp(size.Y)))
		g.window.Option(app.MaxSize(unit.Dp(size.X), unit.Dp(size.Y)))

		win32.SetWindowPosNone(
			g.HWND,
			//  image.Pt(g.dimensions.inset.left, 0),
			monitor.BoundsFromCoordinate(g.position().X).Min.Add(image.Pt(g.dimensions.inset.left, 0)),
			size,
		)
	}
}

// unmaximize restores the window from fullscreen state.
func (g *GUI) unmaximize() {
	g.dimensions.fullscreen = false
	g.nav.NoDrag = false
	g.dimensions.size = g.previous.size
	max := g.dimensions.max()
	g.window.Option(app.Size(unit.Dp(g.dimensions.size.X), unit.Dp(g.dimensions.size.Y)))
	g.window.Option(app.MaxSize(unit.Dp(max.X), unit.Dp(max.Y)))

	g.window.Perform(system.ActionUnmaximize)
}
