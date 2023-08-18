package dropdown

import (
	"image"

	"gioui.org/font"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/cursor"
	"github.com/pidgy/unitehud/nrgba"
)

var (
	Disabled = nrgba.Red.Color()
	Enabled  = nrgba.DarkSeafoam.Color()
)

type List struct {
	Items         []*Item
	Callback      func(*Item, *List)
	WidthModifier int
	Radio         bool
	TextSize      float32

	list *widget.List
}

type Item struct {
	Text     string
	Hint     string
	Checked  widget.Bool
	Value    int
	Disabled bool
	Weight   int

	Callback func(*Item)
}

func (l *List) Checked() *Item {
	for _, item := range l.Items {
		if item.Checked.Value {
			return item
		}
	}
	return nil
}

func (l *List) Default() *Item {
	if len(l.Items) == 0 {
		return &Item{}
	}
	return l.Items[0]
}

func (l *List) Disable() {
	for _, item := range l.Items {
		item.Checked.Value = false
		if item.Text == "Disabled" {
			item.Checked.Value = true
		}
	}
}

func (l *List) Disabled() {
	for _, item := range l.Items {
		if item.Text == "Disabled" {
			item.Checked.Value = true
			return
		}
	}
}

func (l *List) Enabled() {
	for _, item := range l.Items {
		if item.Text == "Disabled" {
			item.Checked.Value = false
			return
		}
	}
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
	style.Track.Color = nrgba.Gray.Color()
	style.Track.Color.A = 0xFF

	return style.Layout(gtx, len(l.Items), func(gtx layout.Context, index int) layout.Dimensions {
		item := l.Items[index]

		check := material.CheckBox(th, &item.Checked, item.Text)
		check.Font.Weight = font.Weight(item.Weight)
		check.Color = nrgba.White.Color()
		check.Size = unit.Dp(l.TextSize)
		check.TextSize = unit.Sp(l.TextSize)
		if l.TextSize == 0 {
			check.Size = unit.Dp(14)
			check.TextSize = unit.Sp(14)
		}
		check.IconColor = nrgba.White.Alpha(50).Color()

		if item.Checked.Changed() {
			if item.Disabled {
				item.Checked.Value = !item.Checked.Value
			} else if item.Callback != nil {
				item.Callback(item)
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
				l.Callback(item, l)
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
		case item.Checked.Hovered(), item.Checked.Focused():
			hoverItem(gtx, index)
		case item.Checked.Value:
			selectedItem(gtx, index)
		}

		if l.WidthModifier == 0 {
			l.WidthModifier = 1
		}

		dims := layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			dim := check.Layout(gtx)
			dim.Size.X = gtx.Constraints.Max.X / l.WidthModifier
			return dim
		})

		if item.Hint != "" {
			l := material.Label(
				th,
				check.TextSize*unit.Sp(.9),
				item.Hint,
			)
			l.Color = nrgba.Transparent30.Color()

			dims = l.Layout(gtx)
		}

		return dims
	})
}

func selectedItem(gtx layout.Context, index int) {
	cursor.Is(pointer.CursorDefault)

	widget.Border{
		Color:        nrgba.White.Alpha(5).Color(),
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(3),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
			nrgba.Black.Alpha(50),
		)
	})
}

func hoverItem(gtx layout.Context, index int) {
	cursor.Is(pointer.CursorPointer)

	colorRect(gtx,
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
		nrgba.White.Alpha(5),
	)
}

func colorRect(gtx layout.Context, rect clip.Rect, nrgba nrgba.NRGBA) layout.Dimensions {
	defer rect.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: nrgba.Color()}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: rect.Max}
}
