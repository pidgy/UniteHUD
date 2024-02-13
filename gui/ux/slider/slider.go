// https://github.com/gioui/gio-example/blob/main/tabs/slider.go
package slider

import (
	"fmt"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/nrgba"
	"github.com/pidgy/unitehud/gui/cursor"
)

type Widget struct {
	Theme          *material.Theme
	Label          material.LabelStyle
	TextColors     []nrgba.NRGBA
	OnValueChanged func(float32)
	Min, Max       float32

	slider material.SliderStyle
}

func (s *Widget) Layout(gtx layout.Context) layout.Dimensions {
	if s.slider.Float == nil {
		s.slider = material.Slider(s.Theme, &widget.Float{Value: float32(config.Current.Advanced.DecreasedCaptureLevel)})
	}
	s.slider.Float.Value = float32(config.Current.Advanced.DecreasedCaptureLevel)
	if s.Max == 0 {
		s.Max = 1
	}

	v := s.slider.Float.Value
	switch {
	case v < s.Min:
		v = s.Min
	case v > s.Max:
		v = s.Max
	}

	col := int((float32(len(s.TextColors)) * (v / s.Max)))
	if col == len(s.TextColors) {
		col--
	}
	s.Label.Color = s.TextColors[col].Color()

	s.slider.Color = s.Label.Color
	s.Label.Text = fmt.Sprintf("%.0f", s.slider.Float.Value)

	return layout.Flex{
		Axis: layout.Horizontal,
	}.Layout(gtx,
		layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
			if s.slider.Float.Update(gtx) {
				cursor.Is(pointer.CursorPointer)
				s.OnValueChanged(s.slider.Float.Value)
			}
			return s.slider.Layout(gtx)
		}),

		layout.Flexed(.1, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, s.Label.Layout)
		}),
	)
}
