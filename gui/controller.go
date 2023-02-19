package gui

import (
	"fmt"
	"image"
	"image/color"
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
	"github.com/pidgy/unitehud/gui/visual/screen"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/team"
)

const (
	scoringLabel    = "Scoring"
	energyLabel     = "Energy / Self Scoring"
	objectivesLabel = "Objectives"
	timeLabel       = "Time"
	kosLabel        = "KOs"
	koSelfLabel     = "Self KOs"

	startLabel = " Start"
	stopLabel  = " Stop"
)

var (
	controller      = false
	controllerTitle = fmt.Sprintf("UniteHUD %s Controller", global.Version)

	colors = map[string]color.NRGBA{
		"enabled":   rgba.N(rgba.Alpha(rgba.DarkSeafoam, 0xC0)),
		"disabled":  rgba.N(rgba.PaleRed),
		"pressed":   rgba.N(rgba.DarkGray),
		"released":  rgba.N(rgba.Gray),
		"regi-icon": rgba.N(rgba.Alpha(rgba.DarkYellow, 200)),
	}

	offsets = map[string]unit.Value{
		"score":       unit.Px(95),
		"+1":          unit.Px(140),
		"+10":         unit.Px(185),
		"+50":         unit.Px(230),
		"+100":        unit.Px(275),
		"+/-":         unit.Px(345),
		"header":      unit.Px(10),
		"line1":       unit.Px(8),
		"line2":       unit.Px(10),
		"regi1":       unit.Px(140),
		"regi2":       unit.Px(210),
		"regi3":       unit.Px(280),
		"regi-orange": unit.Px(0),
		"regi-purple": unit.Px(20),
		"regi-label":  unit.Px(45),
		"regi-icon":   unit.Px(15),
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
				Callback: func() {
					config.Current.DisableScoring = !config.Current.DisableScoring
				},
			},
			{
				Text: energyLabel,
				Checked: widget.Bool{
					Value: !config.Current.DisableEnergy,
				},
				Callback: func() {
					config.Current.DisableEnergy = !config.Current.DisableEnergy
				},
			},
			{
				Text: objectivesLabel,
				Checked: widget.Bool{
					Value: !config.Current.DisableObjectives,
				},
				Callback: func() {
					config.Current.DisableObjectives = !config.Current.DisableObjectives
				},
			},
			{
				Text: timeLabel,
				Checked: widget.Bool{
					Value: !config.Current.DisableTime,
				},
				Callback: func() {
					config.Current.DisableTime = !config.Current.DisableTime
				},
			},
			{
				Text: kosLabel,
				Checked: widget.Bool{
					Value: !config.Current.DisableKOs,
				},
				Callback: func() {
					config.Current.DisableKOs = !config.Current.DisableKOs
				},
			},
			{
				Text: koSelfLabel,
				Checked: widget.Bool{
					Value: !config.Current.DisableDefeated,
				},
				Callback: func() {
					config.Current.DisableDefeated = !config.Current.DisableDefeated
				},
			},
		},
		Callback: func(i *dropdown.Item) {
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
		Callback: func(i *dropdown.Item) {
			if config.Current.Profile == strings.ToLower(i.Text) {
				return
			}

			config.Current.Profile = strings.ToLower(i.Text)

			err := config.Load(config.Current.Profile)
			if err != nil {
				notify.Error("Failed to load %s profile configuration", config.Current.Profile)
				return
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
)

func (g *GUI) controller() {
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
			app.Size(unit.Px(dx), unit.Px(dy)),
		)

		var ops op.Ops

		for e := range w.Events() {
			switch e.(type) {
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e.(system.FrameEvent))

				updateControllerUI()

				colorBox(gtx, gtx.Constraints.Max, color.NRGBA{R: 25, G: 25, B: 25, A: 255})

				o, p, s := server.Scores()

				x := float32(0)

				// Frame Event Layout.
				{
					// Title Header.
					{
						x += 5

						layout.Inset{
							Top:  unit.Px(x),
							Left: unit.Px(0),
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							l := material.Label(
								g.normal,
								g.normal.TextSize.Scale(20.0/16.0),
								controllerTitle,
							)
							l.Font.Weight = 100
							l.Color = rgba.N(rgba.White)
							l.Alignment = text.Middle
							return l.Layout(gtx)
						},
						)
					}

					// Title Line.
					{
						x += 25

						layout.Inset{
							Top:  unit.Px(x),
							Left: unit.Px(5),
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X-5, 2)}.Push(gtx.Ops).Pop()
							paint.ColorOp{Color: rgba.N(rgba.Alpha(rgba.White, 0x5F))}.Add(gtx.Ops)
							paint.PaintOp{}.Add(gtx.Ops)
							return layout.Dimensions{Size: gtx.Constraints.Max}
						},
						)
					}

					// Profile Header.
					{
						x += 20

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["header"],
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							l := material.Label(
								g.normal,
								g.normal.TextSize.Scale(18.0/16.0),
								"Profile",
							)
							l.Font.Weight = 200
							l.Color = rgba.N(rgba.White)
							return l.Layout(gtx)
						},
						)
					}

					// Profile Section Line 1.
					{
						x += 25

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["line1"],
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X-int(offsets["line1"].V), 1)}.Push(gtx.Ops).Pop()
							paint.ColorOp{Color: rgba.N(rgba.Alpha(rgba.White, 0x5F))}.Add(gtx.Ops)
							paint.PaintOp{}.Add(gtx.Ops)
							return layout.Dimensions{Size: gtx.Constraints.Max}
						},
						)
					}

					// Profile.
					{
						x += 5

						layout.Inset{
							Top:  unit.Px(x),
							Left: unit.Px(5),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return profile.Layout(gtx, g.normal)
							},
						)
					}

					// Profile Section Line 2.
					{
						x += 50

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["line2"],
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X-int(offsets["line2"].V), 1)}.Push(gtx.Ops).Pop()
							paint.ColorOp{Color: rgba.N(rgba.Alpha(rgba.White, 0x05))}.Add(gtx.Ops)
							paint.PaintOp{}.Add(gtx.Ops)
							return layout.Dimensions{Size: gtx.Constraints.Max}
						},
						)
					}

					// Scores Header.
					{
						x += 15

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["header"],
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							l := material.Label(
								g.normal,
								g.normal.TextSize.Scale(18.0/16.0),
								"Score",
							)
							l.Font.Weight = 200
							l.Color = rgba.N(rgba.White)
							return l.Layout(gtx)
						},
						)
					}

					// Scores Section Line 1.
					{
						x += 25

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["line1"],
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X-int(offsets["line1"].V), 1)}.Push(gtx.Ops).Pop()
							paint.ColorOp{Color: rgba.N(rgba.Alpha(rgba.White, 0x5F))}.Add(gtx.Ops)
							paint.PaintOp{}.Add(gtx.Ops)
							return layout.Dimensions{Size: gtx.Constraints.Max}
						},
						)
					}

					// Purple.
					{
						x += 5

						layout.Inset{
							Top:  unit.Px(x),
							Left: unit.Px(30),
						}.Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								l := material.Label(
									g.normal,
									g.normal.TextSize.Scale(15.0/16.0),
									"Purple",
								)
								l.Color = rgba.N(rgba.White)

								return l.Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["score"],
						}.Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								l := material.Label(
									g.normal,
									g.normal.TextSize.Scale(15.0/16.0),
									fmt.Sprintf("%d", p),
								)
								l.Font.Weight = 300
								l.Color = rgba.N(rgba.Purple)

								return l.Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: offsets["+1"],
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return purple["+1"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: offsets["+10"],
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return purple["+10"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: offsets["+50"],
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return purple["+50"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: offsets["+100"],
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return purple["+100"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: offsets["+/-"],
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								negated[team.Purple].Released = colors["enabled"]
								if negated[team.Purple].Text == "-" {
									negated[team.Purple].Released = rgba.N(rgba.Alpha(rgba.Red, 0xCC))
								}

								return negated[team.Purple].Layout(gtx)
							},
						)
					}

					// Orange.
					{
						x += 25

						layout.Inset{
							Top:  unit.Px(x),
							Left: unit.Px(30),
						}.Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								l := material.Label(
									g.normal,
									g.normal.TextSize.Scale(15.0/16.0),
									"Orange",
								)
								l.Color = rgba.N(rgba.White)

								return l.Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["score"],
						}.Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								l := material.Label(
									g.normal,
									g.normal.TextSize.Scale(15.0/16.0),
									fmt.Sprintf("%d", o),
								)
								l.Font.Weight = 300
								l.Color = rgba.N(rgba.Orange)

								return l.Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: offsets["+1"],
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return orange["+1"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: offsets["+10"],
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return orange["+10"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: offsets["+50"],
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return orange["+50"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: offsets["+100"],
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return orange["+100"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: offsets["+/-"],
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								negated[team.Orange].Released = colors["enabled"]
								if negated[team.Orange].Text == "-" {
									negated[team.Orange].Released = rgba.N(rgba.Alpha(rgba.Red, 0xCC))
								}

								return negated[team.Orange].Layout(gtx)
							},
						)
					}

					// Self.
					{
						x += 25

						layout.Inset{
							Top:  unit.Px(x),
							Left: unit.Px(30),
						}.Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								l := material.Label(
									g.normal,
									g.normal.TextSize.Scale(15.0/16.0),
									"Self",
								)
								l.Color = rgba.N(rgba.White)

								return l.Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["score"],
						}.Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								l := material.Label(
									g.normal,
									g.normal.TextSize.Scale(15.0/16.0),
									fmt.Sprintf("%d", s),
								)
								l.Font.Weight = 300
								l.Color = rgba.N(rgba.User)

								return l.Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: offsets["+1"],
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return self["+1"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: offsets["+10"],
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return self["+10"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: offsets["+50"],
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return self["+50"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: offsets["+100"],
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return self["+100"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: offsets["+/-"],
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								negated[team.Self].Released = colors["enabled"]
								if negated[team.Self].Text == "-" {
									negated[team.Self].Released = rgba.N(rgba.Alpha(rgba.Red, 0xCC))
								}

								return negated[team.Self].Layout(gtx)
							},
						)
					}

					// Scores Section Line 2.
					{
						x += 30

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["line2"],
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X-int(offsets["line2"].V), 1)}.Push(gtx.Ops).Pop()
							paint.ColorOp{Color: rgba.N(rgba.Alpha(rgba.White, 0x05))}.Add(gtx.Ops)
							paint.PaintOp{}.Add(gtx.Ops)
							return layout.Dimensions{Size: gtx.Constraints.Max}
						},
						)
					}

					// Objectives Header.
					{
						x += 15

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["header"],
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							l := material.Label(
								g.normal,
								g.normal.TextSize.Scale(18.0/16.0),
								"Objectives\t\t\t  1\t\t\t\t 2 \t\t\t  3",
							)
							l.Font.Weight = 200
							l.Color = rgba.N(rgba.White)
							return l.Layout(gtx)
						},
						)
					}

					// Objectives Section Line 1.
					{
						x += 25

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["line1"],
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X-int(offsets["line1"].V), 1)}.Push(gtx.Ops).Pop()
							paint.ColorOp{Color: rgba.N(rgba.Alpha(rgba.White, 0x5F))}.Add(gtx.Ops)
							paint.PaintOp{}.Add(gtx.Ops)
							return layout.Dimensions{Size: gtx.Constraints.Max}
						},
						)
					}

					// Regieleki.
					{
						x += 5

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["regi-icon"],
						}.Layout(gtx, (&screen.Screen{
							BorderColor: colors["regi-icon"],
							Border:      true,
							ScaleX:      5,
							ScaleY:      5,
							Image:       images["regieleki"],
						}).Layout)

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["regi-label"],
						}.Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								l := material.Label(
									g.normal,
									g.normal.TextSize.Scale(15.0/16.0),
									"Regieleki",
								)
								l.Color = rgba.N(rgba.White)

								return l.Layout(gtx)
							},
						)

						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi1"].V + offsets["regi-orange"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["orange-regieleki-1"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi1"].V + offsets["regi-purple"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["purple-regieleki-1"].Layout(gtx)
							},
						)

						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi2"].V + offsets["regi-orange"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["orange-regieleki-2"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi2"].V + offsets["regi-purple"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["purple-regieleki-2"].Layout(gtx)
							},
						)

						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi3"].V + offsets["regi-orange"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["orange-regieleki-3"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi3"].V + offsets["regi-purple"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["purple-regieleki-3"].Layout(gtx)
							},
						)
					}

					// Registeel.
					{
						x += 25

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["regi-icon"],
						}.Layout(gtx, (&screen.Screen{
							BorderColor: colors["regi-icon"],
							Border:      true,
							ScaleX:      5,
							ScaleY:      5,
							Image:       images["registeel"],
						}).Layout)

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["regi-label"],
						}.Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								l := material.Label(
									g.normal,
									g.normal.TextSize.Scale(15.0/16.0),
									"Registeel",
								)
								l.Color = rgba.N(rgba.White)

								return l.Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi1"].V + offsets["regi-orange"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["orange-registeel-1"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi1"].V + offsets["regi-purple"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["purple-registeel-1"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi2"].V + offsets["regi-orange"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["orange-registeel-2"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi2"].V + offsets["regi-purple"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["purple-registeel-2"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi3"].V + offsets["regi-orange"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["orange-registeel-3"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi3"].V + offsets["regi-purple"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["purple-registeel-3"].Layout(gtx)
							},
						)
					}

					// Regirock.
					{
						x += 25

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["regi-icon"],
						}.Layout(gtx, (&screen.Screen{
							BorderColor: colors["regi-icon"],
							Border:      true,
							ScaleX:      5,
							ScaleY:      5,
							Image:       images["regirock"],
						}).Layout)

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["regi-label"],
						}.Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								l := material.Label(
									g.normal,
									g.normal.TextSize.Scale(15.0/16.0),
									"Regirock",
								)
								l.Color = rgba.N(rgba.White)

								return l.Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi1"].V + offsets["regi-orange"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["orange-regirock-1"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi1"].V + offsets["regi-purple"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["purple-regirock-1"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi2"].V + offsets["regi-orange"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["orange-regirock-2"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi2"].V + offsets["regi-purple"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["purple-regirock-2"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi3"].V + offsets["regi-orange"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["orange-regirock-3"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi3"].V + offsets["regi-purple"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["purple-regirock-3"].Layout(gtx)
							},
						)
					}

					// Regice.
					{
						x += 25

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["regi-icon"],
						}.Layout(gtx, (&screen.Screen{
							BorderColor: colors["regi-icon"],
							Border:      true,
							ScaleX:      5,
							ScaleY:      5,
							Image:       images["regice"],
						}).Layout)

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["regi-label"],
						}.Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								l := material.Label(
									g.normal,
									g.normal.TextSize.Scale(15.0/16.0),
									"Regice",
								)
								l.Color = rgba.N(rgba.White)

								return l.Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi1"].V + offsets["regi-orange"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["orange-regice-1"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi1"].V + offsets["regi-purple"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["purple-regice-1"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi2"].V + offsets["regi-orange"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["orange-regice-2"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi2"].V + offsets["regi-purple"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["purple-regice-2"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi3"].V + offsets["regi-orange"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["orange-regice-3"].Layout(gtx)
							},
						)
						layout.Inset{
							Top:  unit.Px(x + 2),
							Left: unit.Px(offsets["regi3"].V + offsets["regi-purple"].V),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return regis["purple-regice-3"].Layout(gtx)
							},
						)
					}

					// Objectives Section Line 2.
					{
						x += 30

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["line2"],
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X-int(offsets["line2"].V), 1)}.Push(gtx.Ops).Pop()
							paint.ColorOp{Color: rgba.N(rgba.Alpha(rgba.White, 0x05))}.Add(gtx.Ops)
							paint.PaintOp{}.Add(gtx.Ops)
							return layout.Dimensions{Size: gtx.Constraints.Max}
						},
						)
					}

					// Matching Header.
					{
						x += 15

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["header"],
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							l := material.Label(
								g.normal,
								g.normal.TextSize.Scale(18.0/16.0),
								"Matching",
							)
							l.Font.Weight = 200
							l.Color = rgba.N(rgba.White)
							return l.Layout(gtx)
						},
						)
					}

					// Matching Section Line 1.
					{
						x += 25

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["line1"],
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X-int(offsets["line1"].V), 1)}.Push(gtx.Ops).Pop()
							paint.ColorOp{Color: rgba.N(rgba.Alpha(rgba.White, 0x5F))}.Add(gtx.Ops)
							paint.PaintOp{}.Add(gtx.Ops)
							return layout.Dimensions{Size: gtx.Constraints.Max}
						},
						)
					}

					// Matching.
					{
						x += 5

						layout.Inset{
							Left: unit.Px(5),
							Top:  unit.Px(x),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return matching.Layout(gtx, g.normal)
							},
						)
					}

					// Matching Section Line 2.
					{
						x += 135

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["line2"],
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X-int(offsets["line2"].V), 1)}.Push(gtx.Ops).Pop()
							paint.ColorOp{Color: rgba.N(rgba.Alpha(rgba.White, 0x05))}.Add(gtx.Ops)
							paint.PaintOp{}.Add(gtx.Ops)
							return layout.Dimensions{Size: gtx.Constraints.Max}
						},
						)
					}

					// Game Header.
					{
						x += 15

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["header"],
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							l := material.Label(
								g.normal,
								g.normal.TextSize.Scale(18.0/16.0),
								"Game",
							)
							l.Font.Weight = 200
							l.Color = rgba.N(rgba.White)
							return l.Layout(gtx)
						},
						)
					}

					// Game Section Line 1.
					{
						x += 25

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["line1"],
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X-int(offsets["line1"].V), 1)}.Push(gtx.Ops).Pop()
							paint.ColorOp{Color: rgba.N(rgba.Alpha(rgba.White, 0x5F))}.Add(gtx.Ops)
							paint.PaintOp{}.Add(gtx.Ops)
							return layout.Dimensions{Size: gtx.Constraints.Max}
						},
						)
					}

					// Game.
					{
						x += 5

						layout.Inset{
							Top:   unit.Px(x),
							Left:  unit.Px(20),
							Right: unit.Px(20),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return controls["start-stop"].Layout(gtx)
							},
						)

						layout.Inset{
							Top:   unit.Px(x),
							Left:  unit.Px(80),
							Right: unit.Px(20),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return controls["clear"].Layout(gtx)
							},
						)
					}

					// Game Section Line 2.
					{
						x += 30

						layout.Inset{
							Top:  unit.Px(x),
							Left: offsets["line2"],
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X-int(offsets["line2"].V), 1)}.Push(gtx.Ops).Pop()
							paint.ColorOp{Color: rgba.N(rgba.Alpha(rgba.White, 0x05))}.Add(gtx.Ops)
							paint.PaintOp{}.Add(gtx.Ops)
							return layout.Dimensions{Size: gtx.Constraints.Max}
						},
						)
					}
				}

				e.(system.FrameEvent).Frame(gtx.Ops)
			case system.StageEvent:
				w.Raise()
			case system.DestroyEvent:
				return
			}
		}
	}()
}

