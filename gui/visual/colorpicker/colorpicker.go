package colorpicker

import (
	"fmt"
	"image/color"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/colorpicker"

	"github.com/pidgy/unitehud/fonts"
	"github.com/pidgy/unitehud/gui/visual/decorate"
	"github.com/pidgy/unitehud/nrgba"
)

type Widget struct {
	DrawButton bool

	slider colorpicker.PickerStyle
	radio  colorpicker.MuxStyle

	list material.ListStyle

	options []colorpicker.MuxOption
}

type Options colorpicker.MuxOption

func New(style *fonts.Style, options ...Options) *Widget {
	collection := fonts.NewCollection()

	c := &Widget{
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
	defer c.ApplyDefaults()

	for _, o := range options {
		c.options = append(c.options, colorpicker.MuxOption(o))
	}

	return c
}

func (c *Widget) Layout(gtx layout.Context) layout.Dimensions {
	//gtx.Constraints.Max.Y = 250
	gtx.Constraints.Min.X = 1 // Sets the radio button size.

	c.slider.ContrastBg = nrgba.Black.Color()
	c.slider.Fg = nrgba.Black.Color()
	//c.slider.Editor = material.Editor(c.radio.Theme, &c.slider.Editor, "")

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
		}
	}

	c.slider.Label = ""

	widgets := []layout.Widget{
		c.slider.Layout,
		func(gtx layout.Context) layout.Dimensions {
			return c.radio.Layout(gtx)
		},
	}

	decorate.Scrollbar(&c.list.ScrollbarStyle)

	return c.list.Layout(gtx, len(widgets), func(gtx layout.Context, index int) layout.Dimensions {
		return widgets[index](gtx)
	})
}

func (c *Widget) ApplyDefaults() {
	collection := fonts.NewCollection()

	w := colorpicker.NewMuxState(c.options...)

	c.radio = colorpicker.Mux(collection.Calibri().Theme, &w, "Options")
	c.radio.Theme.Fg = nrgba.White.Color()

	c.slider.Label = c.options[0].Label
	c.slider.SetColor(*c.options[0].Value)
	c.slider.Theme.Fg = nrgba.White.Color()
	c.slider.Theme.ContrastFg = nrgba.Black.Color()
	c.slider.Theme.ContrastBg = nrgba.Black.Color()
}

func hex(c color.NRGBA) string {
	return fmt.Sprintf("%02X%02X%02X", c.R, c.G, c.B)
}
