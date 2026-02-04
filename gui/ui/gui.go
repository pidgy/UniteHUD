package ui

import (
	"image"
	"runtime"
	"time"
	"unsafe"

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
	"github.com/pidgy/unitehud/system/wapi"
)

// GUI manages the main application window, state, and UI lifecycle.
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

// UI holds the global GUI instance.
var UI *GUI

// New initializes and returns a new GUI instance.
func New() *GUI {
	err := wapi.SetProcessDPIAwareness(wapi.PerMonitorAware)
	if err != nil {
		notify.Warn("[UI] <ini:f:set> DPI awareness, %v", err)
	}

	min := image.Pt(1100, 700)
	max := min

	notify.System("[UI] Generating")

	notify.Debug("[UI] Taskbar Height: %d", monitor.TaskbarHeight())

	UI = &GUI{
		window: app.NewWindow(
			app.Title(exe.Title),
			app.Decorated(false),
			app.Size(unit.Dp(min.X), unit.Dp(min.Y)),
			app.WindowMode.Option(app.Windowed),
		),

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
		exe.Title,
		UI.minimize,
		UI.resize,
		UI.Close,
		// func() {UI.window.Perform(system.ActionClose)},
	)

	notify.Debug("[UI] Minimum window size set to %dx%d", min.X, min.Y)

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

// attachWindowRight positions a window to the right of the main window.
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
	g.window.Perform(system.ActionMaximize)
	g.dimensions.fullscreen = true
	g.nav.NoDrag = true

	// size := g.squeeze()

	// g.previous.position = g.position()
	// g.previous.size = g.dimensions.size

	// g.dimensions.fullscreen = true
	// g.nav.NoDrag = true

	// wapi.SetWindowPosShow(
	// 	g.HWND,
	// 	image.Pt(0, 0).Add(image.Pt(g.inset.left, 0)),
	// 	size,
	// )
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

// position returns the current window position.
func (g *GUI) position() image.Point {
	r := &wapi.Rect{}
	wapi.GetWindowRect.Call(g.HWND, uintptr(unsafe.Pointer(r)))
	return image.Pt(int(r.Left), int(r.Top))
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

// setInsetLeft increases the left inset and adjusts the window.
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

// setInsetRight increases the right inset and adjusts the window.
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

// squeeze shrinks the window to fit current insets.
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

// unmaximize restores the window from fullscreen state.
func (g *GUI) unmaximize() {
	g.window.Perform(system.ActionUnmaximize)
	g.dimensions.fullscreen = false
	g.nav.NoDrag = false

	g.window.Option(app.MinSize(unit.Dp(g.dimensions.min.X), unit.Dp(g.dimensions.min.Y)))
	g.dimensions.fullscreen = false
	g.nav.NoDrag = false

	// wapi.SetWindowPosShow(
	// 	g.HWND,
	// 	g.previous.position,
	// 	g.previous.size,
	// )
}

// unsetInsetLeft decreases the left inset and adjusts the window.
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

// unsetInsetRight decreases the right inset and adjusts the window.
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
