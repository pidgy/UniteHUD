package dropdown

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/rgba"
)

const alpha = 0xCC

var (
	Disabled = rgba.N(rgba.Red)
	Enabled  = rgba.N(rgba.DarkSeafoam)
)

type List struct {
	Items         []*Item
	Callback      func(i *Item)
	WidthModifier int
	Radio         bool
	TextSize      float32

	list *widget.List
}

type Item struct {
	Text     string
	Checked  widget.Bool
	Value    int
	Disabled bool
	Weight   int

	Callback func()
}

// Layout handles drawing the letters view.
func (l *List) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if l.list == nil {
		l.list = &widget.List{
			Scrollbar: widget.Scrollbar{},
			List: layout.List{
				Axis:      layout.Vertical,
				Alignment: layout.Start,
			},
		}
	}

	style := material.List(th, l.list)
	style.Track.Color = color.NRGBA(rgba.Gray)
	style.Track.Color.A = 0xFF

	return style.Layout(gtx, len(l.Items), func(gtx layout.Context, index int) layout.Dimensions {
		item := l.Items[index]

		check := material.CheckBox(th, &item.Checked, item.Text)
		check.Font.Weight = text.Weight(item.Weight)
		check.Color = color.NRGBA(rgba.White)
		check.Size = unit.Px(15)
		check.IconColor = rgba.N(rgba.White)

		if item.Checked.Changed() {
			if item.Disabled {
				item.Checked.Value = !item.Checked.Value
			} else if item.Callback != nil {
				item.Callback()
			}

			if l.Radio {
				for i := range l.Items {
					if i == index {
						l.Items[i].Checked.Value = true
						continue
					}

					l.Items[i].Checked.Value = false
				}
			}

			if !item.Disabled && l.Callback != nil {
				l.Callback(item)
			}
		}

		if item.Checked.Value {
			check.Color = Enabled
			if item.Text == "Disabled" {
				check.Color = Disabled
			}
		}
		if item.Disabled {
			check.Color = Disabled
		}
		switch {
		case item.Checked.Hovered():
			hoverItem(&gtx, index)
		case item.Checked.Value:
			selectedItem(&gtx, index)
		}

		if l.WidthModifier == 0 {
			l.WidthModifier = 1
		}

		return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			dim := check.Layout(gtx)
			dim.Size.X = gtx.Constraints.Max.X / l.WidthModifier
			return dim
		})
	})
}

func selectedItem(gtx *layout.Context, index int) {
	widget.Border{
		Color:        rgba.N(rgba.Alpha(rgba.White, 5)),
		Width:        unit.Px(1),
		CornerRadius: unit.Px(3),
	}.Layout(*gtx, func(gtx layout.Context) layout.Dimensions {
		return colorRect(gtx,
			clip.Rect{
				Min: image.Pt(
					0,
					0,
				),
				Max: image.Pt(
					gtx.Constraints.Max.X,
					20,
				),
			},
			rgba.N(rgba.Alpha(rgba.Black, 100)),
		)
	})
}

func hoverItem(gtx *layout.Context, index int) {
	widget.Border{
		Color:        rgba.N(rgba.Alpha(rgba.White, 100)),
		Width:        unit.Px(1),
		CornerRadius: unit.Px(3),
	}.Layout(*gtx, func(gtx layout.Context) layout.Dimensions {
		return colorRect(gtx,
			clip.Rect{
				Min: image.Pt(
					0,
					0,
				),
				Max: image.Pt(
					gtx.Constraints.Max.X,
					20,
				),
			},
			rgba.N(rgba.Alpha(rgba.White, 5)),
		)
	})
}

func colorRRect(gtx layout.Context, rect clip.RRect, color color.NRGBA) layout.Dimensions {
	defer rect.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: color}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: image.Pt(int(rect.Rect.Max.X), int(rect.Rect.Max.Y))}
}

func colorRect(gtx layout.Context, rect clip.Rect, color color.NRGBA) layout.Dimensions {
	defer rect.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: color}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: rect.Max}
}

func colorBox(gtx layout.Context, size image.Point, color color.NRGBA) layout.Dimensions {
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: color}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: size}
}

func fill(gtx layout.Context, backgroundColor color.NRGBA, w layout.Widget) layout.Dimensions {
	colorBox(gtx, gtx.Constraints.Max, backgroundColor)
	return layout.NW.Layout(gtx, w)
}
