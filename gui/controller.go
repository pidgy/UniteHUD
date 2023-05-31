package gui

import (
	"fmt"
	"image"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/dropdown"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/nrgba"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/video/window/electron"
)

const (
	scoringLabel    = "Scoring"
	energyLabel     = "Energy / Self Scoring"
	objectivesLabel = "Objectives"
	timeLabel       = "Time"
	kosLabel        = "KOs"
	koSelfLabel     = "Self KOs"
	allLabel        = "All"

	startLabel = " Start"
	stopLabel  = " Stop"
)

var (
	controller      = false
	controllerTitle = fmt.Sprintf("UniteHUD %s Controller", global.Version)

	colors = map[string]nrgba.NRGBA{
		"enabled":               nrgba.DarkSeafoam.Alpha(0xC0),
		"disabled":              nrgba.PaleRed,
		"pressed":               nrgba.Transparent30,
		"released":              nrgba.Gray,
		"regi-icon":             nrgba.DarkYellow.Alpha(200),
		"header":                nrgba.White,
		"header-bar-horizontal": nrgba.White.Alpha(50),
	}

	offsets = map[string]unit.Dp{
		"score":       unit.Dp(95),
		"+1":          unit.Dp(140),
		"+10":         unit.Dp(185),
		"+50":         unit.Dp(230),
		"+100":        unit.Dp(275),
		"+/-":         unit.Dp(345),
		"header":      unit.Dp(10),
		"line1":       unit.Dp(8),
		"line2":       unit.Dp(10),
		"regi1":       unit.Dp(140),
		"regi2":       unit.Dp(210),
		"regi3":       unit.Dp(280),
		"regi-orange": unit.Dp(0),
		"regi-purple": unit.Dp(20),
		"regi-label":  unit.Dp(45),
		"regi-icon":   unit.Dp(15),
	}

	orange = map[string]*button.Button{
		"+1":   plusButton(team.Orange, 1),
		"+10":  plusButton(team.Orange, 10),
		"+50":  plusButton(team.Orange, 50),
		"+100": plusButton(team.Orange, 100),
	}

	purple = map[string]*button.Button{
		"+1":   plusButton(team.Purple, 1),
		"+10":  plusButton(team.Purple, 10),
		"+50":  plusButton(team.Purple, 50),
		"+100": plusButton(team.Purple, 100),
	}

	self = map[string]*button.Button{
		"+1":   plusButton(team.Self, 1),
		"+10":  plusButton(team.Self, 10),
		"+50":  plusButton(team.Self, 50),
		"+100": plusButton(team.Self, 100),
	}

	scores = map[*team.Team]map[string]*button.Button{
		team.Orange: orange,
		team.Purple: purple,
		team.Self:   self,
	}

	negated = map[*team.Team]*button.Button{
		team.Orange: plusMinusButton(team.Orange),
		team.Purple: plusMinusButton(team.Purple),
		team.Self:   plusMinusButton(team.Self),
	}

	regis = map[string]*button.CircleButton{
		"purple-regice-1": bottomObjectiveButton(team.Purple, "regice", 1),
		"purple-regice-2": bottomObjectiveButton(team.Purple, "regice", 2),
		"purple-regice-3": bottomObjectiveButton(team.Purple, "regice", 3),
		"orange-regice-1": bottomObjectiveButton(team.Orange, "regice", 1),
		"orange-regice-2": bottomObjectiveButton(team.Orange, "regice", 2),
		"orange-regice-3": bottomObjectiveButton(team.Orange, "regice", 3),

		"purple-regieleki-1": regielekiButton(team.Purple, 1),
		"purple-regieleki-2": regielekiButton(team.Purple, 2),
		"purple-regieleki-3": regielekiButton(team.Purple, 3),
		"orange-regieleki-1": regielekiButton(team.Orange, 1),
		"orange-regieleki-2": regielekiButton(team.Orange, 2),
		"orange-regieleki-3": regielekiButton(team.Orange, 3),

		"purple-regirock-1": bottomObjectiveButton(team.Purple, "regirock", 1),
		"purple-regirock-2": bottomObjectiveButton(team.Purple, "regirock", 2),
		"purple-regirock-3": bottomObjectiveButton(team.Purple, "regirock", 3),
		"orange-regirock-1": bottomObjectiveButton(team.Orange, "regirock", 1),
		"orange-regirock-2": bottomObjectiveButton(team.Orange, "regirock", 2),
		"orange-regirock-3": bottomObjectiveButton(team.Orange, "regirock", 3),

		"purple-registeel-1": bottomObjectiveButton(team.Purple, "registeel", 1),
		"purple-registeel-2": bottomObjectiveButton(team.Purple, "registeel", 2),
		"purple-registeel-3": bottomObjectiveButton(team.Purple, "registeel", 3),
		"orange-registeel-1": bottomObjectiveButton(team.Orange, "registeel", 1),
		"orange-registeel-2": bottomObjectiveButton(team.Orange, "registeel", 2),
		"orange-registeel-3": bottomObjectiveButton(team.Orange, "registeel", 3),
	}

	matching = &dropdown.List{
		Items: []*dropdown.Item{
			{
				Text: scoringLabel,
				Checked: widget.Bool{
					Value: !config.Current.DisableScoring,
				},
				Callback: func(this *dropdown.Item) {
					config.Current.DisableScoring = this.Checked.Value
				},
			},
			{
				Text: energyLabel,
				Checked: widget.Bool{
					Value: !config.Current.DisableEnergy,
				},
				Callback: func(this *dropdown.Item) {
					config.Current.DisableEnergy = this.Checked.Value
				},
			},
			{
				Text: objectivesLabel,
				Checked: widget.Bool{
					Value: !config.Current.DisableObjectives,
				},
				Callback: func(this *dropdown.Item) {
					config.Current.DisableObjectives = this.Checked.Value
				},
			},
			{
				Text: timeLabel,
				Checked: widget.Bool{
					Value: !config.Current.DisableTime,
				},
				Callback: func(this *dropdown.Item) {
					config.Current.DisableTime = this.Checked.Value
				},
			},
			{
				Text: kosLabel,
				Checked: widget.Bool{
					Value: !config.Current.DisableKOs,
				},
				Callback: func(this *dropdown.Item) {
					config.Current.DisableKOs = this.Checked.Value
				},
			},
			{
				Text: koSelfLabel,
				Checked: widget.Bool{
					Value: !config.Current.DisableDefeated,
				},
				Callback: func(this *dropdown.Item) {
					config.Current.DisableDefeated = this.Checked.Value
				},
			},
			{
				Text: allLabel,
				Callback: func(_ *dropdown.Item) {
					// Empty to force devicelist callback.
				},
			},
		},
		Callback: func(i *dropdown.Item, _ *dropdown.List) {
			if i.Text == allLabel {
				config.Current.DisableScoring = i.Checked.Value
				config.Current.DisableEnergy = i.Checked.Value
				config.Current.DisableObjectives = i.Checked.Value
				config.Current.DisableTime = i.Checked.Value
				config.Current.DisableKOs = i.Checked.Value
				config.Current.DisableDefeated = i.Checked.Value
				i.Checked.Value = !i.Checked.Value
			}

			status := "enabled"
			if !i.Checked.Value {
				status = "disabled"
			}

			notify.System("%s matching %s for %s profile", i.Text, status, config.Current.Profile)

			err := config.Current.Save()
			if err != nil {
				notify.Error("Failed to save configuration (%v)", err)
			}
		},
	}

	profile = &dropdown.List{
		Radio: true,
		Items: []*dropdown.Item{
			{
				Text: strings.Title(config.ProfilePlayer),
				Checked: widget.Bool{
					Value: config.Current.Profile == config.ProfilePlayer,
				},
			},
			{
				Text: strings.Title(config.ProfileBroadcaster),
				Checked: widget.Bool{
					Value: config.Current.Profile == config.ProfileBroadcaster,
				},
			},
		},
		Callback: func(i *dropdown.Item, _ *dropdown.List) {
			if config.Current.Profile == strings.ToLower(i.Text) {
				return
			}

			electron.Close()

			config.Current.Profile = strings.ToLower(i.Text)

			err := config.Load(config.Current.Profile)
			if err != nil {
				notify.Error("Failed to load %s profile configuration", config.Current.Profile)
				return
			}

			if config.Current.Window == config.BrowserWindow {
				err = electron.Open()
				if err != nil {
					notify.Error("Failed to open %s (%v)", config.BrowserWindow, err)
				}
			}

			notify.System("Profile set to %s mode", i.Text)

			time.AfterFunc(time.Second, func() {
				err := config.Current.Save()
				if err != nil {
					notify.Error("Failed to save %s profile configuration", config.Current.Profile)
				}
			})
		},
	}

	controls = map[string]*button.Button{
		"start-stop": startStopButton(),
		"clear":      clearButton(),
	}

	images = map[string]image.Image{
		"regice":    nil,
		"regieleki": nil,
		"regirock":  nil,
		"registeel": nil,
	}

	advanced = map[string]*button.Button{
		"reset": resetConfigurationButton(),
	}

	general = &dropdown.List{
		Radio: true,
		Items: []*dropdown.Item{
			{
				Text: "Previews",
				Hint: "To ease CPU usage, avoid rendering successful captures on the main screen",
				Checked: widget.Bool{
					Value: config.Current.DisablePreviews,
				},
				Callback: func(this *dropdown.Item) {
					config.Current.DisablePreviews = this.Checked.Value
				},
			},
			{
				Text: strings.Title(config.ProfileBroadcaster),
				Checked: widget.Bool{
					Value: config.Current.Profile == config.ProfileBroadcaster,
				},
			},
		},
		Callback: func(_ *dropdown.Item, l *dropdown.List) {

		},
	}
)

