package gui

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/rs/zerolog/log"
	"github.com/skratchdot/open-golang/open"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/gui/visual/area"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/dropdown"
	"github.com/pidgy/unitehud/gui/visual/help"
	"github.com/pidgy/unitehud/gui/visual/screen"
	"github.com/pidgy/unitehud/gui/visual/spinner"
	"github.com/pidgy/unitehud/gui/visual/split"
	"github.com/pidgy/unitehud/gui/visual/textblock"
	"github.com/pidgy/unitehud/history"
	"github.com/pidgy/unitehud/match"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/state"
	"github.com/pidgy/unitehud/stats"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/video"
	"github.com/pidgy/unitehud/video/device"
)

type GUI struct {
	*app.Window
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
}

type Action string

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
)

const Title = "UniteHUD Server (" + global.Version + ")"

var Window *GUI

func New() {
	Window = &GUI{
		Window: app.NewWindow(
			app.Title(Title),
			app.Size(
				unit.Px(975),
				unit.Px(715),
			),
		),
		Preview: true,
		Actions: make(chan Action, 1024),
		resize:  true,
		ecoMode: true,
	}

	cas, err := textblock.NewCascadiaCode()
	if err != nil {
		notify.Error("Failed to create CPU/RAM graph (%v)", err)
	}

	Window.cascadia = material.NewTheme(cas.Collection())
	Window.normal = material.NewTheme(gofont.Collection())
}

func (g *GUI) Open() {
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
					log.Err(err).Send()
				}
			case "configure":
				g.resize = true

				next, err = g.configure()
				if err != nil {
					log.Err(err).Send()
				}
			case "help_configure":
				h := help.Configuration()

				next, err = g.configurationHelpDialog(h.Help, h.Layout)
				if err != nil {
					log.Err(err).Send()
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

	procTicker := 3
	procTicks := 0

	peakCPU := 0.0
	peakRAM := 0.0

	for range time.NewTicker(time.Second).C {
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

		if procTicker == procTicks {

			procTicks = -1
		}
		procTicks++
	}
}

