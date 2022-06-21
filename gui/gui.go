package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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
	"github.com/pidgy/unitehud/stats"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/window"
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

	cpu, ram, uptime, clients string
	time                      time.Time
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
)

var Window *GUI

const title = "Pokemon UNITE HUD Server"

func New() {
	Window = &GUI{
		Window: app.NewWindow(
			app.Title(title),
		),
		Preview: true,
		Actions: make(chan Action, 1024),
		resize:  true,
	}
}

func (g *GUI) Open() {
	time.AfterFunc(time.Second, func() {
		g.open = true
	})

	go g.preview()

	go func() {
		defer func() {
			g.Actions <- Closing
		}()

		var err error

		next := "main"
		for next != "" {
			switch next {
			case "main":
				g.resize = true

				next, err = g.main()
				if err != nil {
					log.Error().Err(err).Send()
				}
			case "configure":
				g.resize = true

				next, err = g.configure()
				if err != nil {
					log.Error().Err(err).Send()
				}
			case "help_configure":
				h := help.Configuration()

				next, err = g.configurationHelpDialog(h.Help, h.Layout)
				if err != nil {
					log.Error().Err(err).Send()
				}
			default:
				return
			}
		}
	}()

	go g.proc()

	app.Main()
}

func (g *GUI) proc() {
	handle, err := syscall.GetCurrentProcess()
	if err != nil {
		notify.Feed(color.RGBA{G: 0xFF, B: 0xFF, A: 0xFF}, "Failed to monitor usage: (%s)", err.Error())
		return
	}

	var ctime, etime, ktime, utime syscall.Filetime
	err = syscall.GetProcessTimes(handle, &ctime, &etime, &ktime, &utime)
	if err != nil {
		notify.Feed(color.RGBA{G: 0xFF, B: 0xFF, A: 0xFF}, "Failed to monitor usage: (%s)", err.Error())
		return
	}

	prev := ctime.Nanoseconds()
	usage := ktime.Nanoseconds() + utime.Nanoseconds() // Always overflows

	g.time = time.Now()

	cpus := float64(runtime.NumCPU()) - 2

	for range time.NewTicker(time.Second).C {
		err := syscall.GetProcessTimes(handle, &ctime, &etime, &ktime, &utime)
		if err != nil {
			notify.Feed(color.RGBA{G: 0xFF, B: 0xFF, A: 0xFF}, "Failed to monitor usage: (%s)", err.Error())
			continue
		}

		now := time.Now().UnixNano()
		diff := now - prev
		current := ktime.Nanoseconds() + utime.Nanoseconds()
		diff2 := current - usage
		prev = now
		usage = current

		g.cpu = fmt.Sprintf("CPU: %.1f%s", (100*float64(diff2)/float64(diff))/cpus, "%")

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		g.ram = fmt.Sprintf("RAM: %d%s", (m.Sys / 1024 / 1024), "MB")

		run := time.Time{}.Add(time.Since(g.time))
		g.uptime = fmt.Sprintf("RUN: %02d:%02d", run.Minute(), run.Second())

		clients := server.Clients()
		g.clients = fmt.Sprintf("OBS: %01d", clients)
	}
}

