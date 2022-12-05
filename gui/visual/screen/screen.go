package screen

import (
	"image"
	"image/color"

	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/rs/zerolog/log"
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
	defer func() {
		r := recover()
		if r != nil {
			log.Error().Err(r.(error)).Msg("layout failed")
		}
	}()

	if s == nil || s.Image == nil || s.Image.Bounds().Size().Eq(image.Pt(0, 0)) {
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

	return s.layout(gtx)
}

func (s *Screen) layout(gtx layout.Context) layout.Dimensions {
	rect := image.Rect(0, 0, s.Image.Bounds().Max.X/s.ScaleX, s.Image.Bounds().Max.Y/s.ScaleY)
	dst := image.NewRGBA(rect)
	draw.NearestNeighbor.Scale(dst, dst.Rect, s.Image, s.Image.Bounds(), draw.Over, nil)

	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	op := paint.NewImageOp(dst)
	op.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	if !s.Border {
		return layout.Dimensions{Size: rect.Size()}
	}

	return widget.Border{
		Color:        s.BorderColor,
		Width:        unit.Px(3),
		CornerRadius: unit.Px(1),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: rect.Size()}
	})
}
