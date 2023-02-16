package gui

import (
	"image/color"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/rgba"
)

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
				w.Invalidate()

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
			Released: rgba.N(rgba.Gray),
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
		}
		yButton.Click = func() {
			destroyed = false
			y()

			w.Close()
		}

		nButton := &button.Button{
			Text:     "\t    No",
			Released: rgba.N(rgba.Gray),
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
				w.Invalidate()

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
