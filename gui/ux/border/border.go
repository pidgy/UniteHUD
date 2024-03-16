package border

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
)

// Bottom lays out a widget and draws a border below it.
type Bottom struct {
	NRGBA        nrgba.NRGBA
	CornerRadius unit.Sp
	Width        unit.Sp
}

// Top lays out a widget and draws a border above it.
type Top struct {
	NRGBA        nrgba.NRGBA
	CornerRadius unit.Sp
	Width        unit.Sp
}

func (b *Bottom) Layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	dims := w(gtx)
	sz := layout.FPt(dims.Size)

	rr := int(gtx.Sp(b.CornerRadius))
	width := float32(gtx.Sp(b.Width))
	sz.X -= width
	sz.Y -= width

	r := image.Rect(0, int(sz.Y), int(sz.X), int(sz.Y))
	r = r.Add(image.Point{X: int(width * 0.5), Y: int(width * 0.5)})

	paint.FillShape(gtx.Ops,
		b.NRGBA.Color(),
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

	rr := int(gtx.Sp(t.CornerRadius))
	width := float32(gtx.Sp(t.Width))
	sz.X -= width
	sz.Y -= width

	r := image.Rect(0, 0, int(sz.X), 0)
	r = r.Add(image.Point{X: int(width * 0.5), Y: int(width * 0.5)})

	paint.FillShape(gtx.Ops,
		t.NRGBA.Color(),
		clip.Stroke{
			Path:  clip.UniformRRect(r, rr).Path(gtx.Ops),
			Width: width,
		}.Op(),
	)

	return dims
}