func (g *GUI) main() (next string, err error) {
	g.Window.Raise()

	split := &split.Vertical{
		Ratio: .70,
	}

	header := material.H5(g.normal, Title)
	header.Color = rgba.N(rgba.White)
	header.Alignment = text.Middle

	configButton := &button.Button{
		Text:     " Configure",
		Released: rgba.N(rgba.Background),
		Pressed:  rgba.N(rgba.DarkGray),
	}

	configButton.Click = func() {
		configButton.Active = false
		next = "configure"
	}

	spinRun := spinner.Running()
	defer spinRun.Stop()

	spinStop := spinner.Stopped()
	defer spinStop.Stop()

	spinRecord := spinner.Recording()
	defer spinRecord.Stop()

	reloadButton := &button.Button{
		Text:     "\tReload",
		Released: rgba.N(rgba.Background),
		Pressed:  rgba.N(rgba.DarkGray),
	}

	reloadButton.Click = func() {
		reloadButton.Active = !reloadButton.Active
		g.Actions <- Debug
	}

	startButton := &button.Button{
		Text:     "\t  Start",
		Released: rgba.N(rgba.Background),
		Pressed:  rgba.N(rgba.DarkGray),
	}

	stopButton := &button.Button{
		Text:     "\t  Stop",
		Disabled: true,
		Released: rgba.N(rgba.Disabled),
	}

	recordButton := &button.Button{
		Text:     "\tRecord",
		Disabled: true,
	}

	startButton.Click = func() {
		device.Hide = false

		g.Preview = false

		configButton.Active = true
		configButton.Disabled = true
		configButton.Released = rgba.N(rgba.Disabled)

		stopButton.Active = false
		stopButton.Disabled = false
		stopButton.Released = rgba.N(rgba.Background)

		startButton.Active = false
		startButton.Disabled = true
		startButton.Released = rgba.N(rgba.Disabled)

		recordButton.Active = false
		recordButton.Disabled = false
		recordButton.Released = rgba.N(rgba.Background)

		g.Actions <- Start
		g.Running = true
	}

	stopButton.Click = func() {
		device.Hide = true

		configButton.Active = false
		configButton.Disabled = false
		configButton.Released = rgba.N(rgba.Background)

		stopButton.Active = false
		stopButton.Disabled = true
		stopButton.Released = rgba.N(rgba.Disabled)

		startButton.Active = false
		startButton.Disabled = false
		startButton.Released = rgba.N(rgba.Background)

		recordButton.Active = false
		recordButton.Disabled = true
		recordButton.Released = rgba.N(rgba.Disabled)

		g.Actions <- Stop
		g.Running = false
	}

	updateRecordButton := func() {
		if config.Current.Record {
			recordButton.Text = " Recording"
			recordButton.Released = rgba.N(rgba.DarkRed)
		} else {
			recordButton.Text = "\tRecord"
			recordButton.Released = rgba.N(rgba.Background)
			if recordButton.Disabled {
				recordButton.Released = rgba.N(rgba.Disabled)
			}
		}
	}

	recordButton.Click = func() {
		recordButton.Active = !recordButton.Active
		g.Actions <- Record
	}

	openButton := &button.Button{
		Text:     "\t  Open",
		Released: rgba.N(rgba.Background),
		Pressed:  rgba.N(rgba.DarkGray),
	}

	openButton.Click = func() {
		openButton.Active = !openButton.Active

		g.Actions <- Open
	}

	notifyFeedTextBlock, err := textblock.NewCascadiaCodeSemiBold()
	if err != nil {
		notifyFeedTextBlock = &textblock.TextBlock{}
		notify.Error("Failed to load font: (%v)", err)
	}

	statsButton := &button.Button{
		Text:           "¼",
		Released:       rgba.N(rgba.CoolBlue),
		Pressed:        rgba.N(rgba.DarkGray),
		Size:           image.Pt(30, 16),
		TextSize:       unit.Sp(12),
		TextOffsetTop:  -4,
		TextOffsetLeft: 0,
		BorderWidth:    unit.Sp(.5),
	}

	statsButton.Click = func() {
		stats.Data()

		s, ok := state.Dump()
		if !ok {
			notify.Warn(s)
		} else {
			notify.System(s)
		}

		statsButton.Active = !statsButton.Active
	}

	historyButton := &button.Button{
		Text:           "±",
		Released:       rgba.N(rgba.BloodOrange),
		Pressed:        rgba.N(rgba.DarkGray),
		Size:           image.Pt(30, 15),
		TextSize:       unit.Sp(14),
		TextOffsetTop:  -6,
		TextOffsetLeft: 0,
		BorderWidth:    unit.Sp(.5),
	}

	historyButton.Click = func() {
		historyButton.Active = !historyButton.Active
		history.Dump()
	}

	obsButton := &button.Button{
		Text:           "obs",
		Released:       rgba.N(rgba.Purple),
		Pressed:        rgba.N(rgba.Purple),
		Size:           image.Pt(30, 15),
		TextSize:       unit.Sp(12),
		TextOffsetTop:  -5,
		TextOffsetLeft: -5,
		BorderWidth:    unit.Sp(.5),
	}

	obsButton.Click = func() {
		obsButton.Active = !obsButton.Active

		g.ToastOK("UniteHUD Overlay", "Drag \"UniteHUD Client\" into any OBS scene.", func() {
			ex, err := os.Executable()
			if err != nil {
				notify.Error("Failed to open www/ directory: %v", err)
				return
			}

			dir := filepath.Dir(ex)
			err = open.Run(dir + "/www")
			if err != nil {
				notify.Error("Failed to open www/ directory: %v", err)
				return
			}
		},
		)
	}

	clearButton := &button.Button{
		Text:           "cls",
		Released:       rgba.N(rgba.PaleRed),
		Pressed:        rgba.N(rgba.DarkRed),
		Size:           image.Pt(30, 15),
		TextSize:       unit.Sp(12),
		TextOffsetTop:  -4,
		TextOffsetLeft: -4,
		BorderWidth:    unit.Sp(.5),
	}

	clearButton.Click = func() {
		clearButton.Active = !clearButton.Active

		notify.CLS()
	}

	ecoButton := &button.Button{
		Text:           "eco",
		Released:       rgba.N(rgba.ForestGreen),
		Pressed:        rgba.N(rgba.DarkGray),
		Size:           image.Pt(30, 15),
		TextSize:       unit.Sp(12),
		TextOffsetTop:  -4,
		TextOffsetLeft: -6,
		BorderWidth:    unit.Sp(.5),
		Active:         !g.ecoMode,
	}

	ecoButton.Click = func() {
		g.ecoMode = !g.ecoMode
		ecoButton.Active = !g.ecoMode

		if g.ecoMode {
			notify.System("Resource saver has been enabled")
		} else {
			notify.System("Resource saver has been disabled")
		}
	}

	preview := &button.Image{
		Screen: &screen.Screen{
			Border:      true,
			BorderColor: rgba.N(rgba.Background),
		},
	}
	preview.Click = func() {
		preview.Hide = !preview.Hide
	}

	var ops op.Ops

	for next == "" {
		if !g.open {
			time.Sleep(time.Millisecond * 10)
			continue
		}

		if config.Current.Crashed != "" {
			g.ToastCrash(fmt.Sprintf("%s recently crashed for the following reason", Title), config.Current.Crashed)
			config.Current.Report("")
		}

		if g.ecoMode && state.Since() > time.Minute*30 && !stopButton.Disabled {
			notify.System("Idling for 30 minutes without activity, stopping to save resources")
			stopButton.Click()
		}

		e := <-g.Events()
		switch e := e.(type) {
		case app.ConfigEvent:
		case system.DestroyEvent:
			return "", e.Err
		case system.FrameEvent:
			g.Window.Option(app.Title(Title))

			gtx := layout.NewContext(&ops, e)
			pointer.CursorNameOp{Name: pointer.CursorGrab}.Add(gtx.Ops)

			background := clip.Rect{
				Max: gtx.Constraints.Max,
			}.Push(gtx.Ops)
			paint.ColorOp{Color: rgba.N(rgba.Background)}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			background.Pop()

			split.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return Fill(
						gtx,
						color.NRGBA{R: 25, G: 25, B: 25, A: 255},
						func(gtx layout.Context) layout.Dimensions {
							{
								layout.Inset{
									Left: unit.Px(2),
									Top:  unit.Px(5),
								}.Layout(gtx, header.Layout)

								win := config.Current.Window
								if config.Current.VideoCaptureDevice != config.NoVideoCaptureDevice {
									win = device.Name(config.Current.VideoCaptureDevice)
								}

								windowHeader := material.Caption(g.normal, win)
								windowHeader.Color = rgba.N(rgba.DarkSeafoam)
								windowHeader.Alignment = text.Middle
								windowHeader.Font.Weight = text.Bold

								if config.Current.LostWindow != "" {
									windowHeader = material.Caption(g.normal, config.Current.LostWindow)
									windowHeader.Color = rgba.N(rgba.PaleRed)
								}

								layout.Inset{
									Left: unit.Px(2),
									Top:  unit.Px(45),
								}.Layout(gtx, windowHeader.Layout)
							}
							{
								cpu := material.H5(g.normal, g.cpu)
								cpu.Color = rgba.N(rgba.White)
								cpu.Alignment = text.Middle
								cpu.TextSize = unit.Sp(11)

								layout.Inset{
									Top:  unit.Px(28),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 408)),
								}.Layout(gtx, cpu.Layout)

								cpuGraph := material.H5(g.cascadia, stats.CPUData())
								cpuGraph.Color = rgba.N(rgba.Background)
								cpuGraph.TextSize = unit.Sp(9)

								layout.Inset{
									Top:  unit.Px(1),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 450)),
								}.Layout(gtx, cpuGraph.Layout)

								ram := material.H5(g.normal, g.ram)
								ram.Color = rgba.N(rgba.White)
								ram.Alignment = text.Middle
								ram.TextSize = unit.Sp(11)

								layout.Inset{
									Top:  unit.Px(28),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 248)),
								}.Layout(gtx, ram.Layout)

								ramGraph := material.H5(g.cascadia, stats.RAMData())
								ramGraph.Color = rgba.N(rgba.Background)
								ramGraph.TextSize = unit.Sp(9)

								layout.Inset{
									Top:  unit.Px(1),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 300)),
								}.Layout(gtx, ramGraph.Layout)

								uptime := material.H5(g.normal, g.uptime)
								uptime.Color = rgba.N(rgba.SlateGray)
								uptime.Alignment = text.Middle
								uptime.TextSize = unit.Sp(12)

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 90)),
									Top:  unit.Px(34),
								}.Layout(gtx, uptime.Layout)

								h := fmt.Sprintf("%d/%2d", team.Balls.Holding, team.Balls.HoldingMax)
								if team.Balls.Holding < 10 {
									h = "0" + h
								}
								holding := material.H5(g.normal, h)
								holding.Color = rgba.N(team.Self.RGBA)
								holding.Alignment = text.Middle
								holding.TextSize = unit.Sp(13)

								layout.Inset{
									Top:  unit.Px(50),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 35)),
								}.Layout(gtx, holding.Layout)

								symbol := material.H5(g.normal, spinStop.Next())
								symbol.Alignment = text.Middle
								symbol.TextSize = unit.Sp(14)
								symbol.Font.Weight = text.ExtraBold
								symbol.Color = rgba.N(rgba.SlateGray)

								acronym := material.H5(g.normal, "STP")
								acronym.Alignment = text.Middle
								acronym.TextSize = unit.Sp(11)
								acronym.Color = rgba.N(rgba.SlateGray)

								down := float32(1)
								left := 1

								if config.Current.Record {
									symbol.Text = spinRecord.Next()
									symbol.Color = rgba.N(rgba.Red)
									symbol.TextSize = unit.Sp(20)
									acronym.Font.Weight = text.ExtraBold

									acronym.Text = "REC"
									acronym.Color = rgba.N(rgba.Red)

									left = 0
									down = 0
								} else if g.Running {
									symbol.Text = spinRun.Next()
									symbol.Color = rgba.N(rgba.Green)

									acronym.Text = "RUN"
									acronym.Color = rgba.N(rgba.Green)

									left = 3
									down = .5
								}

								layout.Inset{
									Top:  unit.Px(46 + down),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 145 - left)),
								}.Layout(gtx, symbol.Layout)

								layout.Inset{
									Top:  unit.Px(50),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 135)),
								}.Layout(gtx, acronym.Layout)

								if global.DebugMode {
									dbg := material.H5(g.normal, "DBG")
									dbg.Alignment = text.Middle
									dbg.TextSize = unit.Sp(11)
									dbg.Color = rgba.N(rgba.SeaBlue)

									layout.Inset{
										Top:  unit.Px(34),
										Left: unit.Px(float32(gtx.Constraints.Max.X - 135)),
									}.Layout(gtx, dbg.Layout)
								}
							}
							{
								o, p, s := server.Scores()

								purple := material.H5(g.normal, fmt.Sprintf("%d", p))
								purple.Color = rgba.N(team.Purple.RGBA)
								purple.Alignment = text.Middle
								purple.TextSize = unit.Sp(13)

								layout.Inset{
									Top:  unit.Px(2),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 35)),
								}.Layout(gtx, purple.Layout)

								orange := material.H5(g.normal, fmt.Sprintf("%d", o))
								orange.Color = rgba.N(team.Orange.RGBA)
								orange.Alignment = text.Middle
								orange.TextSize = unit.Sp(13)

								layout.Inset{
									Top:  unit.Px(18),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 35)),
								}.Layout(gtx, orange.Layout)

								self := material.H5(g.normal, strconv.Itoa(s))
								self.Color = rgba.N(team.Self.RGBA)
								self.Alignment = text.Middle
								self.TextSize = unit.Sp(13)

								layout.Inset{
									Top:  unit.Px(34),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 35)),
								}.Layout(gtx, self.Layout)

								clock := material.H5(g.normal, server.Clock())
								clock.Color = rgba.N(rgba.White)
								clock.Alignment = text.Middle
								clock.TextSize = unit.Sp(13)
								layout.Inset{
									Top:  unit.Px(2),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 90)),
								}.Layout(gtx, clock.Layout)
							}
							{
								clients := server.Clients()

								connectedClients := material.H5(g.normal, fmt.Sprintf("OBS %d", clients))
								connectedClients.Color = rgba.N(rgba.PaleRed)
								if clients > 0 {
									connectedClients.Color = rgba.N(rgba.Seafoam)
								}
								connectedClients.Alignment = text.Middle
								connectedClients.TextSize = unit.Sp(11)

								layout.Inset{
									Top:  unit.Px(50),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 90)),
								}.Layout(gtx, connectedClients.Layout)

							}
							{
								objs := server.Regielekis()

								for i, t := range objs {
									objective := material.H5(g.normal, "R")

									objective.Color = rgba.N(team.Color(t))
									objective.Alignment = text.Middle
									objective.TextSize = unit.Sp(12)

									layout.Inset{
										Top:  unit.Px(18),
										Left: unit.Px(float32(gtx.Constraints.Max.X-90) + float32(i*12)),
									}.Layout(gtx, objective.Layout)
								}
							}

							{
								layout.Inset{
									Top: unit.Px(65),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return notifyFeedTextBlock.Layout(gtx, notify.Feeds())
									},
								)
							}

							return layout.Dimensions{Size: gtx.Constraints.Max}
						},
					)
				},

				func(gtx layout.Context) layout.Dimensions {
					return Fill(
						gtx,
						color.NRGBA{R: 25, G: 25, B: 25, A: 255},
						func(gtx layout.Context) layout.Dimensions {
							{
								updateRecordButton()

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - statsButton.Size.X - 2)),
									Top:  unit.Px(float32(gtx.Constraints.Min.Y + 2)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return statsButton.Layout(gtx)
									},
								)

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - historyButton.Size.X*2 - 2)),
									Top:  unit.Px(float32(gtx.Constraints.Min.Y + 2)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return historyButton.Layout(gtx)
									},
								)

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - obsButton.Size.X - 2)),
									Top:  unit.Px(float32(gtx.Constraints.Min.Y + obsButton.Size.Y + 2)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return obsButton.Layout(gtx)
									},
								)

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - clearButton.Size.X*2 - 2)),
									Top:  unit.Px(float32(gtx.Constraints.Min.Y + clearButton.Size.Y + 2)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return clearButton.Layout(gtx)
									},
								)

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - ecoButton.Size.X - 2)),
									Top:  unit.Px(float32(gtx.Constraints.Min.Y + ecoButton.Size.Y*2 + 2)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return ecoButton.Layout(gtx)
									},
								)
							}
							// Right-side buttons.
							{
								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
									Top:  unit.Px(float32(gtx.Constraints.Max.Y - 335)),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									// Assigned from captureWindow.
									preview.SetImage(notify.Preview)

									return preview.Layout(g.cascadia, gtx)
								},
								)

								if global.DebugMode {
									layout.Inset{
										Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
										Top:  unit.Px(float32(gtx.Constraints.Max.Y - 390)),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return reloadButton.Layout(gtx)
										},
									)
								}

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
									Top:  unit.Px(float32(gtx.Constraints.Max.Y - 265)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return startButton.Layout(gtx)
									},
								)

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
									Top:  unit.Px(float32(gtx.Constraints.Max.Y - 210)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return stopButton.Layout(gtx)
									},
								)

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
									Top:  unit.Px(float32(gtx.Constraints.Max.Y - 155)),
								}.Layout(gtx,
									func(gtx layout.Context) layout.Dimensions {
										return configButton.Layout(gtx)
									},
								)

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
									Top:  unit.Px(float32(gtx.Constraints.Max.Y - 100)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return recordButton.Layout(gtx)
									},
								)

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
									Top:  unit.Px(float32(gtx.Constraints.Max.Y - 45)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return openButton.Layout(gtx)
									},
								)
							}
							// Event images.
							{
								layout.Inset{
									Top:  unit.Px(65),
									Left: unit.Px(5),
								}.Layout(gtx, (&screen.Screen{
									Border:      true,
									BorderColor: rgba.N(team.Purple.RGBA),
									Image:       notify.PurpleScore,
								}).Layout)

								layout.Inset{
									Top:  unit.Px(120),
									Left: unit.Px(5),
								}.Layout(gtx, (&screen.Screen{
									Border:      true,
									BorderColor: rgba.N(team.Orange.RGBA),
									Image:       notify.OrangeScore,
								}).Layout)

								layout.Inset{
									Top:  unit.Px(175),
									Left: unit.Px(5),
								}.Layout(gtx, (&screen.Screen{
									Border:      true,
									BorderColor: rgba.N(team.Self.RGBA),
									Image:       notify.Balls,
									ScaleX:      2,
									ScaleY:      2,
								}).Layout)

								layout.Inset{
									Top:  unit.Px(232),
									Left: unit.Px(5),
								}.Layout(gtx, (&screen.Screen{
									Border:      true,
									BorderColor: rgba.N(team.Self.RGBA),
									Image:       notify.SelfScore,
									ScaleX:      4,
									ScaleY:      4,
								}).Layout)

								layout.Inset{
									Top:  unit.Px(175),
									Left: unit.Px(68),
								}.Layout(gtx, (&screen.Screen{
									Image:       notify.Time,
									Border:      true,
									BorderColor: rgba.N(team.Time.RGBA),
									ScaleX:      2,
									ScaleY:      2,
								}).Layout)
							}

							return layout.Dimensions{Size: gtx.Constraints.Max}
						},
					)
				},
			)

			e.Frame(gtx.Ops)
		}
	}

	return next, nil
}