func (g *GUI) controller() {
	list := material.List(g.normal, &widget.List{
		Scrollbar: widget.Scrollbar{},
		List: layout.List{
			Axis:      layout.Vertical,
			Alignment: layout.Baseline,
		},
	})

	go func() {
		controller = true
		defer func() { controller = false }()

		for k := range images {
			mat := gocv.IMRead(fmt.Sprintf(`%s/icon/%s.png`, config.Current.Assets(), k), gocv.IMReadColor)
			img, err := mat.ToImage()
			if err != nil {
				g.ToastError(err)
				return
			}

			images[k] = img
		}

		dx, dy := float32(400), float32(715)

		w := app.NewWindow(
			app.Title(controllerTitle),
			app.Size(unit.Dp(dx), unit.Dp(dy)),
		)

		var ops op.Ops

		for e := range w.Events() {
			switch e.(type) {
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e.(system.FrameEvent))

				updateControllerUI()

				colorBox(gtx, gtx.Constraints.Max, nrgba.BackgroundAlt)

				o, p, s := server.Scores()

				y := float32(0)

				list.Layout(gtx, 70, func(gtx layout.Context, index int) layout.Dimensions {
					// Frame Event Layout.
					y += 5

					layout.Inset{
						Top:  unit.Dp(y),
						Left: unit.Dp(0),
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(
							g.normal,
							g.normal.TextSize*unit.Sp(20.0/16.0),
							controllerTitle,
						)
						l.Font.Weight = 100
						l.Color = colors["header"].Color()
						l.Alignment = text.Middle
						return l.Layout(gtx)
					})

					y += 20

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["header"],
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(
							g.normal,
							g.normal.TextSize*unit.Sp(18.0/16.0),
							"Profile",
						)
						l.Font.Weight = 200
						l.Color = colors["header"].Color()
						return l.Layout(gtx)
					})

					y += 25

					layout.Inset{
						Top:  unit.Dp(y),
						Left: unit.Dp(5),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return profile.Layout(gtx, g.normal)
						})

					y += 50

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["line2"],
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X-int(offsets["line2"]), 1)}.Push(gtx.Ops).Pop()
						paint.ColorOp{Color: nrgba.White.Alpha(0x05).Color()}.Add(gtx.Ops)
						paint.PaintOp{}.Add(gtx.Ops)
						return layout.Dimensions{Size: gtx.Constraints.Max}
					})

					y += 15

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["header"],
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(
							g.normal,
							g.normal.TextSize*unit.Sp(18.0/16.0),
							"Score",
						)
						l.Font.Weight = 200
						l.Color = colors["header"].Color()
						return l.Layout(gtx)
					})

					y += 25

					layout.Inset{
						Top:  unit.Dp(y),
						Left: unit.Dp(30),
					}.Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							l := material.Label(
								g.normal,
								g.normal.TextSize*unit.Sp(15.0/16.0),
								"Purple",
							)
							l.Color = colors["header"].Color()

							return l.Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["score"],
					}.Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							l := material.Label(
								g.normal,
								g.normal.TextSize*unit.Sp(15.0/16.0),
								fmt.Sprintf("%d", p),
							)
							l.Font.Weight = 300
							l.Color = nrgba.Purple.Color()

							return l.Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: offsets["+1"],
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return purple["+1"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: offsets["+10"],
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return purple["+10"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: offsets["+50"],
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return purple["+50"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: offsets["+100"],
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return purple["+100"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: offsets["+/-"],
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							negated[team.Purple].Released = colors["enabled"]
							if negated[team.Purple].Text == "-" {
								negated[team.Purple].Released = nrgba.Red.Alpha(0xCC)
							}

							return negated[team.Purple].Layout(gtx)
						})

					y += 25

					layout.Inset{
						Top:  unit.Dp(y),
						Left: unit.Dp(30),
					}.Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							l := material.Label(
								g.normal,
								g.normal.TextSize*unit.Sp(15.0/16.0),
								"Orange",
							)
							l.Color = colors["header"].Color()

							return l.Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["score"],
					}.Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							l := material.Label(
								g.normal,
								g.normal.TextSize*unit.Sp(15.0/16.0),
								fmt.Sprintf("%d", o),
							)
							l.Font.Weight = 300
							l.Color = nrgba.Orange.Color()

							return l.Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: offsets["+1"],
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return orange["+1"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: offsets["+10"],
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return orange["+10"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: offsets["+50"],
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return orange["+50"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: offsets["+100"],
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return orange["+100"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: offsets["+/-"],
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							negated[team.Orange].Released = colors["enabled"]
							if negated[team.Orange].Text == "-" {
								negated[team.Orange].Released = nrgba.Red.Alpha(0xCC)
							}

							return negated[team.Orange].Layout(gtx)
						})

					y += 25

					layout.Inset{
						Top:  unit.Dp(y),
						Left: unit.Dp(30),
					}.Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							l := material.Label(
								g.normal,
								g.normal.TextSize*unit.Sp(15.0/16.0),
								"Self",
							)
							l.Color = colors["header"].Color()

							return l.Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["score"],
					}.Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							l := material.Label(
								g.normal,
								g.normal.TextSize*unit.Sp(15.0/16.0),
								fmt.Sprintf("%d", s),
							)
							l.Font.Weight = 300
							l.Color = nrgba.User.Color()

							return l.Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: offsets["+1"],
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return self["+1"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: offsets["+10"],
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return self["+10"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: offsets["+50"],
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return self["+50"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: offsets["+100"],
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return self["+100"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: offsets["+/-"],
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							negated[team.Self].Released = colors["enabled"]
							if negated[team.Self].Text == "-" {
								negated[team.Self].Released = nrgba.Red.Alpha(0xCC)
							}

							return negated[team.Self].Layout(gtx)
						})

					y += 30

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["line2"],
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X-int(offsets["line2"]), 1)}.Push(gtx.Ops).Pop()
						paint.ColorOp{Color: nrgba.White.Alpha(0x05).Color()}.Add(gtx.Ops)
						paint.PaintOp{}.Add(gtx.Ops)
						return layout.Dimensions{Size: gtx.Constraints.Max}
					})

					y += 15

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["header"],
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(
							g.normal,
							g.normal.TextSize*unit.Sp(18.0/16.0),
							"Objectives        1         2         3",
						)
						l.Font.Weight = 200
						l.Color = colors["header"].Color()
						return l.Layout(gtx)
					})

					y += 25

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["regi-icon"],
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return (widget.Image{
							Src:      paint.NewImageOp(images["regieleki"]),
							Position: layout.NW,
							Scale:    .2,
						}).Layout(gtx)
					})

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["regi-label"],
					}.Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							l := material.Label(
								g.normal,
								g.normal.TextSize*unit.Sp(15.0/16.0),
								"Regieleki",
							)
							l.Color = colors["header"].Color()

							return l.Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi1"] + offsets["regi-orange"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["orange-regieleki-1"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi1"] + offsets["regi-purple"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["purple-regieleki-1"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi2"] + offsets["regi-orange"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["orange-regieleki-2"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi2"] + offsets["regi-purple"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["purple-regieleki-2"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi3"] + offsets["regi-orange"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["orange-regieleki-3"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi3"] + offsets["regi-purple"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["purple-regieleki-3"].Layout(gtx)
						})

					y += 25

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["regi-icon"],
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return (widget.Image{
							Src:      paint.NewImageOp(images["regirock"]),
							Position: layout.NW,
							Scale:    .2,
						}).Layout(gtx)
					})

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["regi-label"],
					}.Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							l := material.Label(
								g.normal,
								g.normal.TextSize*unit.Sp(15.0/16.0),
								"Registeel",
							)
							l.Color = colors["header"].Color()

							return l.Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi1"] + offsets["regi-orange"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["orange-registeel-1"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi1"] + offsets["regi-purple"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["purple-registeel-1"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi2"] + offsets["regi-orange"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["orange-registeel-2"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi2"] + offsets["regi-purple"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["purple-registeel-2"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi3"] + offsets["regi-orange"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["orange-registeel-3"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi3"] + offsets["regi-purple"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["purple-registeel-3"].Layout(gtx)
						})

					y += 25

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["regi-icon"],
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return (widget.Image{
							Src:      paint.NewImageOp(images["regirock"]),
							Position: layout.NW,
							Scale:    .2,
						}).Layout(gtx)
					})

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["regi-label"],
					}.Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							l := material.Label(
								g.normal,
								g.normal.TextSize*unit.Sp(15.0/16.0),
								"Regirock",
							)
							l.Color = colors["header"].Color()

							return l.Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi1"] + offsets["regi-orange"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["orange-regirock-1"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi1"] + offsets["regi-purple"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["purple-regirock-1"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi2"] + offsets["regi-orange"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["orange-regirock-2"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi2"] + offsets["regi-purple"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["purple-regirock-2"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi3"] + offsets["regi-orange"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["orange-regirock-3"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi3"] + offsets["regi-purple"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["purple-regirock-3"].Layout(gtx)
						})

					y += 25

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["regi-icon"],
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return (widget.Image{
							Src:      paint.NewImageOp(images["regice"]),
							Position: layout.NW,
							Scale:    .2,
						}).Layout(gtx)
					})

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["regi-label"],
					}.Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							l := material.Label(
								g.normal,
								g.normal.TextSize*unit.Sp(15.0/16.0),
								"Regice",
							)
							l.Color = colors["header"].Color()

							return l.Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi1"] + offsets["regi-orange"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["orange-regice-1"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi1"] + offsets["regi-purple"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["purple-regice-1"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi2"] + offsets["regi-orange"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["orange-regice-2"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi2"] + offsets["regi-purple"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["purple-regice-2"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi3"] + offsets["regi-orange"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["orange-regice-3"].Layout(gtx)
						})

					layout.Inset{
						Top:  unit.Dp(y + 2),
						Left: unit.Dp(offsets["regi3"] + offsets["regi-purple"]),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return regis["purple-regice-3"].Layout(gtx)
						})

					y += 30

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["line2"],
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X-int(offsets["line2"]), 1)}.Push(gtx.Ops).Pop()
						paint.ColorOp{Color: nrgba.White.Alpha(0x05).Color()}.Add(gtx.Ops)
						paint.PaintOp{}.Add(gtx.Ops)
						return layout.Dimensions{Size: gtx.Constraints.Max}
					})

					y += 15

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["header"],
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(
							g.normal,
							g.normal.TextSize*unit.Sp(18.0/16.0),
							"Matching",
						)
						l.Font.Weight = 200
						l.Color = colors["header"].Color()
						return l.Layout(gtx)
					})

					y += 25

					layout.Inset{
						Left: unit.Dp(5),
						Top:  unit.Dp(y),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return matching.Layout(gtx, g.normal)
						})

					y += 155

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["line2"],
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X-int(offsets["line2"]), 1)}.Push(gtx.Ops).Pop()
						paint.ColorOp{Color: nrgba.White.Alpha(0x05).Color()}.Add(gtx.Ops)
						paint.PaintOp{}.Add(gtx.Ops)
						return layout.Dimensions{Size: gtx.Constraints.Max}
					})

					y += 15

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["header"],
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(
							g.normal,
							g.normal.TextSize*unit.Sp(18.0/16.0),
							"Game",
						)
						l.Font.Weight = 200
						l.Color = colors["header"].Color()
						return l.Layout(gtx)
					})

					y += 25

					layout.Inset{
						Top:   unit.Dp(y),
						Left:  unit.Dp(10),
						Right: unit.Dp(20),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return controls["start-stop"].Layout(gtx)
						})

					layout.Inset{
						Top:   unit.Dp(y),
						Left:  unit.Dp(65),
						Right: unit.Dp(20),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return controls["clear"].Layout(gtx)
						})

					y += 40

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["line2"],
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X-int(offsets["line2"]), 1)}.Push(gtx.Ops).Pop()
						paint.ColorOp{Color: nrgba.White.Alpha(0x05).Color()}.Add(gtx.Ops)
						paint.PaintOp{}.Add(gtx.Ops)
						return layout.Dimensions{Size: gtx.Constraints.Max}
					})

					y += 15

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["header"],
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(
							g.normal,
							g.normal.TextSize*unit.Sp(18.0/16.0),
							"General",
						)
						l.Font.Weight = 200
						l.Color = colors["header"].Color()
						return l.Layout(gtx)
					})

					y += 25

					layout.Inset{
						Top:  unit.Dp(y),
						Left: unit.Dp(5),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return general.Layout(gtx, g.normal)
						})

					y += 60

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["line2"],
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X-int(offsets["line2"]), 1)}.Push(gtx.Ops).Pop()
						paint.ColorOp{Color: nrgba.White.Alpha(0x05).Color()}.Add(gtx.Ops)
						paint.PaintOp{}.Add(gtx.Ops)
						return layout.Dimensions{Size: gtx.Constraints.Max}
					})

					y += 15

					layout.Inset{
						Top:  unit.Dp(y),
						Left: offsets["header"],
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(
							g.normal,
							g.normal.TextSize*unit.Sp(18.0/16.0),
							"Advanced",
						)
						l.Font.Weight = 200
						l.Color = colors["header"].Color()
						return l.Layout(gtx)
					})

					y += 25

					layout.Inset{
						Top:   unit.Dp(y),
						Left:  unit.Dp(10),
						Right: unit.Dp(20),
					}.Layout(
						gtx,
						func(gtx layout.Context) layout.Dimensions {
							return advanced["reset"].Layout(gtx)
						})

					return layout.Dimensions{Size: gtx.Constraints.Max}
				})

				e.(system.FrameEvent).Frame(gtx.Ops)
			case system.StageEvent:
				w.Perform(system.ActionRaise)
			case system.DestroyEvent:
				return
			}
		}
	}()
}

