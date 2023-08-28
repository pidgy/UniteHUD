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

	"github.com/pidgy/unitehud/fonts"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/nrgba"
)

const (
	toastTextSize = unit.Sp(16)
)

func (g *GUI) ToastCrash(reason string, closed, logs func()) {
	g.toastActive = true
	defer func() { g.toastActive = false }()

	width, height := float32(500), float32(125)

	w := app.NewWindow(
		app.Title("Crashed"),
		app.Size(unit.Dp(width), unit.Dp(height)),
		app.MaxSize(unit.Dp(width), unit.Dp(height)),
		app.MinSize(unit.Dp(width), unit.Dp(height)),
		app.Decorated(false),
	)

	bar := title.New(fmt.Sprintf("%s Crash", title.Default), fonts.NewCollection(), nil, nil, func() {
		w.Perform(system.ActionClose)
	})
	bar.NoTip = true

	c := material.Label(bar.Collection.Calibri().Theme, toastTextSize, reason)
	c.Color = nrgba.PastelRed.Color()
	c.Alignment = text.Middle

	btn := &button.Button{
		Text:            "View Logs",
		TextSize:        unit.Sp(16),
		Font:            bar.Collection.Calibri(),
		Pressed:         nrgba.Transparent30,
		Released:        nrgba.DarkGray,
		BorderWidth:     unit.Sp(0),
		NoBorder:        true,
		Size:            image.Pt(96, 32),
		TextInsetBottom: -2,

		Click: func(this *button.Button) {
			defer this.Deactivate()

			if logs != nil {
				logs()
			}

			w.Perform(system.ActionClose)
		},
	}

	var ops op.Ops

	for e := range w.Events() {
		switch e := e.(type) {
		case system.DestroyEvent:
			if closed != nil {
				closed()
			}
			return
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)

			bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				colorBox(gtx, gtx.Constraints.Max, nrgba.BackgroundAlt)

				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Flexed(.1, layout.Spacer{Height: 5}.Layout),

					layout.Flexed(.3, func(gtx layout.Context) layout.Dimensions {
						return c.Layout(gtx)
					}),

					layout.Flexed(.2, layout.Spacer{Height: 1}.Layout),

					layout.Flexed(.3, func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, btn.Layout)
					}),

					layout.Flexed(.1, layout.Spacer{Height: 5}.Layout),
				)
			})

			w.Perform(system.ActionCenter)
			w.Perform(system.ActionRaise)
			w.Invalidate()

			e.Frame(gtx.Ops)
		}
	}
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

	g.ToastOK("Error", strings.Join(es, " "))
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

	g.toastActive = true
	defer func() { g.toastActive = false }()

	width := unit.Dp(525)
	height := unit.Dp(150)

	w := app.NewWindow(
		app.Title(q),
		app.Size(width, height),
		app.MinSize(width, height),
		app.MaxSize(width, height),
		app.Decorated(false),
	)

	bar := title.New(q, fonts.NewCollection(), nil, nil, nil)
	bar.NoTip = true

	checked := widget.Bool{}
	check := material.CheckBox(bar.Collection.Calibri().Theme, &checked, option)
	check.Font.Weight = font.Weight(500)
	check.Color = nrgba.White.Color()
	check.Size = unit.Dp(20)
	check.IconColor = nrgba.White.Color()
	check.TextSize = unit.Sp(16)

	input := &widget.Editor{
		Alignment:  text.Start,
		SingleLine: true,
		Submit:     true,
		InputHint:  key.HintURL,
	}
	input.SetCaret(0, 0)

	editor := material.Editor(bar.Collection.Calibri().Theme, input, hint)
	editor.Color = nrgba.Black.Color()
	editor.HintColor = nrgba.Gray.Alpha(200).Color()
	editor.TextSize = unit.Sp(16)

	okButton := &button.Button{
		Text:        "OK",
		Font:        bar.Collection.Calibri(),
		Released:    nrgba.Gray,
		Pressed:     nrgba.Transparent30,
		BorderWidth: unit.Sp(1.5),
		Click: func(this *button.Button) {
			defer this.Deactivate()

			if callback != nil {
				text := input.Text()
				if text == "" {
					text = hint
				}
				callback(text, checked.Value)
			}

			w.Perform(system.ActionClose)
		},
	}

	cancelButton := &button.Button{
		Text:        "Cancel",
		Font:        bar.Collection.Calibri(),
		Released:    nrgba.Gray,
		Pressed:     nrgba.Transparent30,
		BorderWidth: unit.Sp(1.5),
		Click: func(this *button.Button) {
			defer this.Deactivate()

			w.Perform(system.ActionClose)
		},
	}

	var ops op.Ops

	for e := range w.Events() {
		switch e := e.(type) {
		case system.DestroyEvent:
			return nil
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)

			bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				colorBox(gtx, gtx.Constraints.Max, nrgba.DarkGray)

				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(layout.Spacer{Height: 5}.Layout),

					layout.Flexed(.5, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{
							Left:  unit.Dp(10),
							Right: unit.Dp(15),
						}.Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								defer clip.RRect{SE: 3, SW: 3, NE: 3, NW: 3, Rect: image.Rectangle{
									Max: image.Pt(gtx.Constraints.Max.X+5, 25),
								}}.Push(gtx.Ops).Pop()

								paint.ColorOp{Color: nrgba.White.Color()}.Add(gtx.Ops)
								paint.PaintOp{}.Add(gtx.Ops)

								return layout.Inset{Left: unit.Dp(2), Right: unit.Dp(2), Top: unit.Dp(2)}.Layout(gtx,
									func(gtx layout.Context) layout.Dimensions {
										return editor.Layout(gtx)
									},
								)
							},
						)
					}),

					layout.Rigid(layout.Spacer{Height: 2}.Layout),

					layout.Flexed(.5, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{
							Left: unit.Dp(10),
						}.Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								return layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									dim := check.Layout(gtx)
									return dim
								})
							},
						)
					}),

					layout.Rigid(layout.Spacer{Height: 2}.Layout),

					layout.Flexed(.5, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(
							gtx,
							layout.Rigid(layout.Spacer{Width: 10}.Layout),

							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Center.Layout(gtx, cancelButton.Layout)
							}),

							layout.Rigid(layout.Spacer{Width: 1}.Layout),

							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Center.Layout(gtx, okButton.Layout)
							}),
							layout.Rigid(layout.Spacer{Width: 10}.Layout),
						)
					}),

					layout.Rigid(layout.Spacer{Height: 5}.Layout),
				)
			})

			w.Perform(system.ActionCenter)
			w.Perform(system.ActionRaise)
			w.Invalidate()

			e.Frame(gtx.Ops)
		}
	}

	return nil
}