func (g *GUI) display(src image.Image) {
	g.Screen = &screen.Screen{
		Image:  src,
		ScaleX: 2,
		ScaleY: 2,
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
	once := true
	for {
		if g.Preview {
			img, err := video.Capture()
			if err != nil {
				g.ToastError(err)
			}

			g.display(img)
			if once {
				once = !once
			}
		}

		// Redraw the image.
		g.Invalidate()

		time.Sleep(time.Millisecond * 100)
	}
}

func (g *GUI) configure() (next string, err error) {
	g.Preview = true
	device.Hide = false
	defer func() {
		g.Preview = false
		device.Hide = true
	}()

	split := &split.Horizontal{
		Ratio: .6,
	}

	ballsArea := &area.Area{
		Text:     "\tBalls",
		TextSize: unit.Sp(13),
		Min:      config.Current.Balls.Min.Div(2),
		Max:      config.Current.Balls.Max.Div(2),

		Button: &button.Button{
			Active:   true,
			Text:     "\t  Balls",
			Pressed:  rgba.N(rgba.Background),
			Released: rgba.N(rgba.DarkGray),
			Size:     image.Pt(100, 30),
		},
	}

	ballsArea.Button.Click = func() {
		if !ballsArea.Button.Active {
			ballsArea.Text = "\tBalls (Locked)"
			ballsArea.Button.Text = "\tLocked"
			ballsArea.NRGBA.A = 0x9
			return
		}

		ballsArea.Text = "\tBalls"
		ballsArea.Button.Text = "\t  Balls"
		ballsArea.NRGBA.A = 0x4F
	}

	timeArea := &area.Area{
		Text:     "\tTime",
		TextSize: unit.Sp(13),
		Min:      config.Current.Time.Min.Div(2),
		Max:      config.Current.Time.Max.Div(2),
		Button: &button.Button{
			Active:   true,
			Text:     "\t  Time",
			Pressed:  rgba.N(rgba.Background),
			Released: rgba.N(rgba.DarkGray),
			Size:     image.Pt(100, 30),
		},
	}

	timeArea.Button.Click = func() {
		if !timeArea.Button.Active {
			timeArea.Text = "Time (Locked)"
			timeArea.Button.Text = "\tLocked"
			timeArea.NRGBA.A = 0x9
			return
		}

		timeArea.Text = "\tTime"
		timeArea.Button.Text = "\t  Time"
		timeArea.NRGBA.A = 0x4F
	}

	scoreArea := &area.Area{
		Text:  "Score",
		Min:   config.Current.Scores.Min.Div(2),
		Max:   config.Current.Scores.Max.Div(2),
		Theme: g.normal,

		Button: &button.Button{
			Active:   true,
			Text:     "\t Score",
			Pressed:  rgba.N(rgba.Background),
			Released: rgba.N(rgba.DarkGray),
			Size:     image.Pt(100, 30),
		},
	}

	scoreArea.Button.Click = func() {
		if !scoreArea.Button.Active {
			scoreArea.Text = "Score (Locked)"
			scoreArea.Button.Text = "\tLocked"
			scoreArea.NRGBA.A = 0x9
			return
		}

		scoreArea.Text = "Score"
		scoreArea.Button.Text = "\t Score"
		scoreArea.NRGBA.A = 0x4F
	}

	scaleText := material.H5(g.cascadia, "Scale")
	scaleText.Color = rgba.N(rgba.White)
	scaleText.Alignment = text.Middle
	scaleText.TextSize = unit.Sp(11)

	scaleValueText := material.H5(g.cascadia, "1x")
	scaleValueText.Color = rgba.N(rgba.White)
	scaleValueText.Alignment = text.Middle
	scaleValueText.TextSize = unit.Sp(11)

	scaleUpButton := &button.Button{
		Text:     "+",
		Pressed:  color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released: color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:     image.Pt(30, 30),
		TextSize: unit.Sp(12),
	}

	scaleDownButton := &button.Button{
		Text:     "-",
		Pressed:  color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released: color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:     image.Pt(30, 30),
		TextSize: unit.Sp(12),
	}

	scaleUpButton.Click = func() {
		g.Preview = false
		defer g.buttonSpam(scaleUpButton)

		scaleUpButton.Active = !scaleUpButton.Active
		config.Current.Scale += .01
	}

	scaleDownButton.Click = func() {
		g.Preview = false
		defer g.buttonSpam(scaleDownButton)

		scaleDownButton.Active = !scaleDownButton.Active
		config.Current.Scale -= .01
		if config.Current.Scale < 1 {
			config.Current.Scale = 1
		}
	}

	nButton := &button.Button{
		Text:        "^",
		Pressed:     color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released:    color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:        image.Pt(30, 20),
		TextSize:    unit.Sp(18),
		SingleClick: true,
		Click: func() {
			config.Current.Shift.N++
		},
	}

	eButton := &button.Button{
		Text:          ">",
		Pressed:       color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released:      color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:          image.Pt(28, 23),
		TextSize:      unit.Sp(12),
		TextOffsetTop: -2,
		SingleClick:   true,
		Click: func() {
			config.Current.Shift.E++
		},
	}

	sButton := &button.Button{
		Text:          "v",
		Pressed:       color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released:      color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:          image.Pt(30, 20),
		TextSize:      unit.Sp(12),
		TextOffsetTop: -2,
		SingleClick:   true,
		Click: func() {
			config.Current.Shift.S++
		},
	}

	wButton := &button.Button{
		Text:          "<",
		Pressed:       color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released:      color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:          image.Pt(28, 23),
		TextSize:      unit.Sp(12),
		TextOffsetTop: -2,
		SingleClick:   true,
		Click: func() {
			config.Current.Shift.W++
		},
	}

	shiftText := material.H5(g.cascadia, "Shift")
	shiftText.Color = rgba.N(rgba.White)
	shiftText.Alignment = text.Middle
	shiftText.TextSize = unit.Sp(11)

	captureButton := &button.Button{
		Active:      true,
		Text:        "\tCapture",
		Pressed:     rgba.N(rgba.Background),
		Released:    rgba.N(rgba.DarkGray),
		Size:        image.Pt(100, 30),
		SingleClick: true,
		Click: func() {
			dir, err := os.Getwd()
			if err != nil {
				g.ToastOK("Error", "Failed to find current directory")
				return
			}

			for _, fr := range []struct {
				area string
				file string
				rect image.Rectangle
			}{
				{"entire area", "screen_area.png", g.Screen.Bounds()},
				{"Score area", "score_area.png", scoreArea.Rectangle()},
				{"Balls area", "balls_area.png", ballsArea.Rectangle()},
				{"Time area", "time_area.png", timeArea.Rectangle()},
			} {
				noq := make(chan bool)
				g.toastActive = false
				g.ToastYesNo("Capture", fmt.Sprintf("Capture %s?", fr.area), func() { noq <- false }, func() { noq <- true })

				if <-noq {
					continue
				}

				img, err := video.CaptureRect(fr.rect)
				if err != nil {
					g.ToastOK("Error", fmt.Sprintf("Failed to capture %s", fr.file))
					return
				}

				matrix, err := gocv.ImageToMatRGB(img)
				if err != nil {
					g.ToastOK("Error", fmt.Sprintf("Failed to create %s", fr.file))
					return
				}
				defer matrix.Close()

				if !gocv.IMWrite(fr.file, matrix) {
					g.ToastOK("Error", fmt.Sprintf("Failed to save %s", fr.file))
					return
				}

				var sI syscall.StartupInfo
				var pI syscall.ProcessInformation
				argv := syscall.StringToUTF16Ptr(os.Getenv("windir") + "\\system32\\cmd.exe /C " +
					fmt.Sprintf("\"%s\\%s\"", dir, fr.file))

				err = syscall.CreateProcess(nil, argv, nil, nil, true, 0, nil, nil, &sI, &pI)
				if err != nil {
					g.ToastOK("Error", fmt.Sprintf("Failed to open %s", fr.file))
					return
				}
			}
		},
	}

	mapArea := &area.Area{
		Text:     "\t  Map",
		TextSize: unit.Sp(13),
		Min:      config.Current.Map.Min.Div(2),
		Max:      config.Current.Map.Max.Div(2),

		Button: &button.Button{
			Active:   true,
			Text:     "\t  tMap",
			Pressed:  rgba.N(rgba.Background),
			Released: rgba.N(rgba.DarkGray),
			Size:     image.Pt(100, 30),
		},
	}

	resizeButton := &button.Button{
		Text:     "\tResize",
		Pressed:  rgba.N(rgba.Background),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
	}

	resizeButton.Click = func() {
		resizeButton.Active = !resizeButton.Active

		err := video.Resize16x9()
		if err != nil {
			resizeButton.Error()
			notify.Error("%v", err)
			return
		}
	}

	defaultButton := &button.Button{
		Text:     "\tDefault",
		Pressed:  rgba.N(rgba.Background),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
	}

	defaultButton.Click = func() {
		defaultButton.Active = !defaultButton.Active

		config.Current.SetDefaultAreas()

		ballsArea.Min = config.Current.Balls.Min.Div(2)
		ballsArea.Max = config.Current.Balls.Max.Div(2)
		scoreArea.Min = config.Current.Scores.Min.Div(2)
		scoreArea.Max = config.Current.Scores.Max.Div(2)
		mapArea.Min = config.Current.Map.Min.Div(2)
		mapArea.Max = config.Current.Map.Max.Div(2)
		timeArea.Min = config.Current.Time.Min.Div(2)
		timeArea.Max = config.Current.Time.Max.Div(2)
	}

	openConfigFileButton := &button.Button{
		Text:     "\t   Edit",
		Pressed:  rgba.N(rgba.Background),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
	}

	openConfigFileButton.Click = func() {
		openConfigFileButton.Active = !openConfigFileButton.Active

		exe := "C:\\Windows\\system32\\notepad.exe"
		err := exec.Command(exe, config.File).Run()
		if err != nil {
			notify.Error("Failed to open \"%s\" (%v)", config.File, err)
			return
		}

		// Called once window is closed.
		err = config.Load()
		if err != nil {
			notify.Error("Failed to reload \"%s\" (%v)", config.File, err)
			return
		}
	}

	saveButton := &button.Button{
		Text:     "\t  Save",
		Pressed:  rgba.N(rgba.Background),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
	}

	cancelButton := &button.Button{
		Text:     "\tCancel",
		Pressed:  rgba.N(rgba.Background),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
	}

	// Hold on to the previous configuration to overwrite memory saves.
	prev := config.Current

	cancelButton.Click = func() {
		g.ToastYesNo("Cancel", "Discard configuration changes?",
			func() {
				cancelButton.Active = !cancelButton.Active

				server.Clear()

				cancelButton.Disabled = true
				saveButton.Disabled = true
				ballsArea.Button.Disabled = true
				timeArea.Button.Disabled = true
				scoreArea.Button.Disabled = true

				config.Current = prev
				err := config.Current.Save()
				if err != nil {
					notify.Error("Failed to save UniteHUD configuration (%v)")
				}

				notify.System("Configuration omitted")

				g.Actions <- Refresh

				next = "main"
			}, func() {
				cancelButton.Active = !cancelButton.Active
			},
		)
	}

	saveButton.Click = func() {
		g.ToastYesNo("Save", "Save configuration changes?",
			func() {
				saveButton.Active = !saveButton.Active

				server.Clear()

				cancelButton.Disabled = true
				saveButton.Disabled = true
				ballsArea.Button.Disabled = true
				timeArea.Button.Disabled = true
				scoreArea.Button.Disabled = true

				config.Current.Scores = scoreArea.Rectangle()
				config.Current.Time = timeArea.Rectangle()
				config.Current.Balls = ballsArea.Rectangle()
				config.Current.Map = mapArea.Rectangle()

				err := config.Current.Save()
				if err != nil {
					notify.Error("Failed to save UniteHUD configuration (%v)")
				}

				notify.System("Configuration saved to " + config.File)

				g.Actions <- Refresh

				next = "main"
			}, func() {
				saveButton.Active = !saveButton.Active
			},
		)
	}

	screenButton := &button.Button{
		Text:     "\tPreview",
		Pressed:  rgba.N(rgba.Background),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
	}

	screenButton.Click = func() {
		g.Preview = !g.Preview
	}

	windowList := &dropdown.List{
		Items: []*dropdown.Item{},
		Callback: func(i *dropdown.Item) {
			if config.Current.VideoCaptureDevice != config.NoVideoCaptureDevice {
				return
			}

			if i.Text == "" {
				config.Current.Window = config.MainDisplay
				return
			}

			config.Current.Window = i.Text
		},
		WidthModifier: 2,
	}

	populateWindows := func(videoCaptureDisabledEvent bool) {
		windows, _ := video.Sources()

		if len(windows) == len(windowList.Items) && !videoCaptureDisabledEvent {
			return
		}

		windowList.Items = windowList.Items[:0]

		if videoCaptureDisabledEvent && config.Current.Window == "" {
			config.Current.Window = config.MainDisplay
		}

		for _, win := range windows {
			windowList.Items = append(windowList.Items,
				&dropdown.Item{
					Text:     win,
					Disabled: config.Current.VideoCaptureDevice != config.NoVideoCaptureDevice,
					Checked:  widget.Bool{Value: win == config.Current.Window},
				},
			)
		}

		if config.Current.VideoCaptureDevice == config.NoVideoCaptureDevice {
			// populateWindows(false)
			windowList.Items[0].Disabled = false
		} else {
			// unpopulateWindows()
			windowList.Items[0].Disabled = true
		}
	}

	populateWindows(false)

	for _, i := range windowList.Items {
		if i.Text == config.Current.Window {
			i.Checked.Value = true
		}
	}

	deviceList := &dropdown.List{}

	populateDevices := func(videoCaptureDisabledEvent bool) {
		_, devices := video.Sources()

		if len(devices)+1 == len(deviceList.Items) && !videoCaptureDisabledEvent {
			return
		}

		deviceList.Items = []*dropdown.Item{
			{
				Text:  "Disabled",
				Value: config.NoVideoCaptureDevice,
			},
		}

		for _, d := range devices {
			deviceList.Items = append(deviceList.Items, &dropdown.Item{
				Text:  device.Name(d),
				Value: d,
			},
			)
		}

		for _, i := range deviceList.Items {
			i.Checked.Value = false

			if i.Value == config.Current.VideoCaptureDevice {
				i.Checked.Value = true
			}
		}
	}

	deviceList = &dropdown.List{
		Items: []*dropdown.Item{
			{
				Text:  "Disabled",
				Value: config.NoVideoCaptureDevice,
			},
		},
		Callback: func(i *dropdown.Item) {
			video.Close()
			log.Debug().Msg("here")
			time.Sleep(time.Second) // XXX: Fix concurrency error in device.go Close.

			config.Current.VideoCaptureDevice = i.Value

			if i.Text == "Disabled" {
				i.Checked = widget.Bool{Value: true}
				populateWindows(true)
			} else {
				populateWindows(false)
			}

			log.Debug().Int("device", i.Value).Msg("selected video capture device")

			go func() {
				err = video.Load()
				if err != nil {
					g.ToastErrorForce(err)

					config.Current.Window = config.MainDisplay
					config.Current.VideoCaptureDevice = config.NoVideoCaptureDevice

					populateWindows(true)
					populateDevices(true)
					return
				}
			}()
		},
		WidthModifier: 1,
	}

	populateDevices(false)

	resetButton := &button.Button{
		Text:     "\t Reset",
		Pressed:  rgba.N(rgba.Background),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
	}

	resetButton.Click = func() {
		g.ToastYesNo("Reset", "Reset your UniteHUD configuration?", func() {
			resetButton.Active = !resetButton.Active

			deviceList.Callback(deviceList.Items[0])

			err := config.Reset()
			if err != nil {
				notify.Error("Failed to reset configuration (%v)", err)
			}

			config.Current.Reload()

			ballsArea.Min, ballsArea.Max = config.Current.Balls.Min.Div(2), config.Current.Balls.Max.Div(2)
			timeArea.Min, timeArea.Max = config.Current.Time.Min.Div(2), config.Current.Time.Max.Div(2)
			scoreArea.Min, scoreArea.Max = config.Current.Scores.Min.Div(2), config.Current.Scores.Max.Div(2)

			populateWindows(true)
			populateDevices(true)

			saveButton.Click()
		}, func() {
			resetButton.Active = !resetButton.Active
		},
		)
	}

	header := material.H5(g.cascadia, Title)
	header.Color = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	header.Alignment = text.Middle
	header.Font.Weight = text.ExtraBold

	pauseMatchingRoutines := false
	defer func() { pauseMatchingRoutines = true }()
	go g.while(func() { g.matchScore(scoreArea) }, &pauseMatchingRoutines)
	go g.while(func() { g.matchTime(timeArea) }, &pauseMatchingRoutines)
	go g.while(func() { g.matchBalls(ballsArea) }, &pauseMatchingRoutines)
	// go g.run(func() { g.matchMap(mapArea) }, &kill)

	var ops op.Ops

	for next == "" {
		if !g.open {
			time.Sleep(time.Millisecond * 10)
			continue
		}

		populateWindows(false)
		populateDevices(false)

		e := <-g.Events()
		switch e := e.(type) {
		case app.ConfigEvent:
		case system.DestroyEvent:
			return "", nil
		case system.FrameEvent:
			g.Window.Option(app.Title(fmt.Sprintf("%s (%s %s)", Title, g.cpu, g.ram)))

			gtx := layout.NewContext(&ops, e)
			pointer.CursorNameOp{Name: pointer.CursorGrab}.Add(gtx.Ops)

			background := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
			paint.ColorOp{Color: rgba.N(rgba.Background)}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			background.Pop()

			split.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(5)).Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							if !screenButton.Active {
								return layout.Dimensions{Size: gtx.Constraints.Max}
							}

							return Fill(
								gtx,
								color.NRGBA{R: 25, G: 25, B: 25, A: 255},
								g.Screen.Layout)
						},
					)
				},

				func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(5)).Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							return Fill(gtx, color.NRGBA{R: 25, G: 25, B: 25, A: 255},
								func(gtx layout.Context) layout.Dimensions {
									{
										layout.Inset{
											Left:  unit.Px(10),
											Top:   unit.Px(101),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return saveButton.Layout(gtx)
											},
										)

										layout.Inset{
											Left:  unit.Px(115),
											Top:   unit.Px(101),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return cancelButton.Layout(gtx)
											},
										)

										layout.Inset{
											Top:   unit.Px(5),
											Left:  unit.Px(325),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return resizeButton.Layout(gtx)
											},
										)

										layout.Inset{
											Top:   unit.Px(37),
											Left:  unit.Px(325),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return defaultButton.Layout(gtx)
											},
										)

										layout.Inset{
											Left:  unit.Px(220),
											Top:   unit.Px(101),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return resetButton.Layout(gtx)
											},
										)

										layout.Inset{
											Left:  unit.Px(325),
											Top:   unit.Px(69),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return openConfigFileButton.Layout(gtx)
											},
										)

										layout.Inset{
											Left:  unit.Px(325),
											Top:   unit.Px(101),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return screenButton.Layout(gtx)
											},
										)
									}

									// Capture video.
									{
										layout.Inset{
											Left:  unit.Px(float32(gtx.Constraints.Max.X - 519)),
											Top:   unit.Px(3),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												windowListTitle := material.Label(g.cascadia, unit.Px(14), "Window")
												windowListTitle.Color = rgba.N(rgba.SlateGray)
												return windowListTitle.Layout(gtx)
											},
										)

										layout.Inset{
											Left:   unit.Px(float32(gtx.Constraints.Max.X - 520)),
											Top:    unit.Px(20),
											Right:  unit.Px(10),
											Bottom: unit.Px(3),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return widget.Border{
													Color: color.NRGBA{R: 100, G: 100, B: 100, A: 50},
													Width: unit.Px(2),
												}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													return windowList.Layout(gtx, g.normal)
												},
												)
											},
										)
									}

									{
										layout.Inset{
											Left:  unit.Px(float32(gtx.Constraints.Max.X - 249)),
											Top:   unit.Px(3),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												deviceListTitle := material.Label(g.cascadia, unit.Px(14), "Video Capture Device")
												deviceListTitle.Color = rgba.N(rgba.SlateGray)
												return deviceListTitle.Layout(gtx)
											},
										)

										layout.Inset{
											Left:   unit.Px(float32(gtx.Constraints.Max.X - 250)),
											Top:    unit.Px(20),
											Right:  unit.Px(10),
											Bottom: unit.Px(3),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return widget.Border{
													Color: color.NRGBA{R: 100, G: 100, B: 100, A: 50},
													Width: unit.Px(2),
												}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													return deviceList.Layout(gtx, g.normal)
												},
												)
											},
										)
									}

									// Time area rectangle buttons.
									{
										layout.Inset{
											Top:  unit.Px(5),
											Left: unit.Px(220),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return timeArea.Button.Layout(gtx)
											},
										)
									}

									layout.Inset{
										Top:  unit.Px(69),
										Left: unit.Px(220),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return captureButton.Layout(gtx)
										},
									)

									// Balls area rectangle buttons.
									{
										layout.Inset{
											Left: unit.Px(115),
											Top:  unit.Px(5),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return ballsArea.Button.Layout(gtx)
											},
										)
									}

									// Score area rectangle buttons.
									{
										if config.Current.VideoCaptureDevice != config.NoVideoCaptureDevice || config.Current.Window == config.MainDisplay {
											scaleText.Color = rgba.N(rgba.SlateGray)
											scaleValueText.Color = rgba.N(rgba.SlateGray)
											scaleUpButton.Disabled = true
											scaleDownButton.Disabled = true
										} else {
											scaleText.Color = rgba.N(rgba.White)
											scaleValueText.Color = rgba.N(rgba.White)
											scaleUpButton.Disabled = false
											scaleDownButton.Disabled = false
										}

										layout.Inset{
											Left: unit.Px(10),
											Top:  unit.Px(5),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return scoreArea.Button.Layout(gtx)
											},
										)

										layout.Inset{
											Left: unit.Px(10),
											Top:  unit.Px(38),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return scaleDownButton.Layout(gtx)
											},
										)

										layout.Inset{
											Left: unit.Px(45),
											Top:  unit.Px(38),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return scaleText.Layout(gtx)
											},
										)

										layout.Inset{
											Left: unit.Px(45),
											Top:  unit.Px(55),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												if config.Current.VideoCaptureDevice == config.NoVideoCaptureDevice {
													scaleValueText.Text = fmt.Sprintf("%.2fx", config.Current.Scale)
												} else {
													scaleValueText.Text = "1.00x"
												}

												return scaleValueText.Layout(gtx)
											},
										)

										layout.Inset{
											Left: unit.Px(80),
											Top:  unit.Px(38),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return scaleUpButton.Layout(gtx)
											},
										)
									}

									// Shift N,E,S,W
									{
										if config.Current.VideoCaptureDevice != config.NoVideoCaptureDevice || config.Current.Window == config.MainDisplay || config.Current.Scale == 1 {
											nButton.Disabled = true
											eButton.Disabled = true
											sButton.Disabled = true
											wButton.Disabled = true
											shiftText.Color = rgba.N(rgba.SlateGray)
										} else {
											nButton.Disabled = false
											eButton.Disabled = false
											sButton.Disabled = false
											wButton.Disabled = false
											shiftText.Color = rgba.N(rgba.White)
										}

										layout.Inset{
											Left: unit.Px(150),
											Top:  unit.Px(37),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return nButton.Layout(gtx)
											},
										)

										layout.Inset{
											Left: unit.Px(184),
											Top:  unit.Px(55),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return eButton.Layout(gtx)
											},
										)

										layout.Inset{
											Left: unit.Px(150),
											Top:  unit.Px(78),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return sButton.Layout(gtx)
											},
										)

										layout.Inset{
											Left: unit.Px(118),
											Top:  unit.Px(55),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return wButton.Layout(gtx)
											},
										)

										layout.Inset{
											Left: unit.Px(150),
											Top:  unit.Px(60),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return shiftText.Layout(gtx)
											},
										)
									}

									return layout.Dimensions{Size: gtx.Constraints.Max}
								},
							)
						},
					)
				},
			)

			scoreArea.Layout(gtx)
			ballsArea.Layout(gtx)
			timeArea.Layout(gtx)
			// mapArea.Layout(gtx)

			e.Frame(gtx.Ops)
		}
	}

	return next, nil
}

