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
	Items    []*Item
	Callback func(i *Item)

	list *widget.List
}

type Item struct {
	Text    string
	Checked widget.Bool
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

		if item.Checked.Changed() && item.Checked.Value {
			for i := range l.Items {
				if i != index {
					l.Items[i].Checked.Value = false
					continue
				}

				if l.Callback != nil {
					l.Callback(l.Items[i])
				}
			}
		}

		if item.Checked.Value {
			check.Color = color.NRGBA(rgba.Green)
		}

		return layout.S.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			dim := check.Layout(gtx)
			dim.Size.X = gtx.Constraints.Max.X
			return dim
		})
	})
}