func (g *GUI) ToastOK(header, msg string, callbacks ...func()) {
	if g.toastActive {
		return
	}

	g.toastActive = true
	defer func() { g.toastActive = false }()

	width, height := float32(400), float32(125)

	w := app.NewWindow(
		app.Title(header),
		app.Size(unit.Dp(width), unit.Dp(height)),
		app.MaxSize(unit.Dp(width), unit.Dp(height)),
		app.MinSize(unit.Dp(width), unit.Dp(height)),
		app.Decorated(false),
	)

	bar := title.New(fmt.Sprintf("%s %s", title.Default, header), fonts.NewCollection(), nil, nil, func() {
		w.Perform(system.ActionClose)
	})
	bar.NoTip = true

	// Scale.
	m := material.Label(bar.Collection.Calibri().Theme, toastTextSize, msg)
	m.Color = nrgba.White.Color()
	m.Alignment = text.Middle

	okButton := &button.Button{
		Text:            "OK",
		TextSize:        unit.Sp(16),
		Font:            bar.Collection.Calibri(),
		Pressed:         nrgba.Transparent30,
		Released:        nrgba.DarkGray,
		BorderWidth:     unit.Sp(0),
		NoBorder:        true,
		Size:            image.Pt(96, 32),
		TextInsetBottom: -2,

		Click: func(this *button.Button) {
			defer this.Deactivate()

			for _, cb := range callbacks {
				cb()
			}

			w.Perform(system.ActionClose)
		},
	}

	var ops op.Ops

	for e := range w.Events() {
		if e, ok := e.(system.FrameEvent); ok {
			gtx := layout.NewContext(&ops, e)

			bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				colorBox(gtx, gtx.Constraints.Max, nrgba.BackgroundAlt)

				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(layout.Spacer{Height: 10}.Layout),

					layout.Flexed(.5, func(gtx layout.Context) layout.Dimensions {
						return m.Layout(gtx)
					}),

					layout.Rigid(layout.Spacer{Height: 2}.Layout),

					layout.Flexed(.5, func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, okButton.Layout)
					}),

					layout.Rigid(layout.Spacer{Height: 2}.Layout),
				)
			})

			w.Perform(system.ActionCenter)
			w.Perform(system.ActionRaise)
			w.Invalidate()
			e.Frame(gtx.Ops)
		}
	}
}

