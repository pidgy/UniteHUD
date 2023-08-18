package screen

import (
	"image"

	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/nrgba"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

type Screen struct {
	image.Image
	Border      bool
	BorderColor nrgba.NRGBA

	VerticalScale, AutoScale bool

	Splash bool

	Dims layout.Dimensions
}

func (s *Screen) Layout(gtx layout.Context) layout.Dimensions {
	defer func() {
		r := recover()
		if r != nil {
			notify.SystemWarn("Successfully recovered from fatal error (%v)", r)
		}
	}()

	if s == nil || s.Image == nil || s.Image.Bounds().Size().Eq(image.Pt(0, 0)) {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}

	return s.layout(gtx)
}

func (s *Screen) Scale(gtx layout.Context) float32 {
	if s.AutoScale {
		return 1
	}

	if s.VerticalScale {
		return float32(gtx.Constraints.Max.Y) / float32(s.Image.Bounds().Dy()) //+ float32(gtx.Constraints.Max.X)/float32(s.Image.Bounds().Dx())
	}

	return float32(gtx.Constraints.Max.X) / float32(s.Image.Bounds().Dx())
}

func (s *Screen) layout(gtx layout.Context) layout.Dimensions {
	if s.Splash {
		defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
		paint.ColorOp{Color: nrgba.Splash.Color()}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
	}

	if !s.Border {
		s.BorderColor = s.BorderColor.Alpha(0)
	}

	fit := widget.Unscaled
	if s.AutoScale {
		fit = widget.ScaleDown
	}

	s.Dims = widget.Image{
		Src:      paint.NewImageOp(s.Image),
		Position: layout.Center,
		Scale:    s.Scale(gtx),
		Fit:      fit,
	}.Layout(gtx)

	return widget.Border{
		Color:        s.BorderColor.Color(),
		Width:        unit.Dp(3),
		CornerRadius: unit.Dp(1),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return s.Dims
	})
}
