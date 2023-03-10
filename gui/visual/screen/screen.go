package screen

import (
	"image"
	"image/color"

	"gioui.org/unit"
	"gioui.org/widget"

	"gioui.org/layout"
	"gioui.org/op/paint"
)

type Screen struct {
	image.Image
	Border        bool
	BorderColor   color.NRGBA
	VerticalScale bool

	Dims layout.Dimensions
}

func (s *Screen) Layout(gtx layout.Context) layout.Dimensions {
	defer func() {
		r := recover()
		if r != nil {

		}
	}()

	if s == nil || s.Image == nil || s.Image.Bounds().Size().Eq(image.Pt(0, 0)) {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}

	/*
		switch {
		case s.ScaleY == 0 && s.ScaleX == 0:
			s.ScaleX = float32(s.Image.Bounds().Max.X) / 100
			if s.ScaleX == 0 {
				s.ScaleX = 1
			}

			s.ScaleY = float32(s.Image.Bounds().Max.Y) / 50
			if s.ScaleY == 0 {
				s.ScaleY = 1
			}
		case s.ScaleX == 0 && s.ScaleY != 0:
			s.ScaleX = s.ScaleY
		case s.ScaleX != 0 && s.ScaleY == 0:
			s.ScaleY = s.ScaleX
		}
	*/

	return s.layout(gtx)
}

func (s *Screen) Scale(gtx layout.Context) float32 {
	if s.VerticalScale {
		return float32(gtx.Constraints.Max.Y) / float32(s.Image.Bounds().Dy())
	}

	return float32(gtx.Constraints.Max.X) / float32(s.Image.Bounds().Dx())
}

func (s *Screen) layout(gtx layout.Context) layout.Dimensions {
	return widget.Border{
		Color:        s.BorderColor,
		Width:        unit.Px(3),
		CornerRadius: unit.Px(1),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		s.Dims = widget.Image{
			Src:      paint.NewImageOp(s.Image),
			Position: layout.Center,
			Scale:    s.Scale(gtx),
		}.Layout(gtx)

		return s.Dims
	})
}
