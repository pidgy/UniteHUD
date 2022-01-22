package gui

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"sync"
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
	"github.com/vova616/screenshot"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/match"
	"github.com/pidgy/unitehud/window/gui/visual/area"
	"github.com/pidgy/unitehud/window/gui/visual/button"
	"github.com/pidgy/unitehud/window/gui/visual/screen"
	"github.com/pidgy/unitehud/window/gui/visual/split"
	"github.com/pidgy/unitehud/window/gui/visual/textblock"
)

type GUI struct {
	*app.Window
	*screen.Screen

	logs []string

	Preview bool
	open    bool

	Actions chan Action

	once *sync.Once
}

type Action string

const (
	Start = Action("start")
	Stop  = Action("stop")
)

func New() *GUI {
	g := &GUI{
		Window: app.NewWindow(
			app.Title("Pokemon Unite HUD Server Configuration"),
		),
		Preview: true,
		once:    &sync.Once{},
	}

	return g
}

func (g *GUI) Log(txt string) {
	g.logs = append(g.logs, fmt.Sprintf("[%s] %s", time.Now().Format(time.Kitchen), txt))
	if len(g.logs) > 26 {
		g.logs = g.logs[len(g.logs)-27:]
	}
}

func (g *GUI) Open() {
	time.AfterFunc(time.Second, func() {
		g.open = true
	})

	go g.preview()

	go func() {
		defer os.Exit(0)

		next := "main"
		var err error

		for next != "" {
			switch next {
			case "main":
				next, err = g.main()
				if err != nil {
					log.Error().Err(err).Send()
				}

			case "configure":
				next, err = g.configure()
				if err != nil {
					log.Error().Err(err).Send()
				}
			default:
				println("retuyrn")
				return
			}
		}
	}()

	g.Actions = make(chan Action, 1024)

	app.Main()
}

func (g *GUI) Display(src image.Image) {
	g.Screen = &screen.Screen{
		Image: src,
	}

	y := unit.Px(float32(g.Bounds().Max.Y) / 2).Scale(1.15)
	x := unit.Px(float32(g.Bounds().Max.X) / 2).Scale(1.01)

	if g.open {
		g.Window.Option(app.Size(x, y))

		// Redraw the image.
		g.Invalidate()
	}
}

func (g *GUI) main() (string, error) {
	next := ""

	split := &split.Vertical{Ratio: .70}

	var ops op.Ops

	th := material.NewTheme(gofont.Collection())
	title := material.H5(th, "Pokemon Unite HUD Server")
	title.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	title.Alignment = text.Middle

	configButton := &button.Button{
		Text:     " Configure",
		Released: color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F},
		Pressed:  color.NRGBA{A: 0x4F},
	}

	configButton.Click = func() {
		configButton.Active = false
		next = "configure"
	}

	startButton := &button.Button{
		Text:     "\t  Start",
		Released: color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F},
		Pressed:  color.NRGBA{A: 0x4F},
	}

	stopButton := &button.Button{
		Text:     "\t  Stop",
		Released: color.NRGBA{A: 0xF},
		Pressed:  color.NRGBA{A: 0x4F},
		Disabled: true,
	}

	startButton.Click = func() {
		startButton.Active = false
		g.Preview = false

		configButton.Disabled = true
		configButton.Released = color.NRGBA{A: 0xF}

		stopButton.Disabled = false
		stopButton.Released = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F}

		startButton.Disabled = true
		startButton.Released = color.NRGBA{A: 0xF}

		g.Actions <- Start
	}

	stopButton.Click = func() {
		configButton.Disabled = false
		configButton.Released = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F}

		stopButton.Disabled = true
		stopButton.Released = color.NRGBA{A: 0xF}

		startButton.Disabled = false
		startButton.Released = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F}

		g.Actions <- Stop
	}

	textblock := &textblock.TextBlock{}

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
			paint.ColorOp{Color: color.NRGBA{R: 100, G: 100, B: 100, A: 255}}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			background.Pop()

			split.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return Fill(gtx,
						color.NRGBA{R: 25, G: 25, B: 25, A: 255},
						func(gtx layout.Context) layout.Dimensions {
							title.Layout(gtx)
							return layout.Inset{Top: unit.Px(50), Left: unit.Px(5)}.Layout(gtx,
								func(gtx layout.Context) layout.Dimensions {
									return textblock.Layout(gtx, g.logs)
								})
						},
					)
				},

				func(gtx layout.Context) layout.Dimensions {
					return Fill(
						gtx,
						color.NRGBA{R: 25, G: 25, B: 25, A: 255},
						func(gtx layout.Context) layout.Dimensions {
							layout.Inset{Left: unit.Px(float32(gtx.Constraints.Max.X - 125)), Top: unit.Px(float32(gtx.Constraints.Max.Y - 100))}.Layout(
								gtx,
								func(gtx layout.Context) layout.Dimensions {
									return startButton.Layout(gtx)
								})

							layout.Inset{Left: unit.Px(float32(gtx.Constraints.Max.X - 125)), Top: unit.Px(float32(gtx.Constraints.Max.Y - 45))}.Layout(
								gtx,
								func(gtx layout.Context) layout.Dimensions {
									return stopButton.Layout(gtx)
								})

							return layout.Inset{Left: unit.Px(float32(gtx.Constraints.Max.X - 125)), Top: unit.Px(5)}.Layout(gtx,
								func(gtx layout.Context) layout.Dimensions {
									return configButton.Layout(gtx)
								})
						})
				})

			e.Frame(gtx.Ops)
		}
	}

	return next, nil
}

