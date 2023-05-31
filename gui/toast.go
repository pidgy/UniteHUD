package gui

import (
	"fmt"
	"image"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/audio"
	"github.com/pidgy/unitehud/gui/visual/area"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/dropdown"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/nrgba"
)

func (g *GUI) ToastAudioInputOutput(q string, callback func(capture, playback string)) {
	if g.toastActive {
		notify.Warn("Failed to get input, please close other toast windows")
		return
	}

	g.toastActive = true
	defer func() { g.toastActive = false }()

	dx, dy := float32(720), float32(482)

	w := app.NewWindow(
		app.Title("UniteHUD "+q),
		app.Size(unit.Dp(dx), unit.Dp(dy)),
		app.MinSize(unit.Dp(dx), unit.Dp(dy)),
		app.MaxSize(unit.Dp(dx), unit.Dp(dy)),
	)

	var ops op.Ops

	cap, play := "", ""

	capList := &dropdown.List{
		WidthModifier: 1,
		Items: []*dropdown.Item{
			{
				Text:    audio.DeviceDefault,
				Checked: widget.Bool{Value: true},
				Callback: func(i *dropdown.Item) {
					cap = audio.DeviceDefault
					i.Checked.Value = true
				},
			},
		},
		Callback: func(i *dropdown.Item, _ *dropdown.List) {
		},
	}

	playList := &dropdown.List{
		WidthModifier: 1,
		Items: []*dropdown.Item{
			{
				Text:    audio.DeviceDefault,
				Checked: widget.Bool{Value: true},
				Callback: func(i *dropdown.Item) {
					play = audio.DeviceDefault
					i.Checked.Value = true
				},
			},
		},
		Callback: func(i *dropdown.Item, _ *dropdown.List) {
		},
	}

	caps, plays := audio.DeviceNames()

	for _, name := range caps {
		capList.Items = append(capList.Items, &dropdown.Item{
			Text: name,
			Callback: func(i *dropdown.Item) {
				play = i.Text
				i.Checked.Value = true
			},
		})
	}

	for _, name := range plays {
		playList.Items = append(playList.Items, &dropdown.Item{
			Text: name,
			Callback: func(i *dropdown.Item) {
				play = i.Text
				i.Checked.Value = true
			},
		})
	}

	okButton := &button.Button{
		Text:        "OK",
		Released:    nrgba.Gray,
		Pressed:     nrgba.Transparent30,
		BorderWidth: unit.Sp(1.5),
		Click: func(b *button.Button) {
			defer b.Deactivate()
			defer w.Perform(system.ActionClose)

			callback(cap, play)
		},
	}

	for e := range w.Events() {
		switch e := e.(type) {
		case system.DestroyEvent:
			callback("", "")
			return
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)

			ops.Reset()

			colorBox(gtx, gtx.Constraints.Max, nrgba.DarkGray)

			layout.Flex{
				Axis: layout.Vertical,
			}.Layout(gtx,
				layout.Flexed(0.2, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{
						Axis: layout.Horizontal,
					}.Layout(gtx,
						layout.Flexed(0.35, func(gtx layout.Context) layout.Dimensions {
							label := material.Label(g.normal, unit.Sp(18), "Audio In (Capture)")
							label.Color = nrgba.Highlight.Color()
							label.Font.Weight = 200

							return layout.N.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return label.Layout(gtx)
								})
							})
						}),
						layout.Flexed(0.35, func(gtx layout.Context) layout.Dimensions {
							label := material.Label(g.normal, unit.Sp(18), "Audio Out (Playback)")
							label.Color = nrgba.Highlight.Color()
							label.Font.Weight = 200

							return layout.N.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return label.Layout(gtx)
								})
							})
						}),
						layout.Flexed(0.2, func(gtx layout.Context) layout.Dimensions {
							return layout.Dimensions{Size: gtx.Constraints.Max}
						}),
					)
				}),
				layout.Flexed(0.8, func(gtx layout.Context) layout.Dimensions {
					layout.Flex{
						Axis: layout.Horizontal,
					}.Layout(gtx,
						layout.Flexed(0.35, func(gtx layout.Context) layout.Dimensions {
							return capList.Layout(gtx, g.normal)
						}),
						layout.Flexed(0.35, func(gtx layout.Context) layout.Dimensions {
							return playList.Layout(gtx, g.normal)
						}),
						layout.Flexed(0.2, func(gtx layout.Context) layout.Dimensions {
							return layout.S.Layout(gtx,
								func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Bottom: unit.Dp(25)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return okButton.Layout(gtx)
									})
								},
							)
						}),
					)
					return layout.Dimensions{Size: gtx.Constraints.Max}
				}),
			)

			w.Perform(system.ActionCenter)
			w.Perform(system.ActionRaise)
			w.Invalidate()

			e.Frame(gtx.Ops)
		}
	}
}

