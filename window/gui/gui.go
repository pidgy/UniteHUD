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

func (g *GUI) Log(format string, a ...interface{}) {
	txt := fmt.Sprintf(format, a...)

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
							return layout.Inset{Top: unit.Px(50)}.Layout(gtx,
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

	pointsArea := &area.Area{
		Text: "Points",
		Min:  image.Pt(0, 0),
		Max:  image.Pt(100, 100),
		Button: &button.Button{
			Active:   true,
			Text:     "\t Points",
			Pressed:  color.NRGBA{G: 0xFF, A: 0x3F},
			Released: color.NRGBA{G: 0xFF, A: 0x4F},
			Size:     image.Pt(100, 30),
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

	pointsAreaScaleText := material.H5(th, "Scale")
	pointsAreaScaleText.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	pointsAreaScaleText.Alignment = text.Middle
	pointsAreaScaleText.TextSize = unit.Sp(11)

	pointsAreaScaleValueText := material.H5(th, "1x")
	pointsAreaScaleValueText.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	pointsAreaScaleValueText.Alignment = text.Middle
	pointsAreaScaleValueText.TextSize = unit.Sp(11)

	pointsAreaScaleUpButton := &button.Button{
		Text:     "+",
		Pressed:  color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released: color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:     image.Pt(30, 30),
		TextSize: unit.Sp(12),
	}

	pointsAreaScaleDownButton := &button.Button{
		Text:     "-",
		Pressed:  color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released: color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:     image.Pt(30, 30),
		TextSize: unit.Sp(12),
	}

	pointsAreaScaleUpButton.Click = func() {
		g.Preview = false
		defer g.buttonSpam(pointsAreaScaleUpButton)

		pointsAreaScaleUpButton.Active = !pointsAreaScaleUpButton.Active
		config.Current.Scales.Points += .01

		pointsAreaScaleScaleButtons(pointsAreaScaleUpButton, pointsAreaScaleDownButton)
		config.Current.Reload()
	}

	pointsAreaScaleDownButton.Click = func() {
		g.Preview = false
		defer g.buttonSpam(pointsAreaScaleDownButton)

		pointsAreaScaleDownButton.Active = !pointsAreaScaleDownButton.Active
		config.Current.Scales.Points -= .01

		pointsAreaScaleScaleButtons(pointsAreaScaleUpButton, pointsAreaScaleDownButton)
		config.Current.Reload()
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

		timeArea.Text = "Time"
		timeArea.Button.Text = "\t  Time"
		timeArea.NRGBA.A = 0x4F
	}

	timeAreaScaleText := material.H5(th, "Scale")
	timeAreaScaleText.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	timeAreaScaleText.Alignment = text.Middle
	timeAreaScaleText.TextSize = unit.Sp(11)

	timeAreaScaleValueText := material.H5(th, "1x")
	timeAreaScaleValueText.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	timeAreaScaleValueText.Alignment = text.Middle
	timeAreaScaleValueText.TextSize = unit.Sp(11)

	timeAreaScaleUpButton := &button.Button{
		Text:     "+",
		Pressed:  color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released: color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:     image.Pt(30, 30),
		TextSize: unit.Sp(12),
	}

	timeAreaScaleDownButton := &button.Button{
		Text:     "-",
		Pressed:  color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released: color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:     image.Pt(30, 30),
		TextSize: unit.Sp(12),
	}

	timeAreaScaleUpButton.Click = func() {
		g.Preview = false
		defer g.buttonSpam(timeAreaScaleUpButton)

		timeAreaScaleUpButton.Active = !timeAreaScaleUpButton.Active
		config.Current.Scales.Time += .01

		timeAreaScaleScaleButtons(timeAreaScaleUpButton, timeAreaScaleDownButton)
		config.Current.Reload()
	}

	timeAreaScaleDownButton.Click = func() {
		g.Preview = false
		defer g.buttonSpam(timeAreaScaleDownButton)

		timeAreaScaleDownButton.Active = !timeAreaScaleDownButton.Active
		config.Current.Scales.Time -= .01

		timeAreaScaleScaleButtons(timeAreaScaleUpButton, timeAreaScaleDownButton)
		config.Current.Reload()
	}

	scoreArea := &area.Area{
		Text: "Score",
		Min:  config.Current.Scores.Min.Div(2),
		Max:  config.Current.Scores.Max.Div(2),

		Button: &button.Button{
			Active:   true,
			Text:     "\t Score",
			Pressed:  color.NRGBA{G: 0xFF, A: 0x3F},
			Released: color.NRGBA{G: 0xFF, A: 0x4F},
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
	scoreAreaScaleText.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	scoreAreaScaleText.Alignment = text.Middle
	scoreAreaScaleText.TextSize = unit.Sp(11)

	scoreAreaScaleValueText := material.H5(th, "1x")
	scoreAreaScaleValueText.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
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

		scoreAreaScaleScaleButtons(scoreAreaScaleUpButton, scoreAreaScaleDownButton)
		config.Current.Reload()
	}

	scoreAreaScaleDownButton.Click = func() {
		g.Preview = false
		defer g.buttonSpam(scoreAreaScaleDownButton)

		scoreAreaScaleDownButton.Active = !scoreAreaScaleDownButton.Active
		config.Current.Scales.Score -= .01

		scoreAreaScaleScaleButtons(scoreAreaScaleUpButton, scoreAreaScaleDownButton)
		config.Current.Reload()
	}

	pointsAreaScaleScaleButtons(pointsAreaScaleUpButton, pointsAreaScaleDownButton)
	timeAreaScaleScaleButtons(timeAreaScaleUpButton, timeAreaScaleDownButton)
	scoreAreaScaleScaleButtons(scoreAreaScaleUpButton, scoreAreaScaleDownButton)

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

	resetButton := &button.Button{
		Text:     "\t Reset",
		Pressed:  color.NRGBA{R: 0xFF, G: 0xF, B: 0xF, A: 0x3F},
		Released: color.NRGBA{R: 0xF, G: 0xFF, B: 0xF, A: 0x3F},
		Active:   true,
	}

	resetButton.Click = func() {
		err := config.Current.Reset()
		if err != nil {
			log.Error().Err(err).Msg("failed to reset config")
			g.Log("failed to reset configuration (%s)", err.Error())
		}

		config.Current.Reload()

		resetButton.Active = !resetButton.Active
	}

	title := material.H5(th, "Pokemon Unite HUD Server")
	title.Color = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	title.Alignment = text.Middle

	g.once.Do(func() {
		go g.run(func() { g.matchScore(scoreArea) })
		go g.run(func() { g.matchPoints(pointsArea) })
		go g.run(func() { g.matchTime(timeArea) })
	})

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
										Top:   unit.Px(15),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return saveButton.Layout(gtx)
										})

									layout.Inset{
										Left:  unit.Px(float32(gtx.Constraints.Max.X - 220)),
										Right: unit.Px(10),
										Top:   unit.Px(15),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return cancelButton.Layout(gtx)
										})

									layout.Inset{
										Left:  unit.Px(float32(gtx.Constraints.Max.X - 330)),
										Right: unit.Px(10),
										Top:   unit.Px(15),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return screenButton.Layout(gtx)
										})

									layout.Inset{
										Left:  unit.Px(float32(gtx.Constraints.Max.X - 440)),
										Right: unit.Px(10),
										Top:   unit.Px(15),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return resetButton.Layout(gtx)
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
											return pointsArea.Button.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(125),
										Top:  unit.Px(35),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return pointsAreaScaleDownButton.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(160),
										Top:  unit.Px(35),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return pointsAreaScaleText.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(160),
										Top:  unit.Px(50),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											pointsAreaScaleValueText.Text = fmt.Sprintf("%.2fx", config.Current.Scales.Points)
											return pointsAreaScaleValueText.Layout(gtx)
										})

									layout.Inset{
										Left: unit.Px(195),
										Top:  unit.Px(35),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return pointsAreaScaleUpButton.Layout(gtx)
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
			pointsArea.Layout(gtx)
			timeArea.Layout(gtx)

			e.Frame(gtx.Ops)
		}
	}

	return next, nil
}