func (g *GUI) configurationHelpDialog(h *help.Help, widget layout.Widget) (next string, err error) {
	split := &split.Vertical{Ratio: .70}

	var ops op.Ops

	header := material.H5(g.cascadia, "Help: Configuration")
	header.Color = rgba.N(rgba.White)
	header.Alignment = text.Middle

	backwardButton := &button.Button{
		Text:     " <",
		Released: rgba.N(rgba.SlateGray),
		Pressed:  rgba.N(rgba.DarkGray),
		Size:     image.Pt(40, 35),
	}

	backwardButton.Click = func() {
		backwardButton.Active = !backwardButton.Active
		if h.Page != 0 {
			h.Page--
		}
	}

	forwardButton := &button.Button{
		Text:     " >",
		Released: rgba.N(rgba.SlateGray),
		Pressed:  rgba.N(rgba.DarkGray),
		Size:     image.Pt(40, 35),
	}

	forwardButton.Click = func() {
		forwardButton.Active = !forwardButton.Active
		if h.Page != h.Pages-1 {
			h.Page++
		}
	}

	returnButton := &button.Button{
		Text:     "\t  Back",
		Released: rgba.N(rgba.SlateGray),
		Pressed:  rgba.N(rgba.DarkGray),
	}

	returnButton.Click = func() {
		returnButton.Active = false
		next = "configure"
	}

	for next == "" {
		if !g.open {
			time.Sleep(time.Millisecond * 10)
			continue
		}

		e := <-g.Events()
		switch e := e.(type) {
		case app.ConfigEvent:
		case system.DestroyEvent:
			return "", e.Err
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)
			pointer.CursorNameOp{Name: pointer.CursorGrab}.Add(gtx.Ops)

			background := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
			paint.ColorOp{Color: rgba.N(rgba.Background)}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			background.Pop()

			split.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return Fill(gtx,
						color.NRGBA{R: 25, G: 25, B: 25, A: 255},
						func(gtx layout.Context) layout.Dimensions {
							layout.Inset{
								Left: unit.Px(2),
								Top:  unit.Px(10),
							}.Layout(gtx, header.Layout)

							return layout.Inset{Top: unit.Px(50)}.Layout(gtx,
								func(gtx layout.Context) layout.Dimensions {
									return widget(gtx)
								},
							)
						},
					)
				},

				func(gtx layout.Context) layout.Dimensions {
					return Fill(
						gtx,
						color.NRGBA{R: 25, G: 25, B: 25, A: 255},
						func(gtx layout.Context) layout.Dimensions {
							pages := material.H5(g.cascadia, fmt.Sprintf("%d / %d", h.Page+1, h.Pages))
							pages.Color = rgba.N(rgba.White)
							pages.Alignment = text.Middle
							pages.TextSize = unit.Sp(14)
							layout.Inset{
								Left: unit.Px(float32(gtx.Constraints.Max.X - 90)),
								Top:  unit.Px(float32(gtx.Constraints.Max.Y - 130)),
							}.Layout(gtx, pages.Layout)

							layout.Inset{
								Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
								Top:  unit.Px(float32(gtx.Constraints.Max.Y - 100)),
							}.Layout(gtx,
								func(gtx layout.Context) layout.Dimensions {
									return backwardButton.Layout(gtx)
								},
							)

							layout.Inset{
								Left: unit.Px(float32(gtx.Constraints.Max.X - 65)),
								Top:  unit.Px(float32(gtx.Constraints.Max.Y - 100)),
							}.Layout(gtx,
								func(gtx layout.Context) layout.Dimensions {
									return forwardButton.Layout(gtx)
								},
							)

							layout.Inset{
								Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
								Top:  unit.Px(float32(gtx.Constraints.Max.Y - 45)),
							}.Layout(gtx,
								func(gtx layout.Context) layout.Dimensions {
									return returnButton.Layout(gtx)
								},
							)

							return layout.Dimensions{Size: gtx.Constraints.Max}
						},
					)
				},
			)

			e.Frame(gtx.Ops)
		}
	}

	return next, nil
}

