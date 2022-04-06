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
	Border         bool
	BorderColor    color.NRGBA
	ScaleX, ScaleY int
}

func (s *Screen) Layout(gtx layout.Context) layout.Dimensions {
	if s.Image == nil {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}

	switch {
	case s.ScaleY == 0 && s.ScaleX == 0:
		s.ScaleX = s.Image.Bounds().Max.X / 100
		if s.ScaleX == 0 {
			s.ScaleX = 1
		}

		s.ScaleY = s.Image.Bounds().Max.Y / 50
		if s.ScaleY == 0 {
			s.ScaleY = 1
		}
	case s.ScaleX == 0 && s.ScaleY != 0:
		s.ScaleX = s.ScaleY
	case s.ScaleX != 0 && s.ScaleY == 0:
		s.ScaleY = s.ScaleX
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
	dst := image.NewRGBA(image.Rect(0, 0, s.Image.Bounds().Max.X/s.ScaleX, s.Image.Bounds().Max.Y/s.ScaleY))
	draw.NearestNeighbor.Scale(dst, dst.Rect, s.Image, s.Image.Bounds(), draw.Over, nil)

	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	op := paint.NewImageOp(dst)
	op.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	return layout.Dimensions{Size: s.Image.Bounds().Max.Div((s.ScaleX + s.ScaleY) / 2)}
}

func (s *Screen) layout(gtx layout.Context) layout.Dimensions {
	dst := image.NewRGBA(image.Rect(0, 0, s.Image.Bounds().Max.X/s.ScaleX, s.Image.Bounds().Max.Y/s.ScaleY))
	draw.NearestNeighbor.Scale(dst, dst.Rect, s.Image, s.Image.Bounds(), draw.Over, nil)

	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	op := paint.NewImageOp(dst)
	op.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