func (g *GUI) buttonSpam(b *button.Button) {
	b.LastPressed = time.Now()

	time.AfterFunc(time.Second, func() {
		if time.Since(b.LastPressed) >= time.Second {
			g.Preview = true
		}
	})
}

func timeAreaScaleScaleButtons(scaleUpButton, scaleDownButton *button.Button) {
	scaleDownButton.Disabled = false
	scaleDownButton.Released = color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F}
	if config.Current.Scales.Time < 0.01 {
		config.Current.Scales.Time = 0.05
		scaleDownButton.Released = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F}
		scaleDownButton.Disabled = true
	}
	scaleUpButton.Disabled = false
	scaleUpButton.Released = color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F}
	if config.Current.Scales.Time > 0.99 {
		config.Current.Scales.Time = 1.0
		scaleUpButton.Released = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F}
		scaleUpButton.Disabled = true
	}
}

func pointsAreaScaleScaleButtons(scaleUpButton, scaleDownButton *button.Button) {
	scaleDownButton.Disabled = false
	scaleDownButton.Released = color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F}
	if config.Current.Scales.Points < 0.01 {
		config.Current.Scales.Points = 0.05
		scaleDownButton.Released = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F}
		scaleDownButton.Disabled = true
	}
	scaleUpButton.Disabled = false
	scaleUpButton.Released = color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F}
	if config.Current.Scales.Points > 0.99 {
		config.Current.Scales.Points = 1.0
		scaleUpButton.Released = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F}
		scaleUpButton.Disabled = true
	}
}