func bottomObjectiveButton(t *team.Team, name string, n int) *button.CircleButton {
	b := &button.CircleButton{
		Name:        t.Name,
		BorderColor: rgba.N(t.RGBA),
		Released:    colors["released"],
		Pressed:     colors["pressed"],
		Size:        image.Pt(15, 15),
		BorderWidth: unit.Sp(.5),
	}

	b.Click = func() {
		defer b.Deactivate()

		server.SetBottomObjective(t, name, n-1)
	}

	return b
}

func clearButton() *button.Button {
	b := &button.Button{
		Text:           "Clear",
		Released:       colors["disabled"],
		Pressed:        colors["pressed"],
		Size:           image.Pt(50, 25),
		TextSize:       unit.Sp(12),
		TextOffsetTop:  0,
		TextOffsetLeft: -2,
		BorderWidth:    unit.Sp(.5),
	}

	b.Click = func() {
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
	off := -1
	switch n {
	case 10, 50:
		off = -4
	case 100:
		off = -8
	}

	b := &button.Button{
		Text:           fmt.Sprintf("+%d", n),
		Released:       colors["released"],
		Pressed:        colors["pressed"],
		Size:           image.Pt(35, 16),
		TextSize:       unit.Sp(12),
		TextOffsetTop:  -4,
		TextOffsetLeft: float32(off),
		BorderWidth:    unit.Sp(.5),
	}

	b.Click = func() {
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
		Text:           "+",
		Released:       colors["enabled"],
		Pressed:        colors["pressed"],
		Size:           image.Pt(40, 16),
		TextSize:       unit.Sp(14),
		TextOffsetTop:  -5,
		TextOffsetLeft: 5,
		BorderWidth:    unit.Sp(.5),
	}

	b.Click = func() {
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
		BorderColor: rgba.N(t.RGBA),
		Released:    colors["released"],
		Pressed:     colors["pressed"],
		Size:        image.Pt(15, 15),
		BorderWidth: unit.Sp(.5),
	}

	b.Click = func() {
		defer b.Deactivate()

		server.SetRegielekiAt(t, n-1)
	}

	return b
}

func startStopButton() *button.Button {
	b := &button.Button{
		Text:           startLabel,
		Released:       colors["enabled"],
		Pressed:        colors["pressed"],
		Size:           image.Pt(50, 25),
		TextSize:       unit.Sp(12),
		TextOffsetTop:  0,
		TextOffsetLeft: -1,
		BorderWidth:    unit.Sp(.5),
	}

	b.Click = func() {
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

	for _, i := range matching.Items {
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