func bottomObjectiveButton(t *team.Team, name string, n int) *button.CircleButton {
	b := &button.CircleButton{
		Name:        t.Name,
		BorderColor: t.NRGBA,
		Released:    colors["released"],
		Pressed:     colors["pressed"],
		Size:        image.Pt(15, 15),
		BorderWidth: unit.Sp(.5),
	}

	b.Click = func(b *button.CircleButton) {
		defer b.Deactivate()

		server.SetBottomObjective(t, name, n-1)
	}

	return b
}

func clearButton() *button.Button {
	b := &button.Button{
		Text:     "Clear",
		Released: colors["disabled"],
		Pressed:  colors["pressed"],
		Size:     image.Pt(50, 25),
		TextSize: unit.Sp(12),

		BorderWidth: unit.Sp(.5),
	}

	b.Click = func(b *button.Button) {
		defer b.Deactivate()

		ok := server.Match()
		server.Clear()
		if ok {
			server.SetMatchStarted()
			server.SetTime(10, 0)
		}
	}

	return b
}

func plusButton(t *team.Team, n int) *button.Button {
	b := &button.Button{
		Text:        fmt.Sprintf("+%d", n),
		Released:    colors["released"],
		Pressed:     colors["pressed"],
		Size:        image.Pt(35, 16),
		TextSize:    unit.Sp(12),
		BorderWidth: unit.Sp(.5),
	}

	b.Click = func(b *button.Button) {
		defer b.Deactivate()

		val := 1
		if b.Text[0] == '-' {
			val = -1
		}

		val = n * val
		if server.Score(t)+val < 0 {
			val = -server.Score(t)
		}

		server.SetScore(t, val)
	}

	return b
}

