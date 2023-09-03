package colorpicker

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/colorpicker"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/fonts"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/nrgba"
)

type Widget struct {
	DefaultButton *button.Widget
	DrawButton    bool

	slider colorpicker.PickerStyle
	radio  colorpicker.MuxStyle

	list material.ListStyle

	options []colorpicker.MuxOption
}

type Option colorpicker.MuxOption

func New(style *fonts.Style, options ...Option) *Widget {
	collection := fonts.NewCollection()

	c := &Widget{
		DefaultButton: &button.Widget{
			Text:            "Default",
			Pressed:         nrgba.Transparent80,
			Released:        nrgba.DarkGray,
			TextSize:        unit.Sp(14),
			TextInsetBottom: unit.Dp(-2),
			Size:            image.Pt(80, 20),
			BorderWidth:     unit.Sp(.2),
			Font:            style,

			OnHoverHint: func() {},
		},

		slider: colorpicker.PickerStyle{
			MonospaceFace: "monospace",

			Theme: collection.Calibri().Theme,

			State: &colorpicker.State{
				Editor: widget.Editor{
					MaxLen:     6,
					Filter:     "abcdefABCDEF0123456789",
					SingleLine: true,
					Submit:     true,
					InputHint:  key.HintText,
				},
			},
		},

		list: material.List(collection.Calibri().Theme, &widget.List{
			Scrollbar: widget.Scrollbar{},
			List: layout.List{
				Axis:      layout.Vertical,
				Alignment: layout.Start,
			},
		}),
	}
	defer c.defaults()

	for _, o := range options {
		c.options = append(c.options, colorpicker.MuxOption(o))
	}

	c.DefaultButton.Click = func(this *button.Widget) {
		defer this.Deactivate()

		config.Current.SetDefaultTheme()

		c.defaults()
	}

	return c
}

func (c *Widget) Layout(gtx layout.Context) layout.Dimensions {
	//gtx.Constraints.Max.Y = 250
	gtx.Constraints.Min.X = 1 // Sets the radio button size.

	switch {
	case c.radio.Changed():
		col := *c.radio.Color()
		c.slider.SetColor(col)
		c.slider.Label = c.radio.Value
		c.slider.Editor.SetText(hex(col))
	case c.slider.Changed():
		col := c.slider.Color()
		c.radio.Options[c.radio.Value].R = col.R
		c.radio.Options[c.radio.Value].G = col.G
		c.radio.Options[c.radio.Value].B = col.B
		c.radio.Options[c.radio.Value].A = col.A

		if !c.slider.Editor.Focused() {
			c.slider.Editor.SetText(hex(col))
		} else {
			c.slider.Editor.SetText(strings.ToUpper(c.slider.Editor.Text()))
		}
	}

	widgets := []layout.Widget{
		c.slider.Layout,
		func(gtx layout.Context) layout.Dimensions {
			return c.radio.Layout(gtx)
		},
	}
	if c.DrawButton {
		widgets = append([]layout.Widget{c.DefaultButton.Layout}, widgets...)
	}

	return c.list.Layout(gtx, len(widgets), func(gtx layout.Context, index int) layout.Dimensions {
		return widgets[index](gtx)
	})
}

func (c *Widget) defaults() {
	collection := fonts.NewCollection()

	w := colorpicker.NewMuxState(c.options...)

	c.radio = colorpicker.Mux(collection.Calibri().Theme, &w, "Options")
	c.radio.Theme.Fg = nrgba.White.Color()

	c.slider.Label = c.options[0].Label
	c.slider.SetColor(*c.options[0].Value)
	c.slider.Theme.Fg = nrgba.White.Color()
	c.slider.Theme.ContrastFg = nrgba.Black.Color()
	c.slider.Theme.ContrastBg = nrgba.Black.Color()

	c.list.Track.Color = config.Current.Theme.Scrollbar
}

func hex(c color.NRGBA) string {
	return fmt.Sprintf("%02X%02X%02X", c.R, c.G, c.B)
}