func (g *GUI) main() (next string, err error) {
	// g.Window.Raise()

	split := &split.Vertical{
		Ratio: .70,
	}

	th := material.NewTheme(gofont.Collection())

	header := material.H5(th, title)
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

	startButton := &button.Button{
		Text:     "\t  Start",
		Released: rgba.N(rgba.Background),
		Pressed:  rgba.N(rgba.DarkGray),
	}

	stopButton := &button.Button{
		Text:     "\t  Stop",
		Disabled: true,
	}

	recordButton := &button.Button{
		Text:     "\tRecord",
		Disabled: true,
	}

	startButton.Click = func() {
		g.Preview = false

		configButton.Active = true
		configButton.Disabled = true
		configButton.Released = rgba.N(rgba.DarkerGray)

		stopButton.Active = false
		stopButton.Disabled = false
		stopButton.Released = rgba.N(rgba.Background)

		startButton.Active = false
		startButton.Disabled = true
		startButton.Released = rgba.N(rgba.DarkerGray)

		recordButton.Active = false
		recordButton.Disabled = false
		recordButton.Released = rgba.N(rgba.Background)

		g.Actions <- Start
		g.Running = true
	}

	stopButton.Click = func() {
		configButton.Active = false
		configButton.Disabled = false
		configButton.Released = rgba.N(rgba.Background)

		stopButton.Active = false
		stopButton.Disabled = true
		stopButton.Released = rgba.N(rgba.DarkerGray)

		startButton.Active = false
		startButton.Disabled = false
		startButton.Released = rgba.N(rgba.Background)

		recordButton.Active = false
		recordButton.Disabled = true
		recordButton.Released = rgba.N(rgba.DarkerGray)

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
				recordButton.Released = rgba.N(rgba.DarkerGray)
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

	textblock := &textblock.TextBlock{}

	statsButton := &button.Button{
		Text:           "¼",
		Released:       rgba.N(rgba.CoolBlue),
		Pressed:        rgba.N(rgba.DarkGray),
		Size:           image.Pt(15, 20),
		TextSize:       unit.Sp(12),
		TextOffsetTop:  -3,
		TextOffsetLeft: -8,
		BorderWidth:    unit.Sp(.5),
	}

	statsButton.Click = func() {
		stats.Data()
		statsButton.Active = !statsButton.Active
	}

	historyButton := &button.Button{
		Text:           "±",
		Released:       rgba.N(rgba.Seafoam),
		Pressed:        rgba.N(rgba.DarkSeafoam),
		Size:           image.Pt(15, 20),
		TextSize:       unit.Sp(14),
		TextOffsetTop:  -4,
		TextOffsetLeft: -7,
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
		TextOffsetTop:  -4,
		TextOffsetLeft: -5,
		BorderWidth:    unit.Sp(.5),
	}

	obsButton.Click = func() {
		obsButton.Active = !obsButton.Active

		ex, err := os.Executable()
		if err != nil {
			notify.Feed(rgba.Red, "Failed to open www/ directory: %v", err)
			return
		}

		dir := filepath.Dir(ex)
		err = open.Run(dir + "/www")
		if err != nil {
			notify.Feed(rgba.Red, "Failed to open www/ directory: %v", err)
			return
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

		e := <-g.Events()
		switch e := e.(type) {
		case app.ConfigEvent:
		case system.DestroyEvent:
			return "", e.Err
		case system.FrameEvent:
			g.Window.Option(app.Title(title))

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

								windowHeader := material.Caption(th, config.Current.Window)
								windowHeader.Color = rgba.N(rgba.SlateGray)
								windowHeader.Alignment = text.Middle
								windowHeader.Font.Weight = text.Bold
								windowHeaderOffset := float32(0)
								if config.Current.LostWindow != "" {
									windowHeader2 := material.Caption(th, config.Current.LostWindow)
									windowHeader2.Color = rgba.N(rgba.PaleRed)
									windowHeader2.Alignment = text.Middle
									windowHeader2.Font.Weight = text.Bold
									windowHeaderOffset = float32(layout.Inset{
										Left: unit.Px(2),
										Top:  unit.Px(45),
									}.Layout(gtx, windowHeader2.Layout).Size.X) + 3

									windowHeader.Text = fmt.Sprintf("» %s", config.Current.Window)
								}

								layout.Inset{
									Left: unit.Px(2 + windowHeaderOffset),
									Top:  unit.Px(45),
								}.Layout(gtx, windowHeader.Layout)
							}
							{
								cpu := material.H5(th, g.cpu)
								cpu.Color = rgba.N(rgba.White)
								cpu.Alignment = text.Middle
								cpu.TextSize = unit.Sp(12)

								layout.Inset{
									Top:  unit.Px(2),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 175)),
								}.Layout(gtx, cpu.Layout)

								ram := material.H5(th, g.ram)
								ram.Color = rgba.N(rgba.White)
								ram.Alignment = text.Middle
								ram.TextSize = unit.Sp(12)

								layout.Inset{
									Top:  unit.Px(18),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 175)),
								}.Layout(gtx, ram.Layout)

								uptime := material.H5(th, g.uptime)
								uptime.Color = rgba.N(rgba.White)
								uptime.Alignment = text.Middle
								uptime.TextSize = unit.Sp(12)

								layout.Inset{
									Top:  unit.Px(34),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 175)),
								}.Layout(gtx, uptime.Layout)

								h := fmt.Sprintf("%d/%2d", team.Balls.Holding, team.Balls.HoldingMax)
								if team.Balls.Holding < 10 {
									h = "0" + h
								}
								holding := material.H5(th, h)
								holding.Color = rgba.N(team.Self.RGBA)
								holding.Alignment = text.Middle
								holding.TextSize = unit.Sp(13)

								layout.Inset{
									Top:  unit.Px(18),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 90)),
								}.Layout(gtx, holding.Layout)

								symbol := material.H5(th, spinStop.Next())
								symbol.Alignment = text.Middle
								symbol.TextSize = unit.Sp(14)
								symbol.Font.Weight = text.ExtraBold
								symbol.Color = rgba.N(rgba.SlateGray)

								acronym := material.H5(th, "STP")
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
									Top:  unit.Px(31 + down),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 90 - left)),
								}.Layout(gtx, symbol.Layout)

								layout.Inset{
									Top:  unit.Px(35),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 80)),
								}.Layout(gtx, acronym.Layout)
							}
							{
								p, o, s := server.Scores()

								purple := material.H5(th, strconv.Itoa(p))
								purple.Color = rgba.N(team.Purple.RGBA)
								purple.Alignment = text.Middle
								purple.TextSize = unit.Sp(13)

								layout.Inset{
									Top:  unit.Px(2),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 35)),
								}.Layout(gtx, purple.Layout)

								orange := material.H5(th, strconv.Itoa(o))
								orange.Color = rgba.N(team.Orange.RGBA)
								orange.Alignment = text.Middle
								orange.TextSize = unit.Sp(13)

								layout.Inset{
									Top:  unit.Px(18),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 35)),
								}.Layout(gtx, orange.Layout)

								self := material.H5(th, strconv.Itoa(s))
								self.Color = rgba.N(team.Self.RGBA)
								self.Alignment = text.Middle
								self.TextSize = unit.Sp(13)

								layout.Inset{
									Top:  unit.Px(34),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 35)),
								}.Layout(gtx, self.Layout)

								clock := material.H5(th, server.Clock())
								clock.Color = rgba.N(rgba.White)
								clock.Alignment = text.Middle
								clock.TextSize = unit.Sp(13)
								layout.Inset{
									Top:  unit.Px(2),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 90)),
								}.Layout(gtx, clock.Layout)
							}
							{
								connectedClients := material.H5(th, g.clients)
								connectedClients.Color = rgba.N(rgba.Seafoam)
								connectedClients.Alignment = text.Middle
								connectedClients.TextSize = unit.Sp(12)

								if server.Clients() == 0 {
									connectedClients.Color = rgba.N(rgba.PaleRed)
								}

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 175)),
									Top:  unit.Px(50),
								}.Layout(gtx, connectedClients.Layout)
							}

							layout.Inset{
								Top: unit.Px(65),
							}.Layout(
								gtx,
								func(gtx layout.Context) layout.Dimensions {
									return textblock.Layout(gtx, notify.Feeds())
								})

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
									Left: unit.Px(float32(gtx.Constraints.Max.X - statsButton.Size.X)),
									Top:  unit.Px(float32(gtx.Constraints.Min.Y)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return statsButton.Layout(gtx)
									})

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - obsButton.Size.X)),
									Top:  unit.Px(float32(gtx.Constraints.Min.Y + statsButton.Size.Y)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return obsButton.Layout(gtx)
									})

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - (historyButton.Size.X * 2))),
									Top:  unit.Px(float32(gtx.Constraints.Min.Y)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return historyButton.Layout(gtx)
									})
							}
							// Right-side buttons.
							{
								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
									Top:  unit.Px(float32(gtx.Constraints.Max.Y - 335)),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									preview.SetImage(notify.Preview)

									return preview.Layout(th, gtx)
								})

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
									Top:  unit.Px(float32(gtx.Constraints.Max.Y - 265)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return startButton.Layout(gtx)
									})

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
									Top:  unit.Px(float32(gtx.Constraints.Max.Y - 210)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return stopButton.Layout(gtx)
									})

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
									Top:  unit.Px(float32(gtx.Constraints.Max.Y - 155)),
								}.Layout(gtx,
									func(gtx layout.Context) layout.Dimensions {
										return configButton.Layout(gtx)
									})

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
									Top:  unit.Px(float32(gtx.Constraints.Max.Y - 100)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return recordButton.Layout(gtx)
									})

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
									Top:  unit.Px(float32(gtx.Constraints.Max.Y - 45)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return openButton.Layout(gtx)
									})
							}
							// Event images.
							{
								layout.Inset{
									Top:  unit.Px(52),
									Left: unit.Px(5),
								}.Layout(gtx, (&screen.Screen{
									Border:      true,
									BorderColor: rgba.N(team.Purple.RGBA),
									Image:       notify.PurpleScore,
								}).Layout)

								layout.Inset{
									Top:  unit.Px(112),
									Left: unit.Px(5),
								}.Layout(gtx, (&screen.Screen{
									Border:      true,
									BorderColor: rgba.N(team.Orange.RGBA),
									Image:       notify.OrangeScore,
								}).Layout)

								layout.Inset{
									Top:  unit.Px(172),
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
									Top:  unit.Px(292),
									Left: unit.Px(5),
								}.Layout(gtx, (&screen.Screen{
									Image:       notify.Time,
									Border:      true,
									BorderColor: rgba.N(team.Time.RGBA),
									ScaleX:      2,
									ScaleY:      2,
								}).Layout)
							}

							return layout.Dimensions{Size: gtx.Constraints.Max}
						})
				})

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

	x := float32(math.Max(float64(g.Screen.Image.Bounds().Max.X/2), 960))
	y := float32(math.Max(float64((g.Screen.Image.Bounds().Max.Y/2)+175), 715))

	if g.open && g.resize {
		g.Window.Option(
			app.Size(
				unit.Px(x),
				unit.Px(y),
			),
		)

		// Prevent capturing once the window has been resized.
		if !g.resized {
			g.resize = false
			g.resized = true
			g.Preview = false
		}
	}
}