// buttonSpam ensures we only call reload once for multiple button presses.
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

func (g *GUI) ToastCrash(msg, reason string, callbacks ...func()) {
	go func() {
		g.toastActive = true
		defer func() { g.toastActive = false }()

		dx, dy := float32(500), float32(125)

		w := app.NewWindow(
			app.Title("Crash Report"),
			app.Size(unit.Px(dx), unit.Px(dy)),
			app.MaxSize(unit.Px(dx), unit.Px(dy)),
			app.MinSize(unit.Px(dx), unit.Px(dy)),
		)

		m := material.Label(g.normal, g.normal.TextSize.Scale(15.0/16.0), msg)
		m.Color = rgba.N(rgba.White)
		m.Alignment = text.Middle

		c := material.Label(g.normal, g.normal.TextSize.Scale(15.0/16.0), reason)
		c.Color = rgba.N(rgba.PaleRed)
		c.Alignment = text.Middle

		var ops op.Ops

		for e := range w.Events() {
			if _, ok := e.(system.DestroyEvent); ok && len(callbacks) > 0 {
				for _, cb := range callbacks {
					cb()
				}
			}

			if e, ok := e.(system.FrameEvent); ok {
				gtx := layout.NewContext(&ops, e)

				ops.Reset()

				ColorBox(gtx, gtx.Constraints.Max, color.NRGBA{R: 25, G: 25, B: 25, A: 255})

				layout.Inset{
					Top: unit.Px(15),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return m.Layout(gtx)
					},
				)

				layout.Inset{
					Top: unit.Px(40),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return c.Layout(gtx)
					},
				)

				w.Center()
				w.Raise()

				e.Frame(gtx.Ops)
			}
		}
	}()
}

