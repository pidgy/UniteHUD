package gui

import (
	"fmt"
	"image"
	"math"
	"os"
	"runtime"
	"syscall"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/fonts"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/stats"
	"github.com/pidgy/unitehud/video/monitor"
	"github.com/pidgy/unitehud/video/proc"
)

type Action string

type GUI struct {
	loaded bool

	is is.Is

	*app.Window
	*title.Bar

	min, max  image.Point
	size, pos image.Point
	shift     image.Point

	HWND uintptr

	Preview bool
	open    bool

	resizing bool

	resizex bool
	resized bool

	Actions chan Action

	Running bool

	cpu, ram, uptime string
	time             time.Time

	cascadia, normal, toast *material.Theme

	toastActive    bool
	lastToastError error
	lastToastTime  time.Time

	ecoMode bool

	resizeToMax bool

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
		pos:  max.Sub(min).Div(2),

		Preview: true,
		Actions: make(chan Action, 1024),

		resizing: false,
		resizex:  true,

		ecoMode: true,

		normal:   fonts.Default().Theme,
		cascadia: fonts.Cascadia().Theme,
		toast:    fonts.Default().Theme,

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
	defer os.Exit(0)

	g.next(is.MainMenu)
	//g.next(is.TabMenu)

	go func() {
		g.open = true

		g.Actions <- Refresh
		defer func() {
			g.Actions <- Closing
		}()

		for g.is != is.Closing {
			switch g.is {
			case is.TabMenu:
				g.tabs()
			case is.Loading:
				notify.Debug("still loading")
			case is.MainMenu:
				g.resizex = false
				g.main()
			case is.Projecting:
				g.resizex = true
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
	if g.fullscreen() || g.HWND == 0 || g.resizing {
		return
	}
	g.resizing = true

	go func() {
		defer func() { g.resizing = false }()

		if shift.Eq(g.shift) {
			return
		}

		pos := g.pos.Add(shift)
		if !pos.In(image.Rectangle{Min: image.Pt(0, 0).Sub(g.size), Max: g.max.Add(g.size)}) {
			return
		}

		g.pos = pos
		g.shift = shift

		proc.SetWindowPos.Call(g.HWND, uintptr(0), uintptr(g.pos.X), uintptr(g.pos.Y), 0, 0, uintptr(proc.SWPNoSize))
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

func (g *GUI) fullscreen() bool {
	return g.size == g.max
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
	if !g.fullscreen() {
		//max := monitor.MainResolution().Max
		//g.Option(app.MaxSize(unit.Dp(max.X), unit.Dp(max.Y)), app.Fullscreen.Option())

		g.Perform(system.ActionMaximize)
		g.size = g.max
		go proc.SetWindowPos.Call(g.HWND, uintptr(0), uintptr(0), uintptr(0), 0, 0, uintptr(proc.SWPNoSize))
		return
	}

	g.Perform(system.ActionUnmaximize)
	g.size = g.min
	g.Option(app.Size(unit.Dp(g.min.X), unit.Dp(g.min.Y)))
}

func (g *GUI) next(i is.Is) {
	notify.Debug("Next state set to \"%s\"", i)
	g.is = i
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

// func (g *GUI) setFPS(fps int) {
// 	if g.fps.max == fps {
// 		return
// 	}

// 	g.fps.max = fps

// 	go g.draw(fps)
// }

func (g *GUI) toMain(next *string) {
	if *next == "" {
		*next = "main"
	}
}

func max(i, j int) int {
	return int(math.Max(float64(i), float64(j)))
}
