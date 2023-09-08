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

	sliders colorpicker.PickerStyle
	radio   colorpicker.MuxStyle

	list material.ListStyle

	options []colorpicker.MuxOption
}

type Options colorpicker.MuxOption

func New(style *fonts.Style, options ...Options) *Widget {
	collection := fonts.NewCollection()

	c := &Widget{
		sliders: colorpicker.PickerStyle{
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

	c.sliders.ContrastBg = nrgba.Discord.Color()
	c.sliders.Fg = nrgba.Discord.Color()
	//c.slider.Editor = material.Editor(c.radio.Theme, &c.slider.Editor, "")

	switch {
	case c.radio.Changed():
		col := *c.radio.Color()
		c.sliders.SetColor(col)
		c.sliders.Label = c.radio.Value
		c.sliders.Editor.SetText(hex(col))
	case c.sliders.Changed():
		col := c.sliders.Color()
		c.radio.Options[c.radio.Value].R = col.R
		c.radio.Options[c.radio.Value].G = col.G
		c.radio.Options[c.radio.Value].B = col.B
		c.radio.Options[c.radio.Value].A = col.A

		if !c.sliders.Editor.Focused() {
			c.sliders.Editor.SetText(hex(col))
		}
	}

	c.sliders.Label = ""

	widgets := []layout.Widget{
		c.sliders.Layout,
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

	c.sliders.Label = c.options[0].Label
	c.sliders.SetColor(*c.options[0].Value)

	// ContrastFg (sliders text color) is black when:
	// Fg == White
	// ContrastBg == Black
	// c.sliders.Theme.ContrastFg = nrgba.Discord.Color()
	c.sliders.Theme.ContrastBg = nrgba.Discord.Color()
}

func hex(c color.NRGBA) string {
	return fmt.Sprintf("%02X%02X%02X", c.R, c.G, c.B)
}