func (g *GUI) preview() {
	for {
		if g.Preview {
			img, err := window.Capture()
			if err != nil {
				log.Fatal().Err(err).Send()
			}
			g.display(img)
		}

		// Redraw the image.
		g.Invalidate()

		time.Sleep(time.Second / 2)
	}
}

func (g *GUI) configure() (next string, err error) {
	g.Preview = true
	defer func() {
		g.Preview = false
	}()

	split := &split.Horizontal{
		Ratio: .6,
	}

	th := material.NewTheme(gofont.Collection())

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

	ballsAreaScaleText := material.H5(th, "Scale")
	ballsAreaScaleText.Color = rgba.N(rgba.White)
	ballsAreaScaleText.Alignment = text.Middle
	ballsAreaScaleText.TextSize = unit.Sp(11)

	ballsAreaScaleValueText := material.H5(th, "1x")
	ballsAreaScaleValueText.Color = rgba.N(rgba.White)
	ballsAreaScaleValueText.Alignment = text.Middle
	ballsAreaScaleValueText.TextSize = unit.Sp(11)

	ballsAreaScaleUpButton := &button.Button{
		Text:     "+",
		Pressed:  rgba.N(rgba.Background),
		Released: rgba.N(rgba.DarkGray),
		Size:     image.Pt(30, 30),
		TextSize: unit.Sp(12),
	}

	ballsAreaScaleDownButton := &button.Button{
		Text:     "-",
		Pressed:  rgba.N(rgba.Background),
		Released: rgba.N(rgba.DarkGray),
		Size:     image.Pt(30, 30),
		TextSize: unit.Sp(12),
	}

	ballsAreaScaleUpButton.Click = func() {
		g.Preview = false
		defer g.buttonSpam(ballsAreaScaleUpButton)

		ballsAreaScaleUpButton.Active = !ballsAreaScaleUpButton.Active
		config.Current.Scales.Balls += .01

		ballsAreaScaleScaleButtons(ballsArea, ballsAreaScaleUpButton, ballsAreaScaleDownButton)
	}

	ballsAreaScaleDownButton.Click = func() {
		g.Preview = false
		defer g.buttonSpam(ballsAreaScaleDownButton)

		ballsAreaScaleDownButton.Active = !ballsAreaScaleDownButton.Active
		config.Current.Scales.Balls -= .01

		ballsAreaScaleScaleButtons(ballsArea, ballsAreaScaleUpButton, ballsAreaScaleDownButton)
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

	timeAreaScaleText := material.H5(th, "Scale")
	timeAreaScaleText.Color = rgba.N(rgba.White)
	timeAreaScaleText.Alignment = text.Middle
	timeAreaScaleText.TextSize = unit.Sp(11)

	timeAreaScaleValueText := material.H5(th, "1x")
	timeAreaScaleValueText.Color = rgba.N(rgba.White)
	timeAreaScaleValueText.Alignment = text.Middle
	timeAreaScaleValueText.TextSize = unit.Sp(11)

	timeAreaScaleUpButton := &button.Button{
		Text:     "+",
		Pressed:  color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released: rgba.N(rgba.Background),
		Size:     image.Pt(30, 30),
		TextSize: unit.Sp(12),
	}

	timeAreaScaleDownButton := &button.Button{
		Text:     "-",
		Pressed:  color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released: rgba.N(rgba.Background),
		Size:     image.Pt(30, 30),
		TextSize: unit.Sp(12),
	}

	timeAreaScaleUpButton.Click = func() {
		g.Preview = false
		defer g.buttonSpam(timeAreaScaleUpButton)

		timeAreaScaleUpButton.Active = !timeAreaScaleUpButton.Active
		config.Current.Scales.Time += .01

		timeAreaScaleScaleButtons(timeArea, timeAreaScaleUpButton, timeAreaScaleDownButton)
	}

	timeAreaScaleDownButton.Click = func() {
		g.Preview = false
		defer g.buttonSpam(timeAreaScaleDownButton)

		timeAreaScaleDownButton.Active = !timeAreaScaleDownButton.Active
		config.Current.Scales.Time -= .01

		timeAreaScaleScaleButtons(timeArea, timeAreaScaleUpButton, timeAreaScaleDownButton)
	}

	scoreArea := &area.Area{
		Text: "Score",
		Min:  config.Current.Scores.Min.Div(2),
		Max:  config.Current.Scores.Max.Div(2),

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

	scoreAreaScaleText := material.H5(th, "Scale")
	scoreAreaScaleText.Color = rgba.N(rgba.White)
	scoreAreaScaleText.Alignment = text.Middle
	scoreAreaScaleText.TextSize = unit.Sp(11)

	scoreAreaScaleValueText := material.H5(th, "1x")
	scoreAreaScaleValueText.Color = rgba.N(rgba.White)
	scoreAreaScaleValueText.Alignment = text.Middle
	scoreAreaScaleValueText.TextSize = unit.Sp(11)

	scoreAreaScaleUpButton := &button.Button{
		Text:     "+",
		Pressed:  color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released: color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:     image.Pt(30, 30),
		TextSize: unit.Sp(12),
	}

	scoreAreaScaleDownButton := &button.Button{
		Text:     "-",
		Pressed:  color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released: color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:     image.Pt(30, 30),
		TextSize: unit.Sp(12),
	}

	scoreAreaScaleUpButton.Click = func() {
		g.Preview = false
		defer g.buttonSpam(scoreAreaScaleUpButton)

		scoreAreaScaleUpButton.Active = !scoreAreaScaleUpButton.Active
		config.Current.Scales.Score += .01

		scoreAreaScaleScaleButtons(scoreArea, scoreAreaScaleUpButton, scoreAreaScaleDownButton)
	}

	scoreAreaScaleDownButton.Click = func() {
		g.Preview = false
		defer g.buttonSpam(scoreAreaScaleDownButton)

		scoreAreaScaleDownButton.Active = !scoreAreaScaleDownButton.Active
		config.Current.Scales.Score -= .01

		scoreAreaScaleScaleButtons(scoreArea, scoreAreaScaleUpButton, scoreAreaScaleDownButton)
	}

	ballsAreaScaleScaleButtons(ballsArea, ballsAreaScaleUpButton, ballsAreaScaleDownButton)
	timeAreaScaleScaleButtons(timeArea, timeAreaScaleUpButton, timeAreaScaleDownButton)
	scoreAreaScaleScaleButtons(scoreArea, scoreAreaScaleUpButton, scoreAreaScaleDownButton)

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

	cancelButton.Click = func() {
		defer server.Clear()

		cancelButton.Disabled = true
		saveButton.Disabled = true
		ballsArea.Button.Disabled = true
		timeArea.Button.Disabled = true
		scoreArea.Button.Disabled = true

		next = "main"
		notify.Feed(rgba.White, "Configuration omitted")

		g.Actions <- Refresh
	}

	saveButton.Click = func() {
		defer server.Clear()

		cancelButton.Disabled = true
		saveButton.Disabled = true
		ballsArea.Button.Disabled = true
		timeArea.Button.Disabled = true
		scoreArea.Button.Disabled = true

		saveButton.Active = false

		config.Current.Scores = scoreArea.Rectangle()
		config.Current.Time = timeArea.Rectangle()
		config.Current.Balls = ballsArea.Rectangle()
		config.Current.Map = mapArea.Rectangle()

		err := config.Current.Save()
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		next = "main"
		notify.Feed(rgba.White, "Configuration saved to "+config.File)

		g.Actions <- Refresh
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

	resetButton := &button.Button{
		Text:     "\t Reset",
		Pressed:  rgba.N(rgba.Background),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
	}

	resetButton.Click = func() {
		err := config.Reset()
		if err != nil {
			log.Error().Err(err).Msg("failed to reset config")
			notify.Feed(rgba.White, "failed to reset configuration (%s)", err.Error())
		}

		config.Current.Reload()

		ballsArea.Min, ballsArea.Max = config.Current.Balls.Min.Div(2), config.Current.Balls.Max.Div(2)
		timeArea.Min, timeArea.Max = config.Current.Time.Min.Div(2), config.Current.Time.Max.Div(2)
		scoreArea.Min, scoreArea.Max = config.Current.Scores.Min.Div(2), config.Current.Scores.Max.Div(2)

		resetButton.Active = !resetButton.Active
	}

	windowList := &dropdown.List{
		Items: []*dropdown.Item{},
		Callback: func(i *dropdown.Item) {
			if i.Text == "" {
				config.Current.Window = config.MainDisplay
				return
			}

			config.Current.Window = i.Text
		},
	}

	for _, win := range window.Open {
		windowList.Items = append(windowList.Items, &dropdown.Item{Text: win})
	}

	for _, item := range windowList.Items {
		if item.Text == config.Current.Window {
			item.Checked.Value = true
		}
	}

	header := material.H5(th, "Pokemon Unite HUD Server")
	header.Color = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	header.Alignment = text.Middle

	kill := false
	defer func() { kill = true }()
	go g.run(func() { g.matchScore(scoreArea) }, &kill)
	go g.run(func() { g.matchTime(timeArea) }, &kill)
	go g.run(func() { g.matchBalls(ballsArea) }, &kill)
	go g.run(func() { g.matchMap(mapArea) }, &kill)

	var ops op.Ops

	for next == "" {
		if !g.open {
			time.Sleep(time.Millisecond * 10)
			continue
		}

		e := <-g.Events()
		switch e := e.(type) {
		case app.ConfigEvent:
		case system.DestroyEvent:
			return "", nil
		case system.FrameEvent:
			g.Window.Option(app.Title(fmt.Sprintf("%s (%s %s)", title, g.cpu, g.ram)))

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
						})
				},

				func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(5)).Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							return Fill(gtx, color.NRGBA{R: 25, G: 25, B: 25, A: 255},
								func(gtx layout.Context) layout.Dimensions {
									{
										layout.Inset{
											Left:  unit.Px(10),
											Top:   unit.Px(100),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return saveButton.Layout(gtx)
											})

										layout.Inset{
											Left:  unit.Px(115),
											Top:   unit.Px(100),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return cancelButton.Layout(gtx)
											})

										layout.Inset{
											Left:  unit.Px(220),
											Top:   unit.Px(100),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return resetButton.Layout(gtx)
											})

										layout.Inset{
											Left:  unit.Px(325),
											Top:   unit.Px(100),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return screenButton.Layout(gtx)
											})
									}

									// Capture window.
									{
										layout.Inset{
											Left:  unit.Px((float32(gtx.Constraints.Max.X) / 3) * 2),
											Top:   unit.Px(3),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												windowListTitle := material.Label(th, unit.Px(14), "Capture Window")
												windowListTitle.Color = rgba.N(rgba.SlateGray)
												return windowListTitle.Layout(gtx)
											})

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
													return windowList.Layout(gtx, th)
												})
											})
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
											})

										layout.Inset{
											Top:  unit.Px(38),
											Left: unit.Px(220),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return timeAreaScaleDownButton.Layout(gtx)
											})

										layout.Inset{
											Left: unit.Px(255),
											Top:  unit.Px(38),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return timeAreaScaleText.Layout(gtx)
											})

										layout.Inset{
											Left: unit.Px(255),
											Top:  unit.Px(53),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												timeAreaScaleValueText.Text = fmt.Sprintf("%.2fx", config.Current.Scales.Time)
												return timeAreaScaleValueText.Layout(gtx)
											})

										layout.Inset{
											Left: unit.Px(290),
											Top:  unit.Px(38),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return timeAreaScaleUpButton.Layout(gtx)
											})
									}

									// Points area rectangle buttons.
									{
										layout.Inset{
											Left: unit.Px(115),
											Top:  unit.Px(5),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return ballsArea.Button.Layout(gtx)
											})

										layout.Inset{
											Left: unit.Px(115),
											Top:  unit.Px(38),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return ballsAreaScaleDownButton.Layout(gtx)
											})

										layout.Inset{
											Left: unit.Px(150),
											Top:  unit.Px(38),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return ballsAreaScaleText.Layout(gtx)
											})

										layout.Inset{
											Left: unit.Px(150),
											Top:  unit.Px(55),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												ballsAreaScaleValueText.Text = fmt.Sprintf("%.2fx", config.Current.Scales.Balls)
												return ballsAreaScaleValueText.Layout(gtx)
											})

										layout.Inset{
											Left: unit.Px(185),
											Top:  unit.Px(38),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return ballsAreaScaleUpButton.Layout(gtx)
											})
									}

									// Score area rectangle buttons.
									{
										layout.Inset{
											Left: unit.Px(10),
											Top:  unit.Px(5),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return scoreArea.Button.Layout(gtx)
											})

										layout.Inset{
											Left: unit.Px(10),
											Top:  unit.Px(38),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return scoreAreaScaleDownButton.Layout(gtx)
											})

										layout.Inset{
											Left: unit.Px(45),
											Top:  unit.Px(38),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return scoreAreaScaleText.Layout(gtx)
											})

										layout.Inset{
											Left: unit.Px(45),
											Top:  unit.Px(55),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												scoreAreaScaleValueText.Text = fmt.Sprintf("%.2fx", config.Current.Scales.Score)
												return scoreAreaScaleValueText.Layout(gtx)
											})

										layout.Inset{
											Left: unit.Px(80),
											Top:  unit.Px(38),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return scoreAreaScaleUpButton.Layout(gtx)
											})
									}

									return layout.Dimensions{Size: gtx.Constraints.Max}
								},
							)
						})
				},
			)

			scoreArea.Layout(gtx)
			ballsArea.Layout(gtx)
			timeArea.Layout(gtx)
			mapArea.Layout(gtx)

			e.Frame(gtx.Ops)
		}
	}

	return next, nil
}

