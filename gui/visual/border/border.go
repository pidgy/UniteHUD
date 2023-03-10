package border

import (
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

// Bottom lays out a widget and draws a border below it.
type Bottom struct {
	Color        color.NRGBA
	CornerRadius unit.Value
	Width        unit.Value
}

// Top lays out a widget and draws a border above it.
type Top struct {
	Color        color.NRGBA
	CornerRadius unit.Value
	Width        unit.Value
}

func (b *Bottom) Layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	dims := w(gtx)
	sz := layout.FPt(dims.Size)

	rr := float32(gtx.Px(b.CornerRadius))
	width := float32(gtx.Px(b.Width))
	sz.X -= width
	sz.Y -= width

	r := f32.Rect(0, sz.Y, sz.X, sz.Y)
	r = r.Add(f32.Point{X: width * 0.5, Y: width * 0.5})

	paint.FillShape(gtx.Ops,
		b.Color,
		clip.Stroke{
			Path:  clip.UniformRRect(r, rr).Path(gtx.Ops),
			Width: width,
		}.Op(),
	)

	return dims
}

func (t *Top) Layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	dims := w(gtx)
	sz := layout.FPt(dims.Size)

	rr := float32(gtx.Px(t.CornerRadius))
	width := float32(gtx.Px(t.Width))
	sz.X -= width
	sz.Y -= width

	r := f32.Rect(0, 0, sz.X, 0)
	r = r.Add(f32.Point{X: width * 0.5, Y: width * 0.5})

	paint.FillShape(gtx.Ops,
		t.Color,
		clip.Stroke{
			Path:  clip.UniformRRect(r, rr).Path(gtx.Ops),
			Width: width,
		}.Op(),
	)

	return dims
}
