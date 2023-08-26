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
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/pidgy/unitehud/fonts"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/stats"
	"github.com/pidgy/unitehud/video/monitor"
	"github.com/pidgy/unitehud/video/wapi"
)

type Action string

type GUI struct {
	loaded bool

	is is.Is

	*app.Window
	*title.Bar

	min, max, inset image.Point
	size            image.Point
	shift           image.Point

	HWND uintptr

	Preview bool
	open    bool

	fullscreen bool
	resizing   bool

	Actions chan Action

	Running bool

	cpu, ram, uptime string
	time             time.Time

	toastActive    bool
	lastToastError error
	lastToastTime  time.Time

	ecoMode bool

	resizeToMax bool
	firstOpen   bool

	fps struct {
		frames int
		max    int
		ticks  int
	}
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
	min := image.Pt(1080, 700)
	max := monitor.MainResolution().Max

	notify.System("Generating UI")

	Window = &GUI{
		is: is.Loading,

		Window: app.NewWindow(app.Title(title.Default), app.Decorated(false)),

		HWND: 0,

		min: min,
		max: max,

		size: min,

		Preview: true,
		Actions: make(chan Action, 1024),

		resizing: false,

		ecoMode: true,

		fps: struct {
			frames int
			max    int
			ticks  int
		}{0, 60, 0},

		cpu:    "0%",
		ram:    "0MB",
		uptime: "00:00",
	}

	Window.Bar = title.New(
		title.Default,
		fonts.NewCollection(),
		func() {
			Window.Perform(system.ActionMinimize)
		},
		func() {
			Window.maximize()
		},
		func() {
			Window.Perform(system.ActionClose)
		},
	)

	notify.System("Default dimensions detected: %s", max.String())

	go Window.loading()

	go Window.draw()
}

func (g *GUI) Open() {
	g.next(is.MainMenu)

	go func() {
		g.open = true

		g.Actions <- Refresh
		defer func() {
			g.Actions <- Closing
		}()

		for g.is != is.Closing {
			switch g.is {
			case is.Loading:
				notify.Debug("Loading...")
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

	if g.is != is.Closing {
		app.Main()
	}
}

func (g *GUI) SetWindowPos(shift image.Point) {
	if g.fullscreen || g.HWND == 0 || g.resizing {
		return
	}
	g.resizing = true

	go func() {
		defer func() { g.resizing = false }()

		if shift.Eq(g.shift) {
			return
		}

		pos := g.position().Add(shift)
		if !pos.In(image.Rectangle{Min: image.Pt(0, 0).Sub(g.size), Max: g.max.Add(g.size)}) {
			return
		}

		g.shift = shift

		wapi.SetWindowPosNoSize(g.HWND, pos)
	}()
}

func (g *GUI) draw() {
	for {
		tps := time.Second / time.Duration(g.fps.max+1)
		tick := time.NewTicker(tps)
		persecond := time.NewTicker(time.Second)
		g.fps.ticks = 0

		notify.Debug("Running at %dfps", g.fps.max)

		for fps := g.fps.max; fps == g.fps.max; {
			if g.resizing {
				continue
			}

			select {
			case <-persecond.C:
				g.fps.frames = g.fps.ticks
				g.fps.ticks = 0
			case <-tick.C:
				if g.fps.ticks < g.fps.max {
					g.Invalidate()
					g.fps.ticks++
				}
			}
		}
	}
}

func (g *GUI) frame(gtx layout.Context, e system.FrameEvent) {
	op.InvalidateOp{}.Add(gtx.Ops)

	e.Frame(gtx.Ops)

	p, ok := g.Bar.Dragging()
	if ok {
		g.SetWindowPos(p)
		return
	}
}

func (g *GUI) maximize() {
	defer func() { g.fullscreen = !g.fullscreen }()

	if g.fullscreen {
		g.Perform(system.ActionUnmaximize)
		g.size = g.min
		g.Option(app.Size(unit.Dp(g.min.X), unit.Dp(g.min.Y)))
	} else {
		g.Perform(system.ActionMaximize)
		g.size = g.max

		wapi.SetWindowPosShow(g.HWND, g.inset, g.max.Sub(g.inset))
	}
}

func (g *GUI) next(i is.Is) {
	notify.Debug("Next state set to \"%s\"", i)
	g.is = i
}

func (g *GUI) position() image.Point {
	r := &wapi.Rect{}
	wapi.GetWindowRect.Call(g.HWND, uintptr(unsafe.Pointer(r)))
	return image.Pt(int(r.Left), int(r.Top))
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

func (g *GUI) setInset(inset image.Point) {
	g.inset = g.inset.Add(inset)

	if g.fullscreen {
		wapi.SetWindowPosShow(g.HWND, g.inset.Sub(image.Pt(0, g.inset.Y)), image.Pt(g.size.X-g.inset.X, g.max.Y))
	} else {
		pos := g.position()

		// Move the main window over when space is not available for the inset.
		if pos.X < inset.X {
			pos.X += inset.X - pos.X
		}
		if pos.Y < inset.Y {
			pos.Y += inset.Y - pos.Y
		}
		// Shrink the main window when it exceeds max boundaries.
		if g.size.X+g.inset.X > g.max.X {
			g.size.X = g.max.X - g.inset.X
		}
		if g.size.Y+g.inset.Y > g.max.Y {
			g.size.Y = g.max.Y - g.inset.Y
		}

		wapi.SetWindowPosShow(g.HWND, pos, g.size)
	}
}

func (g *GUI) unsetInset(inset image.Point) {
	g.inset = g.inset.Sub(inset)

	min := g.position()
	max := g.size
	if g.fullscreen {
		min = image.Pt(0, 0)
		max = g.max
	}

	wapi.SetWindowPosShow(g.HWND, min, max)
}

func (g *GUI) toMain(next *string) {
	if *next == "" {
		*next = "main"
	}
}

func max(i, j int) int {
	return int(math.Max(float64(i), float64(j)))
}
