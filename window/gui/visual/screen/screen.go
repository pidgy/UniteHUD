package screen

import (
	"image"

	"golang.org/x/image/draw"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

type Screen struct {
	image.Image
}

func (s *Screen) Layout(gtx layout.Context) layout.Dimensions {
	if s.Image == nil {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}

	dst := image.NewRGBA(image.Rect(0, 0, s.Image.Bounds().Max.X/2, s.Image.Bounds().Max.Y/2))
	draw.NearestNeighbor.Scale(dst, dst.Rect, s.Image, s.Image.Bounds(), draw.Over, nil)

	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	op := paint.NewImageOp(dst)
	op.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