func (g *GUI) preview() {
	for {
		if g.Preview {
			img, err := screenshot.CaptureScreen()
			if err != nil {
				log.Fatal().Err(err).Send()
			}

			g.Display(img)
		}

		time.Sleep(time.Second)
	}
}

func (g *GUI) configure() (string, error) {
	g.Preview = true
	defer func() { g.Preview = false }()
	next := ""

	split := &split.Horizontal{Ratio: .75}

	th := material.NewTheme(gofont.Collection())
	var ops op.Ops

	scoreArea := &area.Area{
		Text: "Score",
		Min:  config.Current.Scores.Min.Div(2),
		Max:  config.Current.Scores.Max.Div(2),

		Button: &button.Button{
			Active:   true,
			Text:     "\t Score",
			Pressed:  color.NRGBA{G: 0xFF, A: 0x3F},
			Released: color.NRGBA{G: 0xFF, A: 0x4F},
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

	pointsArea := &area.Area{
		Text: "Points",
		Min:  image.Pt(0, 0),
		Max:  image.Pt(100, 100),
		Button: &button.Button{
			Active:   true,
			Text:     "\t Points",
			Pressed:  color.NRGBA{G: 0xFF, A: 0x3F},
			Released: color.NRGBA{G: 0xFF, A: 0x4F},
		},
	}

	pointsArea.Button.Click = func() {
		if !pointsArea.Button.Active {
			pointsArea.Text = "Points (Locked)"
			pointsArea.Button.Text = "\tLocked"
			pointsArea.NRGBA.A = 0x9
			return
		}

		pointsArea.Text = "Points"
		pointsArea.Button.Text = "\t Points"
		pointsArea.NRGBA.A = 0x4F
	}

	timeArea := &area.Area{
		Text: "Time",
		Min:  config.Current.Time.Min.Div(2),
		Max:  config.Current.Time.Max.Div(2),
		Button: &button.Button{
			Active:   true,
			Text:     "\t  Time",
			Pressed:  color.NRGBA{G: 0xFF, A: 0x3F},
			Released: color.NRGBA{G: 0xFF, A: 0x4F},
		},
	}

	timeArea.Button.Click = func() {
		if !timeArea.Button.Active {
			timeArea.Text = "Time (Locked)"
			timeArea.Button.Text = "\tLocked"
			timeArea.NRGBA.A = 0x9
			return
		}

		timeArea.Text = "Time"
		timeArea.Button.Text = "\t  Time"
		timeArea.NRGBA.A = 0x4F
	}

	g.once.Do(func() {
		go g.run(func() { g.matchScore(scoreArea) })
		go g.run(func() { g.matchPoints(pointsArea) })
		go g.run(func() { g.matchTime(timeArea) })
	})

	scaleUpButton := &button.Button{
		Text:     "+1",
		Pressed:  color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released: color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:     image.Pt(40, 40),
	}

	scaleButtons := func() {}

	scaleUpButton.Click = func() {
		g.Preview = false

		scaleUpButton.Active = !scaleUpButton.Active
		config.Current.Scale += .05
		scaleButtons()
		config.Current.Reload()

		g.Preview = true
	}

	scaleText := material.H5(th, "Scale")
	scaleText.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	scaleText.Alignment = text.Middle
	scaleText.TextSize = unit.Px(15)

	scaleValueText := material.H5(th, "1x")
	scaleValueText.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	scaleValueText.Alignment = text.Middle
	scaleValueText.TextSize = unit.Px(14)

	scaleDownButton := &button.Button{
		Text:     "-1",
		Pressed:  color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released: color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:     image.Pt(40, 40),
	}

	scaleDownButton.Click = func() {
		g.Preview = false

		scaleDownButton.Active = !scaleDownButton.Active
		config.Current.Scale -= .05
		scaleButtons()

		config.Current.Reload()

		g.Preview = true
	}

	scaleButtons = func() {
		scaleDownButton.Disabled = false
		scaleDownButton.Released = color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F}
		if config.Current.Scale < 0.05 {
			config.Current.Scale = 0.05
			scaleDownButton.Released = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F}
			scaleDownButton.Disabled = true
		}
		scaleUpButton.Disabled = false
		scaleUpButton.Released = color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F}
		if config.Current.Scale > 1.0 {
			config.Current.Scale = 1.0
			scaleUpButton.Released = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F}
			scaleUpButton.Disabled = true
		}
	}

	saveButton := &button.Button{
		Text:     "\t  Save",
		Pressed:  color.NRGBA{A: 0x4F},
		Released: color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F},
	}

	cancelButton := &button.Button{
		Text:     "\tCancel",
		Pressed:  color.NRGBA{A: 0x4F},
		Released: color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F},
	}

	cancelButton.Click = func() {
		cancelButton.Disabled = true
		saveButton.Disabled = true
		pointsArea.Button.Disabled = true
		timeArea.Button.Disabled = true
		scoreArea.Button.Disabled = true

		next = "main"
		g.Log("Configuration omitted")
	}

	saveButton.Click = func() {
		cancelButton.Disabled = true
		saveButton.Disabled = true
		pointsArea.Button.Disabled = true
		timeArea.Button.Disabled = true
		scoreArea.Button.Disabled = true

		saveButton.Active = false

		config.Current.Scores = scoreArea.Rectangle()
		config.Current.Time = timeArea.Rectangle()

		err := config.Current.Save()
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		next = "main"
		g.Log("Configuration saved to unitehud.config")
	}

	screenButton := &button.Button{
		Text:     "\tPreview",
		Pressed:  color.NRGBA{R: 0xF, G: 0xFF, B: 0xF, A: 0x3F},
		Released: color.NRGBA{R: 0xFF, G: 0xF, B: 0xF, A: 0x3F},
		Active:   true,
	}

	screenButton.Click = func() {
		g.Preview = !g.Preview
	}

	title := material.H5(th, "Pokemon Unite HUD Server")
	title.Color = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	title.Alignment = text.Middle

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
			gtx := layout.NewContext(&ops, e)
			pointer.CursorNameOp{Name: pointer.CursorGrab}.Add(gtx.Ops)

			background := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
			paint.ColorOp{Color: color.NRGBA{R: 100, G: 100, B: 100, A: 255}}.Add(gtx.Ops)
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
										Top:   unit.Px(10),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return saveButton.Layout(gtx)
										})

									layout.Inset{
										Left:  unit.Px(float32(gtx.Constraints.Max.X - 235)),
										Right: unit.Px(10),
										Top:   unit.Px(10),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return cancelButton.Layout(gtx)
										})

									layout.Inset{
										Left:  unit.Px(float32(gtx.Constraints.Max.X - 360)),
										Right: unit.Px(10),
										Top:   unit.Px(10),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return screenButton.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(450),
										Top:  unit.Px(10),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return scaleUpButton.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(405),
										Top:  unit.Px(10),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return scaleText.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(405),
										Top:  unit.Px(32),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											scaleValueText.Text = fmt.Sprintf("%.2fx", config.Current.Scale)
											return scaleValueText.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(355),
										Top:  unit.Px(10),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return scaleDownButton.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(240),
										Top:  unit.Px(10),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return timeArea.Button.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(125),
										Top:  unit.Px(10),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return pointsArea.Button.Layout(gtx)
										})

									return layout.Inset{
										Left: unit.Px(10),
										Top:  unit.Px(10),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return scoreArea.Button.Layout(gtx)
										})

								},
							)
						})
				},
			)

			scoreArea.Layout(gtx)
			pointsArea.Layout(gtx)
			timeArea.Layout(gtx)

			e.Frame(gtx.Ops)
		}
	}

	return next, nil
}

