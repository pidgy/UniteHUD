package dropdown

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/rgba"
)

type List struct {
	Items         []*Item
	Callback      func(i *Item)
	WidthModifier int

	list *widget.List
}

type Item struct {
	Text     string
	Checked  widget.Bool
	Value    int
	Disabled bool
}

// Layout handles drawing the letters view.
func (l *List) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if l.list == nil {
		l.list = &widget.List{
			Scrollbar: widget.Scrollbar{},
			List: layout.List{
				Axis:      layout.Vertical,
				Alignment: layout.Middle,
			}}
	}

	style := material.List(th, l.list)
	style.Track.Color = color.NRGBA(rgba.White)
	style.Track.Color.A = 0xF

	return style.Layout(gtx, len(l.Items), func(gtx layout.Context, index int) layout.Dimensions {
		item := l.Items[index]

		check := material.CheckBox(th, &item.Checked, item.Text)
		check.TextSize = unit.Dp(12)
		check.Color = color.NRGBA(rgba.White)
		check.Size = unit.Dp(14)

		if item.Checked.Changed() {
			if item.Checked.Value {
				for i := range l.Items {
					if i != index {
						l.Items[i].Checked.Value = false
						continue
					}

					if l.Callback != nil {
						l.Callback(l.Items[i])
					}
				}
			} else {
				item.Checked.Value = true
			}
		}

		if item.Disabled {
			check.Color = color.NRGBA(rgba.System)
		}

		if item.Checked.Value {
			check.Color = color.NRGBA(rgba.Green)

			if item.Text == "Disabled" {
				check.Color = color.NRGBA(rgba.PaleRed)
			}
		}

		if l.WidthModifier == 0 {
			l.WidthModifier = 1
		}

		return layout.S.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			dim := check.Layout(gtx)
			dim.Size.X = gtx.Constraints.Max.X / l.WidthModifier
			return dim
		})
	})
}