func (g *GUI) ToastYesNo(header, msg string, y, n func()) {
	if g.toastActive {
		return
	}

	g.toastActive = true
	defer func() { g.toastActive = false }()

	width, height := unit.Dp(400), unit.Dp(125)

	w := app.NewWindow(
		app.Title(header),
		app.Size(width, height),
		app.MaxSize(width, height),
		app.MinSize(width, height),
		app.Decorated(false),
	)

	bar := title.New(fmt.Sprintf("%s %s", title.Default, header), fonts.NewCollection(), nil, nil, func() {
		w.Perform(system.ActionClose)
	})
	bar.NoTip = true

	// Scale 16.
	m := material.Label(bar.Collection.Calibri().Theme, toastTextSize, msg)
	m.Color = nrgba.White.Color()
	m.Alignment = text.Middle

	yButton := &button.Button{
		Text:            "Yes",
		TextSize:        unit.Sp(16),
		Font:            bar.Collection.Calibri(),
		Pressed:         nrgba.Transparent30,
		Released:        nrgba.DarkGray,
		BorderWidth:     unit.Sp(0),
		Size:            image.Pt(96, 32),
		NoBorder:        true,
		TextInsetBottom: -2,
		Click: func(this *button.Button) {
			if y != nil {
				y()
			}
			w.Perform(system.ActionClose)
		},
	}

	nButton := &button.Button{
		Text:            "No",
		TextSize:        unit.Sp(16),
		Font:            bar.Collection.Calibri(),
		Pressed:         nrgba.Transparent30,
		Released:        nrgba.DarkGray,
		BorderWidth:     unit.Sp(0),
		NoBorder:        true,
		Size:            image.Pt(96, 32),
		TextInsetBottom: -2,
		Click: func(this *button.Button) {
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
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)

			bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				colorBox(gtx, gtx.Constraints.Max, nrgba.BackgroundAlt)

				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(layout.Spacer{Height: 10}.Layout),

					layout.Flexed(.5, func(gtx layout.Context) layout.Dimensions {
						return m.Layout(gtx)
					}),

					layout.Flexed(.5, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(
							gtx,
							layout.Rigid(layout.Spacer{Width: 5}.Layout),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Center.Layout(gtx, yButton.Layout)
							}),
							layout.Rigid(layout.Spacer{Width: 1}.Layout),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Center.Layout(gtx, nButton.Layout)
							}),
							layout.Rigid(layout.Spacer{Width: 5}.Layout),
						)
					}),

					layout.Rigid(layout.Spacer{Height: 2}.Layout),
				)
			})

			w.Perform(system.ActionCenter)
			w.Perform(system.ActionRaise)
			w.Invalidate()
			e.Frame(gtx.Ops)
		}
	}
}