func (g *GUI) ToastOK(title, msg string, callbacks ...func()) {
	if g.toastActive {
		return
	}

	go func() {
		g.toastActive = true
		defer func() { g.toastActive = false }()

		dx, dy := float32(400), float32(100)

		w := app.NewWindow(
			app.Title(title),
			app.Size(unit.Px(dx), unit.Px(dy)),
			app.MaxSize(unit.Px(dx), unit.Px(dy)),
			app.MinSize(unit.Px(dx), unit.Px(dy)),
		)

		m := material.Label(g.normal, g.normal.TextSize.Scale(15.0/16.0), msg)
		m.Color = rgba.N(rgba.White)
		m.Alignment = text.Middle

		okButton := &button.Button{
			Text:     "\t    OK",
			Released: rgba.N(rgba.Background),
			Pressed:  rgba.N(rgba.DarkGray),
		}
		okButton.Click = func() {
			okButton.Active = !okButton.Active

			w.Close()
		}

		var ops op.Ops

		for e := range w.Events() {
			if _, ok := e.(system.DestroyEvent); ok && len(callbacks) > 0 {
				for _, cb := range callbacks {
					cb()
				}
			}

			if e, ok := e.(system.FrameEvent); ok {
				gtx := layout.NewContext(&ops, e)

				ops.Reset()

				ColorBox(gtx, gtx.Constraints.Max, color.NRGBA{R: 25, G: 25, B: 25, A: 255})

				layout.Inset{
					Top: unit.Px(15),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return m.Layout(gtx)
					},
				)

				layout.Inset{
					Left: unit.Px(float32(gtx.Constraints.Max.X/3 + 15)),
					Top:  unit.Px(float32(gtx.Constraints.Max.Y / 2)),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return okButton.Layout(gtx)
					},
				)

				w.Center()
				w.Raise()

				e.Frame(gtx.Ops)
			}
		}
	}()
}