func (g *GUI) ToastCapture(captures []*area.Capture) error {
	if g.toastActive {
		return fmt.Errorf("Failed to get input, please close other toast windows")
	}

	go func() {
		g.toastActive = true
		defer func() { g.toastActive = false }()

		dx, dy := float32(300), float32(200)

		w := app.NewWindow(
			app.Title("UniteHUD Capture Areas"),
			app.Size(unit.Dp(dx), unit.Dp(dy)),
			app.MinSize(unit.Dp(dx), unit.Dp(dy)),
			app.MaxSize(unit.Dp(dx), unit.Dp(dy)),
		)

		checks := make([]material.CheckBoxStyle, len(captures))
		for i, c := range captures {
			checks[i] = material.CheckBox(g.cascadia, &widget.Bool{}, c.Option)
			checks[i].Font.Weight = font.Weight(500)
			checks[i].Color = nrgba.White.Color()
			checks[i].Size = unit.Dp(20)
			checks[i].IconColor = nrgba.White.Color()
			checks[i].TextSize = unit.Sp(12)
		}

		all := material.CheckBox(g.cascadia, &widget.Bool{}, "Select All")
		all.Font.Weight = font.Weight(500)
		all.Color = nrgba.White.Color()
		all.Size = unit.Dp(20)
		all.IconColor = nrgba.White.Color()
		all.TextSize = unit.Sp(12)

		okButton := &button.Button{
			Text:        "Capture",
			Released:    nrgba.Gray,
			Pressed:     nrgba.Transparent30,
			BorderWidth: unit.Sp(1.5),
			Click: func(b *button.Button) {
				defer b.Deactivate()
				defer w.Perform(system.ActionClose)

				for i, check := range checks {
					if check.CheckBox.Value {
						err := captures[i].Open()
						if err != nil {
							g.ToastErrorForce(err)
							return
						}
					}
				}
			},
		}

		var ops op.Ops

		for e := range w.Events() {
			switch e := e.(type) {
			case system.DestroyEvent:
				return
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)

				ops.Reset()

				colorBox(gtx, gtx.Constraints.Max, nrgba.DarkGray)

				if all.CheckBox.Changed() {
					for _, check := range checks {
						check.CheckBox.Value = all.CheckBox.Value
					}
				}

				layout.Inset{
					Top: unit.Dp(10),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						disabled := true

						for i, check := range checks {
							if check.CheckBox.Value {
								disabled = false
							}
							layout.Inset{
								Top:  unit.Dp(float32((15 * i) + (5 * i) + 1)),
								Left: unit.Dp(10),
							}.Layout(gtx,
								func(gtx layout.Context) layout.Dimensions {
									return layout.N.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										dim := check.Layout(gtx)
										dim.Size.X = gtx.Constraints.Max.X
										return dim
									})
								},
							)
						}

						okButton.Disabled = disabled
						okButton.Active = disabled

						return layout.Dimensions{Size: gtx.Constraints.Max}
					},
				)

				layout.Inset{
					Top:  unit.Dp(float32(gtx.Constraints.Max.Y - 40)),
					Left: unit.Dp(10),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return layout.S.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							dim := all.Layout(gtx)
							dim.Size.X = gtx.Constraints.Max.X
							return dim
						})
					},
				)

				layout.Inset{
					Top:  unit.Dp(float32(gtx.Constraints.Max.Y - 40)),
					Left: unit.Dp(float32(gtx.Constraints.Max.X - 110)),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return okButton.Layout(gtx)
					},
				)

				w.Perform(system.ActionCenter)
				w.Perform(system.ActionRaise)
				w.Invalidate()

				e.Frame(gtx.Ops)
			}
		}
	}()

	return nil
}

