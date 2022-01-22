package textblock

import (
	"image"
	"image/color"
	"strings"

	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type TextBlock struct {
	Text string
}

func (t *TextBlock) Layout(gtx layout.Context, texts []string) layout.Dimensions {
	th := material.NewTheme(gofont.Collection())
	th.TextSize = unit.Px(12)

	block := material.H5(th, strings.Join(texts, "\n"))
	block.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	block.Alignment = text.Alignment(text.Start)

	return Fill(gtx,
		color.NRGBA{R: 25, G: 25, B: 100, A: 50},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Px(5), Left: unit.Px(5)}.Layout(gtx, block.Layout)
		},
	)
}

// ColorBox creates a widget with the specified dimensions and color.
func ColorBox(gtx layout.Context, size image.Point, c color.NRGBA) layout.Dimensions {
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: c}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	return widget.Border{
		Color: color.NRGBA{R: 100, G: 100, B: 100, A: 50},
		Width: unit.Px(2),
	}.Layout(
		gtx,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: size}
		})
}

func Fill(gtx layout.Context, backgroundColor color.NRGBA, w layout.Widget) layout.Dimensions {
	ColorBox(gtx, gtx.Constraints.Max, backgroundColor)
	return layout.NW.Layout(gtx, w)
}