func (g *GUI) ToastYesNo(title, msg string, y, n func()) {
	if g.toastActive {
		return
	}

	go func() {
		g.toastActive = true
		defer func() { g.toastActive = false }()

		destroyed := true // Avoid calling n() multiple times.

		w := app.NewWindow(
			app.Title(title),
			app.Size(
				unit.Px(400),
				unit.Px(100),
			),
		)

		m := material.Label(g.normal, g.normal.TextSize.Scale(16.0/16.0), msg)
		m.Color = rgba.N(rgba.White)
		m.Alignment = text.Middle

		yButton := &button.Button{
			Text:     "\t   Yes",
			Released: rgba.N(rgba.Background),
			Pressed:  rgba.N(rgba.DarkGray),
		}
		yButton.Click = func() {
			destroyed = false
			y()

			w.Close()
		}

		nButton := &button.Button{
			Text:     "\t    No",
			Released: rgba.N(rgba.Background),
			Pressed:  rgba.N(rgba.DarkGray),
		}
		nButton.Click = func() {
			destroyed = false
			n()

			w.Close()
		}

		var ops op.Ops

		for e := range w.Events() {
			switch e := e.(type) {
			case system.DestroyEvent:
				if destroyed {
					n()
				}
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)

				ops.Reset()

				ColorBox(gtx, gtx.Constraints.Max, color.NRGBA{R: 25, G: 25, B: 25, A: 255})

				layout.Inset{
					Top: unit.Px(15),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return m.Layout(gtx)
					},
				)

				layout.Inset{
					Left: unit.Px(float32(gtx.Constraints.Max.X/2 - 115)),
					Top:  unit.Px(float32(gtx.Constraints.Max.Y/2) + 5),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return yButton.Layout(gtx)
					},
				)

				layout.Inset{
					Left: unit.Px(float32(gtx.Constraints.Max.X/2 + 15)),
					Top:  unit.Px(float32(gtx.Constraints.Max.Y/2) + 5),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return nButton.Layout(gtx)
					},
				)

				w.Center()
				w.Raise()

				e.Frame(gtx.Ops)
			}
		}
	}()
}

