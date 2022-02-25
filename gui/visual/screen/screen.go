package screen

import (
	"image"
	"image/color"

	"gioui.org/unit"
	"gioui.org/widget"
	"golang.org/x/image/draw"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

type Screen struct {
	image.Image
	Border      bool
	BorderColor color.NRGBA
	Descale     int
}

func (s *Screen) Layout(gtx layout.Context) layout.Dimensions {
	if s.Image == nil {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}

	if s.Descale == 0 {
		s.Descale = 2
	}

	if s.Border {
		return widget.Border{
			Color:        s.BorderColor,
			Width:        unit.Px(3),
			CornerRadius: unit.Px(1),
		}.Layout(gtx, s.borderLayout)
	}

	return s.layout(gtx)
}

func (s *Screen) borderLayout(gtx layout.Context) layout.Dimensions {
	dst := image.NewRGBA(image.Rect(0, 0, s.Image.Bounds().Max.X/s.Descale, s.Image.Bounds().Max.Y/s.Descale))
	draw.NearestNeighbor.Scale(dst, dst.Rect, s.Image, s.Image.Bounds(), draw.Over, nil)

	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	op := paint.NewImageOp(dst)
	op.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	return layout.Dimensions{Size: s.Image.Bounds().Max.Div(s.Descale)}
}

func (s *Screen) layout(gtx layout.Context) layout.Dimensions {
	dst := image.NewRGBA(image.Rect(0, 0, s.Image.Bounds().Max.X/s.Descale, s.Image.Bounds().Max.Y/s.Descale))
	draw.NearestNeighbor.Scale(dst, dst.Rect, s.Image, s.Image.Bounds(), draw.Over, nil)

	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	op := paint.NewImageOp(dst)
	op.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