func (g *GUI) configurationHelpDialog(h *help.Help, widget layout.Widget) (next string, err error) {
	split := &split.Vertical{Ratio: .70}

	var ops op.Ops

	th := material.NewTheme(gofont.Collection())

	header := material.H5(th, "Help: Configuration")
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
								})
						},
					)
				},

				func(gtx layout.Context) layout.Dimensions {
					return Fill(
						gtx,
						color.NRGBA{R: 25, G: 25, B: 25, A: 255},
						func(gtx layout.Context) layout.Dimensions {
							pages := material.H5(th, fmt.Sprintf("%d / %d", h.Page+1, h.Pages))
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
								})

							layout.Inset{
								Left: unit.Px(float32(gtx.Constraints.Max.X - 65)),
								Top:  unit.Px(float32(gtx.Constraints.Max.Y - 100)),
							}.Layout(gtx,
								func(gtx layout.Context) layout.Dimensions {
									return forwardButton.Layout(gtx)
								})

							layout.Inset{
								Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
								Top:  unit.Px(float32(gtx.Constraints.Max.Y - 45)),
							}.Layout(gtx,
								func(gtx layout.Context) layout.Dimensions {
									return returnButton.Layout(gtx)
								})

							return layout.Dimensions{Size: gtx.Constraints.Max}
						})
				})

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
	})
}

