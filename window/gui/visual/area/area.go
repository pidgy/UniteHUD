package area

import (
	"image"
	"image/color"

	"gioui.org/font/gofont"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/window/gui/visual/button"
)

type Area struct {
	Text    string
	Subtext string
	Hidden  bool

	*button.Button

	Min, Max image.Point

	color.NRGBA

	drag         bool
	dragID       pointer.ID
	dragX, dragY float32
}

func (a *Area) Rectangle() image.Rectangle {
	return image.Rect(a.Min.X*2, a.Min.Y*2, a.Max.X*2, a.Max.Y*2)
}

func (a *Area) Layout(gtx layout.Context) layout.Dimensions {
	{
		// handle input
		for _, ev := range gtx.Events(a) {
			e, ok := ev.(pointer.Event)
			if !ok {
				continue
			}

			switch e.Type {
			case pointer.Press:
				if a.drag || !a.Active {
					break
				}

				a.dragID = e.PointerID
				a.dragX = e.Position.X
				a.dragY = e.Position.Y
			case pointer.Drag:
				if a.dragID != e.PointerID || !a.Active {
					break
				}

				deltaX := e.Position.X - a.dragX
				a.dragX = e.Position.X
				deltaY := e.Position.Y - a.dragY
				a.dragY = e.Position.Y

				maxX := int(float32(gtx.Constraints.Max.X)*.99) - 1
				maxY := int(float32(gtx.Constraints.Max.Y)*.85) - 3

				if a.Min.X+int(deltaX) < 0 || a.Min.Y+int(deltaY) < 0 || a.Max.X+int(deltaX) > maxX || a.Max.Y+int(deltaY) > maxY {
					break
				}

				a.Min.X += int(deltaX)
				a.Min.Y += int(deltaY)
				a.Max.X += int(deltaX)
				a.Max.Y += int(deltaY)
			case pointer.Release:
				fallthrough
			case pointer.Cancel:
				a.drag = false
			}
		}

		area := clip.Rect{Min: a.Min, Max: a.Max}.Push(gtx.Ops)
		pointer.InputOp{
			Tag:   a,
			Types: pointer.Press | pointer.Drag | pointer.Release,
			Grab:  a.drag,
		}.Add(gtx.Ops)
		area.Pop()
	}

	layout.UniformInset(unit.Dp(5)).Layout(
		gtx,
		func(gtx layout.Context) layout.Dimensions {
			defer clip.Rect{Min: a.Min, Max: a.Max}.Push(gtx.Ops).Pop()
			paint.ColorOp{Color: a.NRGBA}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)

			return layout.Dimensions{Size: a.Max.Sub(a.Min)}
		})

	layout.Inset{
		Left: unit.Px(float32(a.Min.X) + 5),
		Top:  unit.Px(float32(a.Min.Y) + 5),
	}.Layout(
		gtx,
		func(gtx layout.Context) layout.Dimensions {
			return widget.Border{
				Color: a.NRGBA,
				Width: unit.Px(2),
			}.Layout(
				gtx,
				func(gtx layout.Context) layout.Dimensions {
					defer clip.Rect{Min: a.Min, Max: a.Max}.Push(gtx.Ops).Pop()
					return layout.Dimensions{Size: a.Max.Sub(a.Min)}
				})
		})

	return layout.Inset{
		Left: unit.Px(float32(a.Min.X)),
		Top:  unit.Px(float32(a.Min.Y)),
	}.Layout(
		gtx,
		func(gtx layout.Context) layout.Dimensions {
			title := material.Body1(material.NewTheme(gofont.Collection()), a.Text+" "+a.Subtext)
			title.Color = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
			layout.Inset{
				Left: unit.Px(15),
				Top:  unit.Px(10),
			}.Layout(gtx, title.Layout)

			return layout.Dimensions{Size: a.Max.Sub(a.Min)}
		})
}
