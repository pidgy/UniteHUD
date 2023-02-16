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

	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/video/device"
)

const alpha = 0xCC

var (
	Locked = rgba.N(rgba.Alpha(rgba.Black, alpha))
	Match  = rgba.N(rgba.Alpha(rgba.DarkSeafoam, alpha))
	Miss   = rgba.N(rgba.Alpha(rgba.Red, alpha))
)

type Area struct {
	Text          string
	TextSize      unit.Value
	TextAlignLeft bool
	Subtext       string
	Hidden        bool
	Theme         *material.Theme

	*button.Button

	Min, Max image.Point

	color.NRGBA

	drag         bool
	dragID       pointer.ID
	dragX, dragY float32
}

func (a *Area) Rectangle() image.Rectangle {
	return toScale(image.Rectangle{a.Min, a.Max})
	// return image.Rect(a.Min.X*2, a.Min.Y*2, a.Max.X*2, a.Max.Y*2)
}

func toScale(r image.Rectangle) image.Rectangle {
	return image.Rectangle{r.Min.Mul(2), r.Max.Mul(2)}
}

func (a *Area) Layout(gtx layout.Context) layout.Dimensions {
	if a.TextSize.V == unit.Px(0).V {
		a.TextSize = unit.Px(16)
	}

	if a.Theme == nil {
		a.Theme = material.NewTheme(gofont.Collection())
	}

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

				// maxX := int(float32(gtx.Constraints.Max.X)*.99) - 1
				// maxY := int(float32(gtx.Constraints.Max.Y)*.75) - 3

				if !toScale(image.Rectangle{
					a.Min.Add(image.Pt(int(deltaX), int(deltaY))),
					a.Max.Add(image.Pt(int(deltaX), int(deltaY))),
				}).In(device.HD1080) {
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
			defer clip.Rect{
				Min: a.Min,
				Max: a.Max,
			}.Push(gtx.Ops).Pop()

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

	text := a.Text + " " + a.Subtext

	/*
		layout.Inset{
			Left: unit.Px(float32(a.Min.X)),
			Top:  unit.Px(float32(a.Min.Y)),
		}.Layout(
			gtx,
			func(gtx layout.Context) layout.Dimensions {
				// Text background.
				defer clip.Rect{
					Min: image.Pt(5, 5),
					Max: image.Pt(len(text)*int((a.TextSize.V)-(a.TextSize.V/3)), 25),
				}.Push(gtx.Ops).Pop()

				paint.ColorOp{Color: color.NRGBA{A: 0x9F}}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)

				return layout.Dimensions{Size: a.Max.Sub(a.Min)}
			})
	*/
	return layout.Inset{
		Left: unit.Px(float32(a.Min.X)),
		Top:  unit.Px(float32(a.Min.Y)),
	}.Layout(
		gtx,
		func(gtx layout.Context) layout.Dimensions {
			title := material.Body1(a.Theme, text)
			title.TextSize = a.TextSize
			title.Font.Weight = 500
			title.Color = rgba.N(rgba.White)
			if a.NRGBA == Match {
				title.Font.Weight = 1000
			}

			layout.Inset{
				Left: unit.Px(10),
				Top:  unit.Px(5),
			}.Layout(gtx, title.Layout)

			return layout.Dimensions{Size: a.Max.Sub(a.Min)}
		})
}