func timeAreaScaleScaleButtons(a *area.Area, scaleUpButton, scaleDownButton *button.Button) {
	scaleDownButton.Disabled = false
	scaleDownButton.Released = rgba.N(rgba.Background)
	if config.Current.Scales.Time < 0.01 {
		config.Current.Scales.Time = 0.05
		scaleDownButton.Released = rgba.N(rgba.DarkerGray)
		scaleDownButton.Disabled = true
	}
	scaleUpButton.Disabled = false
	scaleUpButton.Released = rgba.N(rgba.Background)
	if config.Current.Scales.Time > 0.99 {
		config.Current.Scales.Time = 1.0
		scaleUpButton.Released = rgba.N(rgba.DarkerGray)
		scaleUpButton.Disabled = true
	}
}

func ballsAreaScaleScaleButtons(a *area.Area, scaleUpButton, scaleDownButton *button.Button) {
	scaleDownButton.Disabled = false
	scaleDownButton.Released = rgba.N(rgba.Background)
	if config.Current.Scales.Balls < 0.01 {
		config.Current.Scales.Balls = 0.05
		scaleDownButton.Released = rgba.N(rgba.DarkerGray)
		scaleDownButton.Disabled = true
	}
	scaleUpButton.Disabled = false
	scaleUpButton.Released = rgba.N(rgba.Background)
	if config.Current.Scales.Balls > 0.99 {
		config.Current.Scales.Balls = 1.0
		scaleUpButton.Released = rgba.N(rgba.DarkerGray)
		scaleUpButton.Disabled = true
	}
}