func (g *GUI) matchPoints(a *area.Area) {
	a.NRGBA = color.NRGBA{R: 0xFF, A: 0x3F}
}

func (g *GUI) matchScore(a *area.Area) {
	a.NRGBA = color.NRGBA{R: 0xFF, A: 0x3F}
	a.Subtext = "(Not Detected)"

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
		if m.Matches(matrix, g.Image, templates) {
			a.NRGBA = color.NRGBA{G: 0xFF, A: 0x3F}
			a.Subtext = "(Detected)"
			return
		}
	}
}

func (g *GUI) matchTime(a *area.Area) {
	a.NRGBA = color.NRGBA{R: 0xFF, A: 0x3F}
	a.Subtext = "(Not Detected)"

	img, err := screenshot.CaptureRect(a.Rectangle())
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	m := match.Match{}

	if m.Time(matrix, img, config.Current.RegularTime) != 0 {
		a.NRGBA = color.NRGBA{G: 0xFF, A: 0x3F}
		a.Subtext = "(Regular Detected)"
	}

	if m.Time(matrix, img, config.Current.FinalStretch) != 0 {
		a.NRGBA = color.NRGBA{G: 0xFF, A: 0x3F}
		a.Subtext = "(Final Stretch Detected)"
	}
}

func (g *GUI) run(fn func()) {
	for {
		if g.Preview {
			fn()
		}

		time.Sleep(time.Second)
	}
}
