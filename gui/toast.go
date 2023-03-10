package gui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
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

	"github.com/pidgy/unitehud/gui/visual/area"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/rgba"
)

func (g *GUI) ToastCapture(captures []*area.Capture) error {
	if g.toastActive {
		return fmt.Errorf("Failed to get input, please close other toast windows")
	}

	go func() {
		g.toastActive = true
		defer func() { g.toastActive = false }()

		dx, dy := float32(300), float32(200)

		w := app.NewWindow(
			app.Title("Capture Areas"),
			app.Size(unit.Px(dx), unit.Px(dy)),
			app.MinSize(unit.Px(dx), unit.Px(dy)),
			app.MaxSize(unit.Px(dx), unit.Px(dy)),
		)

		checks := make([]material.CheckBoxStyle, len(captures))
		for i, c := range captures {
			checks[i] = material.CheckBox(g.cascadia, &widget.Bool{}, c.Option)
			checks[i].Font.Weight = text.Weight(500)
			checks[i].Color = color.NRGBA(rgba.White)
			checks[i].Size = unit.Px(20)
			checks[i].IconColor = rgba.N(rgba.White)
			checks[i].TextSize = unit.Px(17)
		}

		all := material.CheckBox(g.cascadia, &widget.Bool{}, "Select All")
		all.Font.Weight = text.Weight(500)
		all.Color = color.NRGBA(rgba.White)
		all.Size = unit.Px(20)
		all.IconColor = rgba.N(rgba.White)
		all.TextSize = unit.Px(17)

		okButton := &button.Button{
			Text:     "  Capture",
			Released: rgba.N(rgba.Gray),
			Pressed:  rgba.N(rgba.DarkGray),
			Click: func(b *button.Button) {
				defer b.Deactivate()
				defer w.Close()

				for i, check := range checks {
					if check.CheckBox.Value {
						err := captures[i].Open()
						if err != nil {
							g.ToastError(err)
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

				colorBox(gtx, gtx.Constraints.Max, color.NRGBA{R: 25, G: 25, B: 25, A: 255})

				if all.CheckBox.Changed() {
					for _, check := range checks {
						check.CheckBox.Value = all.CheckBox.Value
					}
				}
				layout.Inset{
					Top: unit.Px(10),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						disabled := true

						for i, check := range checks {
							if check.CheckBox.Value {
								disabled = false
							}
							layout.Inset{
								Top:  unit.Px(float32((15 * i) + (5 * i) + 1)),
								Left: unit.Px(10),
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
					})

				layout.Inset{
					Top:  unit.Px(float32(gtx.Constraints.Max.Y - 40)),
					Left: unit.Px(10),
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
					Top:  unit.Px(float32(gtx.Constraints.Max.Y - 40)),
					Left: unit.Px(float32(gtx.Constraints.Max.X - 110)),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return okButton.Layout(gtx)
					},
				)

				w.Center()
				w.Raise()
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

				colorBox(gtx, gtx.Constraints.Max, color.NRGBA{R: 25, G: 25, B: 25, A: 255})

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

	g.ToastOK("Errorr", strings.Join(es, " "))
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

	g.ToastOK("Error", strings.Join(es, " "))
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
			app.Title(q),
			app.Size(unit.Px(dx), unit.Px(dy)),
			app.MinSize(unit.Px(dx), unit.Px(dy)),
			app.MaxSize(unit.Px(dx), unit.Px(dy)),
		)

		checked := widget.Bool{}
		check := material.CheckBox(g.cascadia, &checked, option)
		check.Font.Weight = text.Weight(500)
		check.Color = color.NRGBA(rgba.White)
		check.Size = unit.Px(16)
		check.IconColor = rgba.N(rgba.White)
		check.TextSize = unit.Px(13)

		input := &widget.Editor{
			Alignment:  text.Start,
			SingleLine: true,
			Submit:     true,
			InputHint:  key.HintURL,
		}
		input.SetCaret(0, 0)

		okButton := &button.Button{
			Text:     "\t    OK",
			Released: rgba.N(rgba.Gray),
			Pressed:  rgba.N(rgba.DarkGray),
			Click: func(b *button.Button) {
				defer b.Deactivate()
				defer w.Close()

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

				colorBox(gtx, gtx.Constraints.Max, color.NRGBA{R: 25, G: 25, B: 25, A: 255})

				layout.Inset{
					Top:    unit.Px(10),
					Left:   unit.Px(10),
					Right:  unit.Px(15),
					Bottom: unit.Px(float32(gtx.Constraints.Max.Y / 2)),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						defer clip.RRect{SE: 3, SW: 3, NE: 3, NW: 3, Rect: f32.
							Rectangle{
							Max: f32.Pt(
								float32(gtx.Constraints.Max.X+5),
								float32(25),
							),
						}}.Push(gtx.Ops).Pop()

						paint.ColorOp{Color: rgba.N(rgba.White)}.Add(gtx.Ops)
						paint.PaintOp{}.Add(gtx.Ops)

						e := material.Editor(g.cascadia, input, hint)
						e.Color = rgba.N(rgba.Black)
						e.HintColor = rgba.N(rgba.Alpha(rgba.Gray, 200))
						e.TextSize = unit.Px(16)
						return layout.Inset{Left: unit.Px(2), Right: unit.Px(2), Top: unit.Px(2)}.Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								return e.Layout(gtx)
							},
						)
					},
				)

				layout.Inset{
					Top:    unit.Px(20),
					Left:   unit.Px(10),
					Bottom: unit.Px(40),
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
					Top:  unit.Px(float32(gtx.Constraints.Max.Y - 40)),
					Left: unit.Px(float32(gtx.Constraints.Max.X - 110)),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return okButton.Layout(gtx)
					},
				)

				w.Center()
				w.Raise()
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
			Released: rgba.N(rgba.Gray),
			Pressed:  rgba.N(rgba.DarkGray),
			Click: func(b *button.Button) {
				defer b.Deactivate()

				w.Close()
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

				colorBox(gtx, gtx.Constraints.Max, color.NRGBA{R: 25, G: 25, B: 25, A: 255})

				layout.Inset{
					Top: unit.Px(15),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return m.Layout(gtx)
					},
				)

				layout.Inset{
					Top:  unit.Px(float32(gtx.Constraints.Max.Y - 40)),
					Left: unit.Px(float32(gtx.Constraints.Max.X - 110)),
				}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return okButton.Layout(gtx)
					},
				)

				w.Center()
				w.Raise()
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
			Released: rgba.N(rgba.Gray),
			Pressed:  rgba.N(rgba.DarkGray),
			Click: func(b *button.Button) {
				destroyed = false
				if y != nil {
					y()
				}
				w.Close()
			},
		}

		nButton := &button.Button{
			Text:     "\t    No",
			Released: rgba.N(rgba.Gray),
			Pressed:  rgba.N(rgba.DarkGray),
			Click: func(b *button.Button) {
				destroyed = false
				if n != nil {
					n()
				}
				w.Close()
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

				colorBox(gtx, gtx.Constraints.Max, color.NRGBA{R: 25, G: 25, B: 25, A: 255})

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
				w.Invalidate()

				e.Frame(gtx.Ops)
			}
		}
	}()
}