func (g *GUI) ToastCrash(msg, reason string, callbacks ...func()) {
	go func() {
		g.toastActive = true
		defer func() { g.toastActive = false }()

		dx, dy := float32(500), float32(125)

		w := app.NewWindow(
			app.Title("UniteHUD Crash Report"),
			app.Size(unit.Dp(dx), unit.Dp(dy)),
			app.MaxSize(unit.Dp(dx), unit.Dp(dy)),
			app.MinSize(unit.Dp(dx), unit.Dp(dy)),
		)

		// Scale.
		m := material.Label(g.normal, g.normal.TextSize, msg)
		m.Color = nrgba.White.Color()
		m.Alignment = text.Middle

		c := material.Label(g.normal, g.normal.TextSize, reason)
		c.Color = nrgba.PaleRed.Color()
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

				colorBox(gtx, gtx.Constraints.Max, nrgba.DarkGray)

				layout.Inset{
					Top: unit.Dp(15),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return m.Layout(gtx)
					},
				)

				layout.Inset{
					Top: unit.Dp(40),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return c.Layout(gtx)
					},
				)

				w.Perform(system.ActionCenter)
				w.Perform(system.ActionRaise)
				w.Invalidate()

				e.Frame(gtx.Ops)
			}
		}
	}()
}

func (g *GUI) ToastError(err error) {
	if (g.lastToastError != nil && err.Error() == g.lastToastError.Error()) && time.Since(g.lastToastTime) < time.Second*10 {
		return
	}

	g.lastToastError = err
	g.lastToastTime = time.Now()

	e := err.Error()
	es := strings.Split(e, " ")
	es[0] = strings.Title(es[0])

	g.ToastOK("UniteHUD Error", strings.Join(es, " "))
}

func (g *GUI) ToastErrorf(format string, a ...interface{}) {
	g.ToastError(fmt.Errorf(format, a...))
}

func (g *GUI) ToastErrorForce(err error) {
	g.lastToastError = err
	g.lastToastTime = time.Now()

	e := err.Error()
	es := strings.Split(e, " ")
	es[0] = strings.Title(es[0])

	g.toastActive = false

	g.ToastOK("UniteHUD Error", strings.Join(es, " "))
}

