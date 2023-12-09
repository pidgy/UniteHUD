package colorpicker

import (
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/colorpicker"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/fonts"
	"github.com/pidgy/unitehud/core/nrgba"
	"github.com/pidgy/unitehud/gui/visual/decorate"
)

type (
	Option colorpicker.MuxOption

	Widget struct {
		DrawButton bool

		sliders colorpicker.PickerStyle
		radio   colorpicker.MuxStyle

		list material.ListStyle

		options []colorpicker.MuxOption
	}
)

func New(style *fonts.Style, options ...Option) *Widget {
	collection := fonts.NewCollection()

	c := &Widget{
		sliders: colorpicker.PickerStyle{
			Theme:         collection.Calibri().Theme,
			MonospaceFace: "sans-serif",
			State: &colorpicker.State{
				Editor: widget.Editor{
					Alignment:       text.Middle,
					MaxLen:          6,
					Filter:          "abcdefABCDEF0123456789",
					SingleLine:      true,
					Submit:          true,
					InputHint:       key.HintText,
					LineHeight:      5,
					LineHeightScale: 2,
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

	c.sliders.ContrastBg = nrgba.White.Color()
	c.sliders.Fg = config.Current.Theme.Borders

	switch {
	case c.radio.Update(gtx):
		col := *c.radio.Color()
		c.sliders.SetColor(col)
		c.sliders.Label = c.radio.Value
		c.sliders.Editor.SetText(nrgba.NRGBA(c.sliders.Color()).Hex())
	case c.sliders.Changed():
		col := c.sliders.Color()
		c.radio.Options[c.radio.Value].R = col.R
		c.radio.Options[c.radio.Value].G = col.G
		c.radio.Options[c.radio.Value].B = col.B
		c.radio.Options[c.radio.Value].A = col.A
		c.sliders.Editor.SetText(nrgba.NRGBA(c.sliders.Color()).Hex())
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

	c.radio = colorpicker.Mux(collection.NotoSans().Theme, &w, "Options")
	c.radio.Theme.Fg = nrgba.White.Color()
	c.radio.Theme.Bg = nrgba.Red.Color()
	c.radio.Theme.TextSize = 12

	c.sliders.Label = c.options[0].Label
	c.sliders.SetColor(*c.options[0].Value)
	c.sliders.TextSize = 12
}