func scoreAreaScaleScaleButtons(a *area.Area, scaleUpButton, scaleDownButton *button.Button) {
	scaleDownButton.Disabled = false
	scaleDownButton.Released = rgba.N(rgba.Background)
	if config.Current.Scales.Score < 0.01 {
		config.Current.Scales.Score = 0.05
		scaleDownButton.Released = rgba.N(rgba.DarkerGray)
		scaleDownButton.Disabled = true
	}
	scaleUpButton.Disabled = false
	scaleUpButton.Released = rgba.N(rgba.Background)
	if config.Current.Scales.Score > 0.99 {
		config.Current.Scales.Score = 1.0
		scaleUpButton.Released = rgba.N(rgba.DarkerGray)
		scaleUpButton.Disabled = true
	}
}

func (g *GUI) matchBalls(a *area.Area) {
	if !g.Preview {
		a.NRGBA = rgba.N(rgba.Alpha(rgba.Black, 0x99))
		return
	}

	a.NRGBA = rgba.N(rgba.Alpha(rgba.Red, 0x99))

	img, err := window.CaptureRect(a.Rectangle())
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	result, _, score := match.Balls(matrix, g.Image)
	switch result {
	case match.Found, match.Duplicate:
		a.NRGBA = rgba.N(rgba.Alpha(rgba.Green, 0x99))
		a.Text = fmt.Sprintf("\t%d", score)
		return
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
}

func (g *GUI) matchScore(a *area.Area) {
	if !g.Preview {
		a.NRGBA = rgba.N(rgba.Alpha(rgba.Black, 0x99))
		return
	}

	// a.NRGBA = rgba.N(rgba.Alpha(rgba.Red, 0x99))
	// a.Subtext = ""

	img, err := window.CaptureRect(a.Rectangle())
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

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

		a.Subtext = result.String()
	}
}

