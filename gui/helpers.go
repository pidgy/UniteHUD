package gui

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/pidgy/unitehud/nrgba"
)

func colorBox(gtx layout.Context, size image.Point, n nrgba.NRGBA) layout.Dimensions {
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: n.Color()}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: size}
}

func fill(gtx layout.Context, bg nrgba.NRGBA, w layout.Widget) layout.Dimensions {
	colorBox(gtx, gtx.Constraints.Max, bg)
	return layout.NW.Layout(gtx, w)
}

func line(gtx layout.Context, rect clip.Rect, n nrgba.NRGBA) layout.Dimensions {
	defer rect.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: n.Color()}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return colorBox(gtx, gtx.Constraints.Max, n)
}
