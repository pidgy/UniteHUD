// https://github.com/gioui/gio-example/blob/main/tabs/slider.go
package slider

import (
	"fmt"
	"image"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/widget/material"
	"github.com/pidgy/unitehud/cursor"
	"github.com/pidgy/unitehud/nrgba"
)

type Slider struct {
	Slider         material.SliderStyle
	Label          material.LabelStyle
	TextColors     []nrgba.NRGBA
	OnValueChanged func(float32)

	dims layout.Dimensions
}

func (s *Slider) Layout(gtx layout.Context) layout.Dimensions {
	defer s.event()

	col := int((float32(len(s.TextColors)) * (s.Slider.Float.Value / s.Slider.Max)))
	if col == len(s.TextColors) {
		col--
	}

	s.Label.Color = s.TextColors[col].Color()
	s.Slider.Color = s.Label.Color
	s.Label.Text = fmt.Sprintf("%.0f", s.Slider.Float.Value)
	for _, e := range gtx.Events(s) {
		e, ok := e.(pointer.Event)
		if !ok {
			continue
		}

		switch e.Type {
		case pointer.Enter, pointer.Move, pointer.Press, pointer.Release:
			cursor.Is(pointer.CursorPointer)
		case pointer.Leave:
			cursor.Is(pointer.CursorDefault)
		}
	}

	return layout.Flex{
		Axis: layout.Horizontal,
	}.Layout(gtx,
		layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
			if s.dims.Size.X != 0 {
				rect := image.Rectangle(gtx.Constraints)
				rect.Min.X = 0
				rect.Max.Y = s.dims.Size.Y
				area := clip.Rect(rect).Push(gtx.Ops)
				pointer.InputOp{
					Tag:   s,
					Types: pointer.Press | pointer.Release | pointer.Enter | pointer.Leave | pointer.Move | pointer.Cancel,
					Grab:  true,
				}.Add(gtx.Ops)
				area.Pop()
			}

			s.dims = s.Slider.Layout(gtx)

			return s.dims
		}),
		layout.Flexed(.1, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, s.Label.Layout)
		}),
	)
}

func (s *Slider) event() {
	if s.OnValueChanged == nil {
		return
	}

	if !s.Slider.Float.Changed() {
		return
	}

	s.OnValueChanged(s.Slider.Float.Value)
}
