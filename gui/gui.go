package gui

import (
	"fmt"
	"image"
	"image/color"
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
	"gioui.org/widget/material"
	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"

	"github.com/pidgy/screenshot"
	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/gui/visual/area"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/help"
	"github.com/pidgy/unitehud/gui/visual/screen"
	"github.com/pidgy/unitehud/gui/visual/spinner"
	"github.com/pidgy/unitehud/gui/visual/split"
	"github.com/pidgy/unitehud/gui/visual/textblock"
	"github.com/pidgy/unitehud/match"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/stats"
	"github.com/pidgy/unitehud/team"
)

type GUI struct {
	*app.Window
	*screen.Screen

	WindowSize image.Rectangle

	Preview bool
	open    bool

	resize  bool
	resized bool

	Actions chan Action

	Running bool

	cpu, ram, uptime string
	time             time.Time
}

type Action string

const (
	Start   = Action("start")
	Stats   = Action("stats")
	Stop    = Action("stop")
	Record  = Action("record")
	Open    = Action("open")
	Closing = Action("closing")
)

var Window *GUI

const title = "Pokemon Unite HUD Server"

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
	}
}
func (g *GUI) main() (next string, err error) {
	// g.Window.Raise()

	split := &split.Vertical{
		Ratio: .70,
	}

	th := material.NewTheme(gofont.Collection())

	header := material.H5(th, title)
	header.Color = color.NRGBA(rgba.White)
	header.Alignment = text.Middle

	configButton := &button.Button{
		Text:     " Configure",
		Released: rgba.Background,
		Pressed:  rgba.DarkGray,
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
		Released: rgba.Background,
		Pressed:  rgba.DarkGray,
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
		configButton.Released = rgba.DarkerGray

		stopButton.Active = false
		stopButton.Disabled = false
		stopButton.Released = rgba.Background

		startButton.Active = false
		startButton.Disabled = true
		startButton.Released = rgba.DarkerGray

		recordButton.Active = false
		recordButton.Disabled = false
		recordButton.Released = rgba.Background

		g.Actions <- Start
		g.Running = true
	}

	stopButton.Click = func() {
		configButton.Active = false
		configButton.Disabled = false
		configButton.Released = rgba.Background

		stopButton.Active = false
		stopButton.Disabled = true
		stopButton.Released = rgba.DarkerGray

		startButton.Active = false
		startButton.Disabled = false
		startButton.Released = rgba.Background

		recordButton.Active = false
		recordButton.Disabled = true
		recordButton.Released = rgba.DarkerGray

		g.Actions <- Stop
		g.Running = false
	}

	recording := func() {
		if config.Current.Record {
			recordButton.Text = " Recording"
			recordButton.Released = rgba.DarkRed
		} else {
			recordButton.Text = "\tRecord"
			recordButton.Released = rgba.Background
			if recordButton.Disabled {
				recordButton.Released = rgba.DarkerGray
			}
		}
	}

	recordButton.Click = func() {
		recordButton.Active = !recordButton.Active
		g.Actions <- Record
	}

	openButton := &button.Button{
		Text:     "\t  Open",
		Released: rgba.Background,
		Pressed:  rgba.DarkGray,
	}

	openButton.Click = func() {
		openButton.Active = !openButton.Active

		g.Actions <- Open
	}

	textblock := &textblock.TextBlock{}

	statsButton := &button.Button{
		Text:           "ƒ",
		Released:       rgba.CoolBlue,
		Pressed:        rgba.DarkGray,
		Size:           image.Pt(15, 15),
		TextSize:       unit.Sp(10),
		TextOffsetTop:  -4,
		TextOffsetLeft: -5,
		BorderWidth:    unit.Sp(.5),
	}

	statsButton.Click = func() {
		stats.Data()
		statsButton.Active = !statsButton.Active
	}

	clearStatsButton := &button.Button{
		Text:           "×",
		Released:       rgba.DarkRed,
		Pressed:        rgba.DarkGray,
		Size:           image.Pt(15, 15),
		TextSize:       unit.Sp(12),
		TextOffsetTop:  -4,
		TextOffsetLeft: -7,
		BorderWidth:    unit.Sp(.5),
	}

	clearStatsButton.Click = func() {
		stats.Clear()
		clearStatsButton.Active = !clearStatsButton.Active
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

			background := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
			paint.ColorOp{Color: rgba.Background}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			background.Pop()

			split.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return Fill(
						gtx,
						color.NRGBA{R: 25, G: 25, B: 25, A: 255},
						func(gtx layout.Context) layout.Dimensions {
							layout.Inset{
								Left: unit.Px(2),
								Top:  unit.Px(10),
							}.Layout(gtx, header.Layout)

							cpu := material.H5(th, g.cpu)
							cpu.Color = color.NRGBA(rgba.White)
							cpu.Alignment = text.Middle
							cpu.TextSize = unit.Sp(12)

							layout.Inset{
								Top:  unit.Px(2),
								Left: unit.Px(float32(gtx.Constraints.Max.X - 175)),
							}.Layout(gtx, cpu.Layout)

							ram := material.H5(th, g.ram)
							ram.Color = color.NRGBA(rgba.White)
							ram.Alignment = text.Middle
							ram.TextSize = unit.Sp(12)

							layout.Inset{
								Top:  unit.Px(18),
								Left: unit.Px(float32(gtx.Constraints.Max.X - 175)),
							}.Layout(gtx, ram.Layout)

							uptime := material.H5(th, g.uptime)
							uptime.Color = color.NRGBA(rgba.White)
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
							holding.Color = color.NRGBA(team.Self.RGBA)
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
							symbol.Color = color.NRGBA(rgba.SlateGray)

							acronym := material.H5(th, "STP")
							acronym.Alignment = text.Middle
							acronym.TextSize = unit.Sp(11)
							acronym.Color = color.NRGBA(rgba.SlateGray)

							down := float32(1)
							left := 1

							if config.Current.Record {
								symbol.Text = spinRecord.Next()
								symbol.Color = color.NRGBA(rgba.Red)
								symbol.TextSize = unit.Sp(20)
								acronym.Font.Weight = text.ExtraBold

								acronym.Text = "REC"
								acronym.Color = color.NRGBA(rgba.Red)

								down = 0
							} else if g.Running {
								symbol.Text = spinRun.Next()
								symbol.Color = color.NRGBA(rgba.Green)

								acronym.Text = "RUN"
								acronym.Color = color.NRGBA(rgba.Green)

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

							p, o, s := server.Scores()

							purple := material.H5(th, strconv.Itoa(p))
							purple.Color = color.NRGBA(team.Purple.RGBA)
							purple.Alignment = text.Middle
							purple.TextSize = unit.Sp(13)

							layout.Inset{
								Top:  unit.Px(2),
								Left: unit.Px(float32(gtx.Constraints.Max.X - 35)),
							}.Layout(gtx, purple.Layout)

							orange := material.H5(th, strconv.Itoa(o))
							orange.Color = color.NRGBA(team.Orange.RGBA)
							orange.Alignment = text.Middle
							orange.TextSize = unit.Sp(13)

							layout.Inset{
								Top:  unit.Px(19),
								Left: unit.Px(float32(gtx.Constraints.Max.X - 35)),
							}.Layout(gtx, orange.Layout)

							self := material.H5(th, strconv.Itoa(s))
							self.Color = color.NRGBA(team.Self.RGBA)
							self.Alignment = text.Middle
							self.TextSize = unit.Sp(13)

							layout.Inset{
								Top:  unit.Px(35),
								Left: unit.Px(float32(gtx.Constraints.Max.X - 35)),
							}.Layout(gtx, self.Layout)

							clock := material.H5(th, server.Clock())
							clock.Color = color.NRGBA(rgba.White)
							clock.Alignment = text.Middle
							clock.TextSize = unit.Sp(13)

							layout.Inset{
								Top:  unit.Px(2),
								Left: unit.Px(float32(gtx.Constraints.Max.X - 90)),
							}.Layout(gtx, clock.Layout)

							layout.Inset{Top: unit.Px(50)}.Layout(
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
							recording()

							layout.Inset{
								Left: unit.Px(float32(gtx.Constraints.Max.X - statsButton.Size.X)),
								Top:  unit.Px(float32(gtx.Constraints.Min.Y)),
							}.Layout(
								gtx,
								func(gtx layout.Context) layout.Dimensions {
									return statsButton.Layout(gtx)
								})

							layout.Inset{
								Left: unit.Px(float32(gtx.Constraints.Max.X - clearStatsButton.Size.X)),
								Top:  unit.Px(float32(gtx.Constraints.Min.Y + statsButton.Size.Y)),
							}.Layout(
								gtx,
								func(gtx layout.Context) layout.Dimensions {
									return clearStatsButton.Layout(gtx)
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

							if notify.PurpleScore != nil {
								layout.Inset{
									Top:  unit.Px(52),
									Left: unit.Px(5),
								}.Layout(gtx, (&screen.Screen{
									Border:      true,
									BorderColor: color.NRGBA(team.Purple.RGBA),
									Image:       notify.PurpleScore,
								}).Layout)
							}
							if notify.OrangeScore != nil {
								layout.Inset{
									Top:  unit.Px(112),
									Left: unit.Px(5),
								}.Layout(gtx, (&screen.Screen{
									Border:      true,
									BorderColor: color.NRGBA(team.Orange.RGBA),
									Image:       notify.OrangeScore,
								}).Layout)
							}
							if notify.Balls != nil {
								layout.Inset{
									Top:  unit.Px(172),
									Left: unit.Px(5),
								}.Layout(gtx, (&screen.Screen{
									Border:      true,
									BorderColor: color.NRGBA(team.Self.RGBA),
									Image:       notify.Balls,
									ScaleX:      2,
									ScaleY:      2,
								}).Layout)
							}
							if notify.SelfScore != nil {
								layout.Inset{
									Top:  unit.Px(232),
									Left: unit.Px(5),
								}.Layout(gtx, (&screen.Screen{
									Border:      true,
									BorderColor: color.NRGBA(team.Self.RGBA),
									Image:       notify.SelfScore,
									ScaleX:      4,
									ScaleY:      4,
								}).Layout)
							}
							if notify.Time != nil {
								layout.Inset{
									Top:  unit.Px(292),
									Left: unit.Px(5),
								}.Layout(gtx, (&screen.Screen{
									Image:       notify.Time,
									Border:      true,
									BorderColor: color.NRGBA(team.Time.RGBA),
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

	x := unit.Px((float32(g.Screen.Bounds().Max.X) / 2) + 10)
	y := unit.Px((float32(g.Screen.Bounds().Max.Y) / 2) + 89)

	if g.open && g.resize {
		g.Window.Option(app.Size(x, y))

		if !g.resized {
			g.resize = false
			g.resized = true
		}
	}
}

func (g *GUI) preview() {
	for {
		if g.Preview {
			img, err := screenshot.CaptureScreen()
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
		Ratio: .75,
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
			Pressed:  rgba.Background,
			Released: rgba.DarkGray,
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
	ballsAreaScaleText.Color = color.NRGBA(rgba.White)
	ballsAreaScaleText.Alignment = text.Middle
	ballsAreaScaleText.TextSize = unit.Sp(11)

	ballsAreaScaleValueText := material.H5(th, "1x")
	ballsAreaScaleValueText.Color = color.NRGBA(rgba.White)
	ballsAreaScaleValueText.Alignment = text.Middle
	ballsAreaScaleValueText.TextSize = unit.Sp(11)

	ballsAreaScaleUpButton := &button.Button{
		Text:     "+",
		Pressed:  rgba.Background,
		Released: rgba.DarkGray,
		Size:     image.Pt(30, 30),
		TextSize: unit.Sp(12),
	}

	ballsAreaScaleDownButton := &button.Button{
		Text:     "-",
		Pressed:  rgba.Background,
		Released: rgba.DarkGray,
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
			Pressed:  rgba.Background,
			Released: rgba.DarkGray,
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
	timeAreaScaleText.Color = color.NRGBA(rgba.White)
	timeAreaScaleText.Alignment = text.Middle
	timeAreaScaleText.TextSize = unit.Sp(11)

	timeAreaScaleValueText := material.H5(th, "1x")
	timeAreaScaleValueText.Color = color.NRGBA(rgba.White)
	timeAreaScaleValueText.Alignment = text.Middle
	timeAreaScaleValueText.TextSize = unit.Sp(11)

	timeAreaScaleUpButton := &button.Button{
		Text:     "+",
		Pressed:  color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released: rgba.Background,
		Size:     image.Pt(30, 30),
		TextSize: unit.Sp(12),
	}

	timeAreaScaleDownButton := &button.Button{
		Text:     "-",
		Pressed:  color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released: rgba.Background,
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
			Pressed:  rgba.Background,
			Released: rgba.DarkGray,
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
	scoreAreaScaleText.Color = color.NRGBA(rgba.White)
	scoreAreaScaleText.Alignment = text.Middle
	scoreAreaScaleText.TextSize = unit.Sp(11)

	scoreAreaScaleValueText := material.H5(th, "1x")
	scoreAreaScaleValueText.Color = color.NRGBA(rgba.White)
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

	/*mapArea := &area.Area{
		Text:     "\t  Map",
		TextSize: unit.Sp(13),
		Min:      config.Current.Map.Min.Div(2),
		Max:      config.Current.Map.Max.Div(2),
		Button: &button.Button{
			Active:   true,
			Text:     "\t  tMap",
			Pressed:  rgba.Background,
			Released: rgba.DarkGray,
			Size:     image.Pt(100, 30),
		},
	}
	*/
	saveButton := &button.Button{
		Text:     "\t  Save",
		Pressed:  rgba.Background,
		Released: rgba.DarkGray,
		Active:   true,
	}

	cancelButton := &button.Button{
		Text:     "\tCancel",
		Pressed:  rgba.Background,
		Released: rgba.DarkGray,
		Active:   true,
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
		// config.Current.Map = mapArea.Rectangle()

		err := config.Current.Save()
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		next = "main"
		notify.Feed(rgba.White, "Configuration saved to "+config.File)
	}

	screenButton := &button.Button{
		Text:     "\tPreview",
		Pressed:  rgba.Background,
		Released: rgba.DarkGray,
		Active:   true,
	}

	screenButton.Click = func() {
		g.Preview = !g.Preview
	}

	resetButton := &button.Button{
		Text:     "\t Reset",
		Pressed:  rgba.Background,
		Released: rgba.DarkGray,
		Active:   true,
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

	helpButton := &button.Button{
		Text:     "\t  Help",
		Pressed:  rgba.Background,
		Released: rgba.DarkGray,
		Active:   true,
	}

	helpButton.Click = func() {
		helpButton.Active = !helpButton.Active
		next = "help_configure"
	}

	button16x9 := &button.Button{
		Text:          "16:9",
		TextSize:      unit.Sp(12),
		TextOffsetTop: -3,
		BorderWidth:   unit.Px(1),
		Pressed:       rgba.Background,
		Released:      rgba.DarkGray,
		Active:        true,
		Size:          image.Pt(45, 18),
	}

	button4x3 := &button.Button{
		Text:          " 4:3",
		TextSize:      unit.Sp(12),
		TextOffsetTop: -3,
		BorderWidth:   unit.Px(1),
		Pressed:       rgba.Background,
		Released:      rgba.DarkGray,
		Active:        true,
		Size:          image.Pt(45, 18),
	}

	button16x9.Click = func() {
		config.Current.Scales.To16x9()
		ballsAreaScaleScaleButtons(ballsArea, ballsAreaScaleUpButton, ballsAreaScaleDownButton)
		timeAreaScaleScaleButtons(timeArea, timeAreaScaleUpButton, timeAreaScaleDownButton)
		scoreAreaScaleScaleButtons(scoreArea, scoreAreaScaleUpButton, scoreAreaScaleDownButton)

		button16x9.Active = !button16x9.Active
	}

	button4x3.Click = func() {
		config.Current.Scales.To4x3()
		ballsAreaScaleScaleButtons(ballsArea, ballsAreaScaleUpButton, ballsAreaScaleDownButton)
		timeAreaScaleScaleButtons(timeArea, timeAreaScaleUpButton, timeAreaScaleDownButton)
		scoreAreaScaleScaleButtons(scoreArea, scoreAreaScaleUpButton, scoreAreaScaleDownButton)

		button4x3.Active = !button4x3.Active
	}

	header := material.H5(th, "Pokemon Unite HUD Server")
	header.Color = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	header.Alignment = text.Middle

	kill := false
	defer func() { kill = true }()
	go g.run(func() { g.matchScore(scoreArea) }, &kill)
	go g.run(func() { g.matchTime(timeArea) }, &kill)
	go g.run(func() { g.matchBalls(ballsArea) }, &kill)
	// go g.run(func() { g.matchMap(mapArea) }, &kill)

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
			paint.ColorOp{Color: rgba.Background}.Add(gtx.Ops)
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
									layout.Inset{
										Left:  unit.Px(float32(gtx.Constraints.Max.X - 110)),
										Right: unit.Px(10),
										Top:   unit.Px(30),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return saveButton.Layout(gtx)
										})

									layout.Inset{
										Left:  unit.Px(float32(gtx.Constraints.Max.X - 220)),
										Right: unit.Px(10),
										Top:   unit.Px(30),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return cancelButton.Layout(gtx)
										})

									layout.Inset{
										Left:  unit.Px(float32(gtx.Constraints.Max.X - 330)),
										Right: unit.Px(10),
										Top:   unit.Px(30),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return screenButton.Layout(gtx)
										})

									layout.Inset{
										Left:  unit.Px(float32(gtx.Constraints.Max.X - 440)),
										Right: unit.Px(10),
										Top:   unit.Px(30),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return resetButton.Layout(gtx)
										})

									layout.Inset{
										Left:  unit.Px(float32(gtx.Constraints.Max.X - 550)),
										Right: unit.Px(10),
										Top:   unit.Px(30),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return helpButton.Layout(gtx)
										})

									layout.Inset{
										Left:  unit.Px(float32(gtx.Constraints.Max.X - 607)),
										Right: unit.Px(10),
										Top:   unit.Px(3),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return button16x9.Layout(gtx)
										})

									layout.Inset{
										Left:  unit.Px(float32(gtx.Constraints.Max.X - 607)),
										Right: unit.Px(10),
										Top:   unit.Px(25),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return button4x3.Layout(gtx)
										})

									// Time Area Rectangle Buttons
									layout.Inset{
										Left: unit.Px(240),
										Top:  unit.Px(3),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return timeArea.Button.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(240),
										Top:  unit.Px(35),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return timeAreaScaleDownButton.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(275),
										Top:  unit.Px(35),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return timeAreaScaleText.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(275),
										Top:  unit.Px(50),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											timeAreaScaleValueText.Text = fmt.Sprintf("%.2fx", config.Current.Scales.Time)
											return timeAreaScaleValueText.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(310),
										Top:  unit.Px(35),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return timeAreaScaleUpButton.Layout(gtx)
										})

									// Points Area Rectangle Buttons
									layout.Inset{
										Left: unit.Px(125),
										Top:  unit.Px(3),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return ballsArea.Button.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(125),
										Top:  unit.Px(35),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return ballsAreaScaleDownButton.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(160),
										Top:  unit.Px(35),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return ballsAreaScaleText.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(160),
										Top:  unit.Px(50),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											ballsAreaScaleValueText.Text = fmt.Sprintf("%.2fx", config.Current.Scales.Balls)
											return ballsAreaScaleValueText.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(195),
										Top:  unit.Px(35),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return ballsAreaScaleUpButton.Layout(gtx)
										})

									// Score Area Rectangle Buttons
									layout.Inset{
										Left: unit.Px(10),
										Top:  unit.Px(3),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return scoreArea.Button.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(10),
										Top:  unit.Px(35),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return scoreAreaScaleDownButton.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(45),
										Top:  unit.Px(35),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return scoreAreaScaleText.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(45),
										Top:  unit.Px(50),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											scoreAreaScaleValueText.Text = fmt.Sprintf("%.2fx", config.Current.Scales.Score)
											return scoreAreaScaleValueText.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(80),
										Top:  unit.Px(35),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return scoreAreaScaleUpButton.Layout(gtx)
										})

									return layout.Dimensions{Size: gtx.Constraints.Max}
								},
							)
						})
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

	th := material.NewTheme(gofont.Collection())

	header := material.H5(th, "Help: Configuration")
	header.Color = color.NRGBA(rgba.White)
	header.Alignment = text.Middle

	backwardButton := &button.Button{
		Text:     " <",
		Released: rgba.SlateGray,
		Pressed:  rgba.DarkGray,
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
		Released: rgba.SlateGray,
		Pressed:  rgba.DarkGray,
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
		Released: rgba.SlateGray,
		Pressed:  rgba.DarkGray,
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
			paint.ColorOp{Color: rgba.Background}.Add(gtx.Ops)
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
							pages.Color = color.NRGBA(rgba.White)
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
	scaleDownButton.Released = rgba.Background
	if config.Current.Scales.Time < 0.01 {
		config.Current.Scales.Time = 0.05
		scaleDownButton.Released = rgba.DarkerGray
		scaleDownButton.Disabled = true
	}
	scaleUpButton.Disabled = false
	scaleUpButton.Released = rgba.Background
	if config.Current.Scales.Time > 0.99 {
		config.Current.Scales.Time = 1.0
		scaleUpButton.Released = rgba.DarkerGray
		scaleUpButton.Disabled = true
	}
}

func ballsAreaScaleScaleButtons(a *area.Area, scaleUpButton, scaleDownButton *button.Button) {
	scaleDownButton.Disabled = false
	scaleDownButton.Released = rgba.Background
	if config.Current.Scales.Balls < 0.01 {
		config.Current.Scales.Balls = 0.05
		scaleDownButton.Released = rgba.DarkerGray
		scaleDownButton.Disabled = true
	}
	scaleUpButton.Disabled = false
	scaleUpButton.Released = rgba.Background
	if config.Current.Scales.Balls > 0.99 {
		config.Current.Scales.Balls = 1.0
		scaleUpButton.Released = rgba.DarkerGray
		scaleUpButton.Disabled = true
	}
}

func scoreAreaScaleScaleButtons(a *area.Area, scaleUpButton, scaleDownButton *button.Button) {
	scaleDownButton.Disabled = false
	scaleDownButton.Released = rgba.Background
	if config.Current.Scales.Score < 0.01 {
		config.Current.Scales.Score = 0.05
		scaleDownButton.Released = rgba.DarkerGray
		scaleDownButton.Disabled = true
	}
	scaleUpButton.Disabled = false
	scaleUpButton.Released = rgba.Background
	if config.Current.Scales.Score > 0.99 {
		config.Current.Scales.Score = 1.0
		scaleUpButton.Released = rgba.DarkerGray
		scaleUpButton.Disabled = true
	}
}

func (g *GUI) matchBalls(a *area.Area) {
	if !g.Preview {
		a.NRGBA = color.NRGBA(rgba.Alpha(rgba.Black, 0x99))
		return
	}

	a.NRGBA = color.NRGBA(rgba.Alpha(rgba.Red, 0x99))

	img, err := screenshot.CaptureRect(a.Rectangle())
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
		a.NRGBA = color.NRGBA(rgba.Alpha(rgba.Green, 0x99))
		a.Text = fmt.Sprintf("\t%d", score)
		return
	case match.NotFound:
		a.NRGBA = color.NRGBA(rgba.Alpha(rgba.Red, 0x99))
		a.Text = "\tBalls"
	case match.Missed:
		a.NRGBA = color.NRGBA(rgba.Alpha(rgba.DarkerYellow, 0x99))
		a.Text = fmt.Sprintf("\t%d?", score)
	case match.Invalid:
		a.NRGBA = color.NRGBA(rgba.Alpha(rgba.Red, 0x99))
		a.Text = "\tBalls"
	}
}

func (g *GUI) matchScore(a *area.Area) {
	if !g.Preview {
		a.NRGBA = color.NRGBA(rgba.Alpha(rgba.Black, 0x99))
		return
	}

	// a.NRGBA = color.NRGBA(rgba.Alpha(rgba.Red, 0x99))
	// a.Subtext = ""

	img, err := screenshot.CaptureRect(a.Rectangle())
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	m := match.Match{}
	for _, templates := range config.Current.Templates["scored"] {
		result, score := m.Matches(matrix, g.Image, templates)
		switch result {
		case match.Found, match.Duplicate:
			a.NRGBA = color.NRGBA(rgba.Alpha(rgba.Green, 0x99))
			a.Subtext = fmt.Sprintf("(+%d)", score)
			return
		case match.NotFound:
			a.NRGBA = color.NRGBA(rgba.Alpha(rgba.Red, 0x99))
		case match.Missed:
			a.NRGBA = color.NRGBA(rgba.Alpha(rgba.DarkerYellow, 0x99))
			a.Subtext = fmt.Sprintf("(%d?)", score)
		case match.Invalid:
			a.NRGBA = color.NRGBA(rgba.Alpha(rgba.Red, 0x99))
		}

		a.Subtext = result.String()
	}
}

func (g *GUI) matchMap(a *area.Area) {
	if !g.Preview {
		a.NRGBA = color.NRGBA(rgba.Alpha(rgba.Black, 0x99))
		return
	}

	a.NRGBA = color.NRGBA(rgba.Alpha(rgba.Red, 0x99))
	a.Subtext = ""

	img, err := screenshot.CaptureRect(a.Rectangle())
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	_, ok := match.MiniMap(matrix, img)
	if ok {
		a.NRGBA = color.NRGBA(rgba.Alpha(rgba.Green, 0x99))
		a.Subtext = "(Found)"
	}
}

func (g *GUI) matchTime(a *area.Area) {
	if !g.Preview {
		a.NRGBA = color.NRGBA(rgba.Alpha(rgba.Black, 0x99))
		return
	}

	a.NRGBA = color.NRGBA(rgba.Alpha(rgba.Red, 0x99))
	a.Subtext = "(00:00)"

	img, err := screenshot.CaptureRect(a.Rectangle())
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	m := match.Match{}

	s, k := m.Time(matrix, img)
	if s != 0 {
		a.NRGBA = color.NRGBA(rgba.Alpha(rgba.Green, 0x99))
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