func (g *GUI) ToastInput(q, hint, option string, callback func(text string, option bool)) error {
	if g.toastActive {
		return fmt.Errorf("Failed to get input, please close other toast windows")
	}

	go func() {
		g.toastActive = true
		defer func() { g.toastActive = false }()

		dx, dy := float32(400), float32(100)

		w := app.NewWindow(
			app.Title("UniteHUD "+q),
			app.Size(unit.Dp(dx), unit.Dp(dy)),
			app.MinSize(unit.Dp(dx), unit.Dp(dy)),
			app.MaxSize(unit.Dp(dx), unit.Dp(dy)),
		)

		checked := widget.Bool{}
		check := material.CheckBox(g.cascadia, &checked, option)
		check.Font.Weight = font.Weight(500)
		check.Color = nrgba.White.Color()
		check.Size = unit.Dp(16)
		check.IconColor = nrgba.White.Color()
		check.TextSize = unit.Sp(13)

		input := &widget.Editor{
			Alignment:  text.Start,
			SingleLine: true,
			Submit:     true,
			InputHint:  key.HintURL,
		}
		input.SetCaret(0, 0)

		okButton := &button.Button{
			Text:        "OK",
			Released:    nrgba.Gray,
			Pressed:     nrgba.Transparent30,
			BorderWidth: unit.Sp(1.5),
			Click: func(b *button.Button) {
				defer b.Deactivate()
				defer w.Perform(system.ActionClose)

				if callback != nil {
					text := input.Text()
					if text == "" {
						text = hint
					}
					callback(text, checked.Value)
				}
			},
		}

		var ops op.Ops

		for e := range w.Events() {
			switch e := e.(type) {
			case system.DestroyEvent:
				return
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)

				ops.Reset()

				colorBox(gtx, gtx.Constraints.Max, nrgba.DarkGray)

				layout.Inset{
					Top:    unit.Dp(10),
					Left:   unit.Dp(10),
					Right:  unit.Dp(15),
					Bottom: unit.Dp(float32(gtx.Constraints.Max.Y / 2)),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						defer clip.RRect{SE: 3, SW: 3, NE: 3, NW: 3, Rect: image.Rectangle{
							Max: image.Pt(gtx.Constraints.Max.X+5, 25),
						}}.Push(gtx.Ops).Pop()

						paint.ColorOp{Color: nrgba.White.Color()}.Add(gtx.Ops)
						paint.PaintOp{}.Add(gtx.Ops)

						e := material.Editor(g.cascadia, input, hint)
						e.Color = nrgba.Black.Color()
						e.HintColor = nrgba.Gray.Alpha(200).Color()
						e.TextSize = unit.Sp(16)
						return layout.Inset{Left: unit.Dp(2), Right: unit.Dp(2), Top: unit.Dp(2)}.Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								return e.Layout(gtx)
							},
						)
					},
				)

				layout.Inset{
					Top:    unit.Dp(20),
					Left:   unit.Dp(10),
					Bottom: unit.Dp(40),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return layout.S.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							dim := check.Layout(gtx)
							dim.Size.X = gtx.Constraints.Max.X
							return dim
						})
					},
				)

				layout.Inset{
					Top:  unit.Dp(float32(gtx.Constraints.Max.Y - 40)),
					Left: unit.Dp(float32(gtx.Constraints.Max.X - 110)),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return okButton.Layout(gtx)
					},
				)

				w.Perform(system.ActionCenter)
				w.Perform(system.ActionRaise)
				w.Invalidate()

				e.Frame(gtx.Ops)
			}
		}
	}()

	return nil
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
			app.Title("UniteHUD "+title),
			app.Size(unit.Dp(dx), unit.Dp(dy)),
			app.MaxSize(unit.Dp(dx), unit.Dp(dy)),
			app.MinSize(unit.Dp(dx), unit.Dp(dy)),
		)

		// Scale.
		m := material.Label(g.normal, g.normal.TextSize, msg)
		m.Color = nrgba.White.Color()
		m.Alignment = text.Middle

		okButton := &button.Button{
			Text:        "OK",
			Released:    nrgba.Gray,
			Pressed:     nrgba.Transparent30,
			BorderWidth: unit.Sp(1.5),
			Click: func(b *button.Button) {
				defer b.Deactivate()

				w.Perform(system.ActionClose)
			},
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

				colorBox(gtx, gtx.Constraints.Max, nrgba.DarkGray)

				layout.Inset{
					Top: unit.Dp(15),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return m.Layout(gtx)
					},
				)

				layout.Inset{
					Top:  unit.Dp(float32(gtx.Constraints.Max.Y - 40)),
					Left: unit.Dp(float32(gtx.Constraints.Max.X - 110)),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return okButton.Layout(gtx)
					},
				)

				w.Perform(system.ActionCenter)
				w.Perform(system.ActionRaise)
				w.Invalidate()

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
			app.Title("UniteHUD "+title),
			app.Size(
				unit.Dp(400),
				unit.Dp(100),
			),
		)

		// Scale 16.
		m := material.Label(g.normal, g.normal.TextSize, msg)
		m.Color = nrgba.White.Color()
		m.Alignment = text.Middle

		yButton := &button.Button{
			Text:        "Yes",
			Released:    nrgba.Gray,
			Pressed:     nrgba.Transparent30,
			BorderWidth: unit.Sp(1.5),
			Click: func(b *button.Button) {
				destroyed = false
				if y != nil {
					y()
				}
				w.Perform(system.ActionClose)
			},
		}

		nButton := &button.Button{
			Text:        "No",
			Released:    nrgba.Gray,
			Pressed:     nrgba.Transparent30,
			BorderWidth: unit.Sp(1.5),
			Click: func(b *button.Button) {
				destroyed = false
				if n != nil {
					n()
				}
				w.Perform(system.ActionClose)
			},
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

				colorBox(gtx, gtx.Constraints.Max, nrgba.DarkGray)

				layout.Inset{
					Top: unit.Dp(15),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return m.Layout(gtx)
					},
				)

				layout.Inset{
					Left: unit.Dp(float32(gtx.Constraints.Max.X/2 - 115)),
					Top:  unit.Dp(float32(gtx.Constraints.Max.Y/2) + 5),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return yButton.Layout(gtx)
					},
				)

				layout.Inset{
					Left: unit.Dp(float32(gtx.Constraints.Max.X/2 + 15)),
					Top:  unit.Dp(float32(gtx.Constraints.Max.Y/2) + 5),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return nButton.Layout(gtx)
					},
				)

				w.Perform(system.ActionCenter)
				w.Perform(system.ActionRaise)
				w.Invalidate()

				e.Frame(gtx.Ops)
			}
		}
	}()
}