func plusMinusButton(t *team.Team) *button.Button {
	b := &button.Button{
		Text:     "+",
		Released: colors["enabled"],
		Pressed:  colors["pressed"],
		Size:     image.Pt(40, 16),
		TextSize: unit.Sp(14),

		BorderWidth: unit.Sp(.5),
	}

	b.Click = func(b *button.Button) {
		defer b.Deactivate()

		switch b.Text {
		case "+":
			b.Text = "-"
		case "-":
			b.Text = "+"
		}

		for _, b2 := range scores[t] {
			if b2.Text == b.Text {
				continue
			}

			b2.Text = b.Text[:1] + b2.Text[1:]
		}
	}

	return b
}

func regielekiButton(t *team.Team, n int) *button.CircleButton {
	b := &button.CircleButton{
		Name:        t.Name,
		BorderColor: t.NRGBA,
		Released:    colors["released"],
		Pressed:     colors["pressed"],
		Size:        image.Pt(15, 15),
		BorderWidth: unit.Sp(.5),
	}

	b.Click = func(b *button.CircleButton) {
		defer b.Deactivate()

		server.SetRegielekiAt(t, n-1)
	}

	return b
}

func resetConfigurationButton() *button.Button {
	b := &button.Button{
		Text:     "Configuration",
		Released: colors["disabled"],
		Pressed:  colors["pressed"],
		Size:     image.Pt(125, 25),
		TextSize: unit.Sp(12),

		BorderWidth: unit.Sp(.5),
		Click: func(b *button.Button) {
			Window.ToastYesNo("Reset", fmt.Sprintf("Reset UniteHUD %s configuration?", config.Current.Profile), func() {
				defer b.Deactivate()
				defer server.Clear()

				electron.Close()

				err := config.Current.Reset()
				if err != nil {
					notify.Error("Failed to reset %s configuration (%v)", config.Current.Profile, err)
				}

				config.Current.Reload()

				notify.Announce("Reset UniteHUD %s configuration", config.Current.Profile)
			}, b.Deactivate)
		},
	}

	return b
}