func scoreAreaScaleScaleButtons(scaleUpButton, scaleDownButton *button.Button) {
	scaleDownButton.Disabled = false
	scaleDownButton.Released = color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F}
	if config.Current.Scales.Score < 0.01 {
		config.Current.Scales.Score = 0.05
		scaleDownButton.Released = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F}
		scaleDownButton.Disabled = true
	}
	scaleUpButton.Disabled = false
	scaleUpButton.Released = color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F}
	if config.Current.Scales.Score > 0.99 {
		config.Current.Scales.Score = 1.0
		scaleUpButton.Released = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F}
		scaleUpButton.Disabled = true
	}
}

func (g *GUI) matchPoints(a *area.Area) {
	if !g.Preview {
		a.NRGBA = color.NRGBA{A: 0x99}
		return
	}

	a.NRGBA = color.NRGBA{R: 0xFF, A: 0x99}
}

func (g *GUI) matchScore(a *area.Area) {
	if !g.Preview {
		a.NRGBA = color.NRGBA{A: 0x99}
		return
	}

	a.NRGBA = color.NRGBA{R: 0xFF, A: 0x99}
	a.Subtext = ""

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
		ok, score := m.Matches(matrix, g.Image, templates)
		if ok {
			a.NRGBA = color.NRGBA{G: 0xFF, A: 0x99}
			a.Subtext = fmt.Sprintf("(+%d)", score)
			return
		}
	}
}

func (g *GUI) matchTime(a *area.Area) {
	if !g.Preview {
		a.NRGBA = color.NRGBA{A: 0x99}
		return
	}

	a.NRGBA = color.NRGBA{R: 0xFF, A: 0x99}
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
		a.NRGBA = color.NRGBA{G: 0xFF, A: 0x99}
		a.Subtext = "(" + k + ")"
	}
}

func (g *GUI) run(fn func()) {
	for {
		fn()

		time.Sleep(time.Second)
	}
}