func (g *GUI) matchMap(a *area.Area) {
	if !g.Preview {
		a.NRGBA = rgba.N(rgba.Alpha(rgba.Black, 0x99))
		return
	}

	a.NRGBA = rgba.N(rgba.Alpha(rgba.Red, 0x99))
	a.Subtext = ""

	img, err := window.CaptureRect(a.Rectangle())
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	_, ok := match.MiniMap(matrix, img)
	if ok {
		a.NRGBA = rgba.N(rgba.Alpha(rgba.Green, 0x99))
		a.Subtext = "(Found)"
	}
}

func (g *GUI) matchTime(a *area.Area) {
	if !g.Preview {
		a.NRGBA = rgba.N(rgba.Alpha(rgba.Black, 0x99))
		return
	}

	a.NRGBA = rgba.N(rgba.Alpha(rgba.Red, 0x99))
	a.Subtext = "(00:00)"

	img, err := window.CaptureRect(a.Rectangle())
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	s, k := match.Time(matrix, img)
	if s != 0 {
		a.NRGBA = rgba.N(rgba.Alpha(rgba.Green, 0x99))
		a.Subtext = "(" + k + ")"
	}
}

func (g *GUI) run(fn func(), kill *bool) {
	for {
		if *kill {
			return
		}

		fn()

		time.Sleep(time.Second)
	}
}
