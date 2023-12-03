// https://github.com/gioui/gio-example/blob/main/tabs/slider.go
package slider

import (
	"fmt"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/widget/material"
	"github.com/pidgy/unitehud/core/nrgba"
	"github.com/pidgy/unitehud/gui/cursor"
)

type Widget struct {
	Slider         material.SliderStyle
	Label          material.LabelStyle
	TextColors     []nrgba.NRGBA
	OnValueChanged func(float32)
}

func (s *Widget) Layout(gtx layout.Context) layout.Dimensions {
	defer s.event()

	v := s.Slider.Float.Value
	if v < 0 {
		v *= -1
	}
	col := int((float32(len(s.TextColors)) * (v / s.Slider.Max)))
	if col == len(s.TextColors) {
		col--
	}

	s.Label.Color = s.TextColors[col].Color()
	s.Slider.Color = s.Label.Color
	s.Label.Text = fmt.Sprintf("%.0f", s.Slider.Float.Value)

	return layout.Flex{
		Axis: layout.Horizontal,
	}.Layout(gtx,
		layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
			return s.Slider.Layout(gtx)
		}),
		layout.Flexed(.1, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, s.Label.Layout)
		}),
	)
}

func (s *Widget) event() {
	if s.Slider.Float.Dragging() {
		cursor.Is(pointer.CursorPointer)
		return
	}

	if !s.Slider.Float.Changed() {
		return
	}

	if s.OnValueChanged == nil {
		return
	}
	s.OnValueChanged(s.Slider.Float.Value)

	cursor.Is(pointer.CursorDefault)
}
