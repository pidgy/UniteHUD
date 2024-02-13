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

	"github.com/pidgy/unitehud/core/nrgba"
	"github.com/pidgy/unitehud/gui/cursor"
	"github.com/pidgy/unitehud/gui/ux/decorate"
)

var (
	Disabled = nrgba.PastelRed.Color()
	Enabled  = nrgba.PastelGreen.Color()
)

type Widget struct {
	Items         []*Item
	Callback      func(item *Item, this *Widget) bool
	WidthModifier int
	Radio         bool
	TextSize      float32
	Theme         *material.Theme

	liststyle material.ListStyle
}

type Item struct {
	Text     string
	Hint     string
	Checked  widget.Bool
	Value    int
	Disabled bool
	Weight   int

	Callback func(this *Item)

	hovered bool
}

func (l *Widget) Checked() *Item {
	for _, item := range l.Items {
		if item.Checked.Value {
			return item
		}
	}
	return nil
}

func (l *Widget) Default() *Item {
	if len(l.Items) == 0 {
		return &Item{}
	}
	return l.Items[0]
}

func (l *Widget) Disable() {
	for _, item := range l.Items {
		item.Checked.Value = false
		if item.Text == "Disabled" {
			item.Checked.Value = true
		}
	}
}

func (l *Widget) Disabled() {
	for _, item := range l.Items {
		if item.Text == "Disabled" {
			item.Checked.Value = true
			return
		}
	}
}

func (l *Widget) Enabled() {
	for _, item := range l.Items {
		if item.Text == "Disabled" {
			item.Checked.Value = false
			return
		}
	}
}

// Layout handles drawing the letters view.
func (l *Widget) Layout(gtx layout.Context) layout.Dimensions {
	if l.liststyle.Scrollbar == nil {
		l.liststyle = material.List(l.Theme,
			&widget.List{
				Scrollbar: widget.Scrollbar{},
				List: layout.List{
					Axis:      layout.Vertical,
					Alignment: layout.Start,
				},
			},
		)
		l.liststyle.AnchorStrategy = material.Overlay
		l.liststyle.Track.MajorPadding = unit.Dp(1)
		l.liststyle.Track.MinorPadding = unit.Dp(1)
	}

	decorate.Scrollbar(&l.liststyle.ScrollbarStyle)
	decorate.List(&l.liststyle)

	return l.liststyle.Layout(gtx, len(l.Items),
		func(gtx layout.Context, index int) layout.Dimensions {
			item := l.Items[index]

			check := material.CheckBox(l.Theme, &item.Checked, item.Text)
			check.Size = unit.Dp(l.TextSize)
			check.TextSize = unit.Sp(l.TextSize)
			check.Font.Weight = font.ExtraBold
			if l.TextSize == 0 {
				check.Size = unit.Dp(12)
				check.TextSize = unit.Sp(12)
			}

			decorate.CheckBox(&check)

			if l.liststyle.Scrollbar.IndicatorHovered() || l.liststyle.Scrollbar.TrackHovered() {
				l.liststyle.Scrollbar.AddDrag(gtx.Ops)
				cursor.Is(pointer.CursorPointer)
			}

			if item.Checked.Update(gtx) {
				enabled := !item.Disabled
				if enabled && l.Callback != nil {
					enabled = l.Callback(item, l)
				}
				if !enabled {
					item.Checked.Value = false

					return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						dim := check.Layout(gtx)
						dim.Size.X = gtx.Constraints.Max.X / l.WidthModifier
						return dim
					})
				}

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
			}

			if item.Checked.Value {
				check.Color = nrgba.DarkSeafoam.Color()
				if item.Text == "Disabled" {
					check.Color = Disabled
				}
			}
			if item.Disabled {
				check.Color = Disabled
			}
			switch {
			case item.Checked.Hovered(), item.Checked.Focused():
				l.hovered(gtx, index)

				item.hovered = true
				cursor.Is(pointer.CursorPointer)
			case item.Checked.Value:
				selectedItem(gtx, index)
			case item.hovered:
				item.hovered = false
				cursor.Is(pointer.CursorDefault)
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
				label := material.Label(
					l.Theme,
					check.TextSize*unit.Sp(.9),
					item.Hint,
				)
				label.Color = nrgba.Transparent80.Color()

				dims = label.Layout(gtx)
			}

			return dims
		},
	)
}

func selectedItem(gtx layout.Context, index int) {
	widget.Border{
		Color:        nrgba.White.Alpha(5).Color(),
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(1),
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

func (l *Widget) hovered(gtx layout.Context, index int) {
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