func startStopButton() *button.Button {
	b := &button.Button{
		Text:     startLabel,
		Released: colors["enabled"],
		Pressed:  colors["pressed"],
		Size:     image.Pt(50, 25),
		TextSize: unit.Sp(12),

		BorderWidth: unit.Sp(.5),
	}

	b.Click = func(b *button.Button) {
		defer b.Deactivate()

		switch b.Text {
		case startLabel:
			b.Text = stopLabel
			b.Released = colors["disabled"]

			server.SetStarted()
			server.SetMatchStarted()
			server.SetTime(10, 0)
		case stopLabel:
			b.Text = startLabel
			b.Released = colors["enabled"]

			server.SetMatchStopped()
			server.SetStopped()
			server.SetTime(0, 0)
		}
	}

	return b
}

func updateControllerUI() {
	for _, b := range self {
		b.Disabled = config.Current.Profile == config.ProfileBroadcaster
		b.Active = b.Disabled
	}

	negated[team.Self].Disabled = config.Current.Profile == config.ProfileBroadcaster
	negated[team.Self].Active = negated[team.Self].Disabled

	for _, i := range profile.Items {
		i.Checked.Value = config.Current.Profile == strings.ToLower(i.Text)
	}

	all := true

	for _, i := range matching.Items {
		if i.Text != allLabel && !i.Checked.Value {
			all = false
		}

		switch i.Text {
		case scoringLabel:
			i.Checked.Value = !config.Current.DisableScoring
			i.Disabled = config.Current.Profile == config.ProfileBroadcaster
		case energyLabel:
			i.Checked.Value = !config.Current.DisableEnergy
			i.Disabled = config.Current.Profile == config.ProfileBroadcaster
		case objectivesLabel:
			i.Checked.Value = !config.Current.DisableObjectives
		case timeLabel:
			i.Checked.Value = !config.Current.DisableTime
		case kosLabel:
			i.Checked.Value = !config.Current.DisableKOs
		case koSelfLabel:
			i.Checked.Value = !config.Current.DisableDefeated
			i.Disabled = config.Current.Profile == config.ProfileBroadcaster
		case allLabel:
			i.Checked.Value = all
		}
	}

	for _, v := range regis {
		v.Released = colors["released"]
	}

	for i, name := range server.Regielekis() {
		orange := regis[fmt.Sprintf("orange-regieleki-%d", i+1)]
		purple := regis[fmt.Sprintf("purple-regieleki-%d", i+1)]

		switch {
		case name == team.None.Name:
			orange.Released = colors["released"]
			purple.Released = colors["released"]
		case name == orange.Name:
			orange.Released = orange.BorderColor
			purple.Released = colors["released"]
		case name == purple.Name:
			orange.Released = colors["released"]
			purple.Released = purple.BorderColor
		}
	}

	for i, obj := range server.Bottom() {
		orange := regis[fmt.Sprintf("orange-%s-%d", obj.Name, i+1)]
		purple := regis[fmt.Sprintf("purple-%s-%d", obj.Name, i+1)]

		switch {
		case obj.Team == team.None.Name:
			orange.Released = colors["released"]
			purple.Released = colors["released"]
		case obj.Team == orange.Name:
			orange.Released = orange.BorderColor
			purple.Released = colors["released"]
		case obj.Team == purple.Name:
			orange.Released = colors["released"]
			purple.Released = purple.BorderColor
		}
	}

	if server.Match() {
		controls["start-stop"].Text = stopLabel
		controls["start-stop"].Released = colors["disabled"]
	} else {
		controls["start-stop"].Text = startLabel
		controls["start-stop"].Released = colors["enabled"]
	}
}