func (g *GUI) ToastError(err error) {
	if err == g.lastToastError && time.Since(g.lastToastTime) < time.Second*10 {
		return
	}

	g.lastToastError = err
	g.lastToastTime = time.Now()

	e := err.Error()
	es := strings.Split(e, " ")
	es[0] = strings.Title(es[0])

	g.ToastOK("Error", strings.Join(es, " "))
}

func (g *GUI) ToastErrorForce(err error) {
	g.lastToastError = err
	g.lastToastTime = time.Now()

	e := err.Error()
	es := strings.Split(e, " ")
	es[0] = strings.Title(es[0])

	g.toastActive = false

	g.ToastOK("Error", strings.Join(es, " "))
}

func (g *GUI) matchBalls(a *area.Area) {
	defer func() {
		r := recover()
		if r != nil {
			log.Error().Err(r.(error)).Msg("match balls failed")
		}
	}()

	if !g.Preview {
		a.NRGBA = rgba.N(rgba.Alpha(rgba.Black, 0x99))
		return
	}

	a.NRGBA = rgba.N(rgba.Alpha(rgba.Red, 0x99))

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		log.Err(err).Send()
		return
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		log.Err(err).Send()
		return
	}
	defer matrix.Close()

	result, _, score := match.Energy(matrix, g.Image)
	switch result {
	case match.Found, match.Duplicate:
		a.NRGBA = rgba.N(rgba.Alpha(rgba.Green, 0x99))
		a.Text = fmt.Sprintf("\t%d", score)
	case match.NotFound:
		a.NRGBA = rgba.N(rgba.Alpha(rgba.Red, 0x99))
		a.Text = "\tBalls"
	case match.Missed:
		a.NRGBA = rgba.N(rgba.Alpha(rgba.DarkerYellow, 0x99))
		a.Text = fmt.Sprintf("\t%d?", score)
	case match.Invalid:
		a.NRGBA = rgba.N(rgba.Alpha(rgba.Red, 0x99))
		a.Text = "\tBalls"
	}

	m, result := match.SelfScore(matrix, img)
	switch result {
	case match.Found:
		if state.EventType(m.Template.Value) == state.PreScore {
			a.NRGBA = rgba.N(rgba.Alpha(rgba.Green, 0x99))
			a.Text = "\tScoring"
		} else {
			a.NRGBA = rgba.N(rgba.Alpha(rgba.Green, 0x99))
			a.Text = "\tScored"
		}
	case match.Invalid:
		a.NRGBA = rgba.N(rgba.Alpha(rgba.Red, 0x99))
		a.Text = "\tInvalid Balls"
	}
}

func (g *GUI) matchScore(a *area.Area) {
	defer func() {
		r := recover()
		if r != nil {
			log.Error().Err(r.(error)).Msg("match score failed")
		}
	}()

	if !g.Preview {
		a.NRGBA = rgba.N(rgba.Alpha(rgba.Black, 0x99))
		return
	}

	// a.NRGBA = rgba.N(rgba.Alpha(rgba.Red, 0x99))
	// a.Subtext = ""

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		log.Err(err).Send()
		return
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		log.Err(err).Send()
		return
	}
	defer matrix.Close()

	for _, templates := range config.Current.Templates["scored"] {
		_, result, score := match.Matches(matrix, g.Image, templates)
		switch result {
		case match.Found, match.Duplicate:
			a.NRGBA = rgba.N(rgba.Alpha(rgba.Green, 0x99))
			a.Subtext = fmt.Sprintf("(+%d)", score)
			return
		case match.NotFound:
			a.NRGBA = rgba.N(rgba.Alpha(rgba.Red, 0x99))
		case match.Missed:
			a.NRGBA = rgba.N(rgba.Alpha(rgba.DarkerYellow, 0x99))
			a.Subtext = fmt.Sprintf("(%d?)", score)
		case match.Invalid:
			a.NRGBA = rgba.N(rgba.Alpha(rgba.Red, 0x99))
		}

		a.Subtext = strings.Title(result.String())
	}
}

func (g *GUI) matchMap(a *area.Area) {
	defer func() {
		r := recover()
		if r != nil {
			log.Error().Err(r.(error)).Msg("match map failed")
		}
	}()

	if !g.Preview {
		a.NRGBA = rgba.N(rgba.Alpha(rgba.Black, 0x99))
		return
	}

	a.NRGBA = rgba.N(rgba.Alpha(rgba.Red, 0x99))
	a.Subtext = ""

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		log.Err(err).Send()
		return
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		log.Err(err).Send()
		return
	}
	defer matrix.Close()

	_, ok := match.MiniMap(matrix, img)
	if ok {
		a.NRGBA = rgba.N(rgba.Alpha(rgba.Green, 0x99))
		a.Subtext = "(Found)"
	}
}

func (g *GUI) matchTime(a *area.Area) {
	defer func() {
		r := recover()
		if r != nil {
			log.Error().Err(r.(error)).Msg("match time failed")
		}
	}()

	if !g.Preview {
		a.NRGBA = rgba.N(rgba.Alpha(rgba.Black, 0x99))
		return
	}

	a.NRGBA = rgba.N(rgba.Alpha(rgba.Red, 0x99))
	a.Subtext = "(00:00)"

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		log.Err(err).Send()
		return
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		log.Err(err).Send()
		return
	}
	defer matrix.Close()

	s, k := match.Time(matrix, img)
	if s != 0 {
		a.NRGBA = rgba.N(rgba.Alpha(rgba.Green, 0x99))
		a.Subtext = "(" + k + ")"
	}
}

func (g *GUI) while(fn func(), wait *bool) {
	for {
		time.Sleep(time.Second)

		if *wait {
			continue
		}

		fn()
	}
}
