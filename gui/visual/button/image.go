package button

import (
	"image"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/pidgy/unitehud/gui/visual/screen"
	"github.com/pidgy/unitehud/nrgba"
)

type Image struct {
	*screen.Screen
	Click func(i *Image)
	Hide  bool

	hover bool
}

func (i *Image) Layout(th *material.Theme, gtx layout.Context) layout.Dimensions {
	tmp := i.Screen.Image

	if i.Screen.Image == nil {
		i.Screen.Image = image.NewRGBA(image.Rect(0, 0, 1920, 1080))
	}

	if i.Hide {
		i.Screen.Image = image.NewRGBA(i.Screen.Bounds())

		hidden := material.H5(th, "Hidden")
		hidden.Color = nrgba.White.Color()
		hidden.Alignment = text.Middle
		hidden.TextSize = unit.Sp(12)

		layout.Inset{
			Top:  unit.Dp(18),
			Left: unit.Dp(29),
		}.Layout(gtx, hidden.Layout)
	}

	dim := i.Screen.Layout(gtx)
	i.Screen.Image = tmp

	for _, e := range gtx.Events(i) {
		if e, ok := e.(pointer.Event); ok {
			switch e.Type {
			case pointer.Enter:
				i.hover = true
				i.Screen.BorderColor = nrgba.White
				i.Screen.Border = true
			case pointer.Leave:
				i.hover = false
				i.Screen.BorderColor = nrgba.Gray
				i.Screen.Border = false
			case pointer.Press:
			case pointer.Release:
				if i.hover && i.Click != nil {
					i.Click(i)

					i.Screen.BorderColor = nrgba.Gray
					i.Screen.Border = false
				}
			}
		}
	}

	area := clip.Rect(image.Rect(0, 0, dim.Size.X, dim.Size.Y)).Push(gtx.Ops)
	pointer.InputOp{
		Tag:   i,
		Types: pointer.Press | pointer.Release | pointer.Enter | pointer.Leave,
	}.Add(gtx.Ops)
	area.Pop()

	return layout.Dimensions{Size: gtx.Constraints.Max}
}

func (i *Image) SetImage(img image.Image) {
	if i.Hide {
		return
	}

	i.Image = img
}
