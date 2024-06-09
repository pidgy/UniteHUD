package checklist

import (
	"image"

	"gioui.org/font"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/gui/cursor"
	"github.com/pidgy/unitehud/gui/ux/decorate"
)

type Item struct {
	Text     string
	Hint     string
	Checked  widget.Bool
	Value    int
	Disabled bool
	Weight   int

	Callback         func(this *Item)
	DisabledCallback func(this *Item)

	check material.CheckBoxStyle

	hovered bool
}

type Widget struct {
	Items         []*Item
	Callback      func(item *Item, this *Widget) (check bool)
	WidthModifier int
	Radio         bool
	TextSize      float32
	Theme         *material.Theme

	liststyle material.ListStyle
}

func (item *Item) hint(gtx layout.Context, theme *material.Theme) (layout.Dimensions, bool) {
	if item.Hint == "" {
		return layout.Dimensions{}, false
	}

	label := material.Label(
		theme,
		item.check.TextSize*unit.Sp(.9),
		item.Hint,
	)
	label.Color = nrgba.Transparent80.Color()

	return label.Layout(gtx), true
}

func (l *Widget) Default() *Item {
	if len(l.Items) == 0 {
		return &Item{}
	}
	return l.Items[0]
}

// Layout handles drawing the letters view.
func (list *Widget) Layout(gtx layout.Context) layout.Dimensions {
	list.defaultList()

	return list.liststyle.Layout(gtx, len(list.Items),
		func(gtx layout.Context, index int) layout.Dimensions {
			item := list.Items[index]
			list.defaultCheckBox(item)

			if !item.Checked.Update(gtx) {
				return list.draw(gtx, item)
			}

			if item.Disabled {
				item.Checked.Value = false
				item.Callback(item)

				return list.draw(gtx, item)
			}

			item.Callback(item)
			list.Callback(item, list)

			list.radio(item)

			return list.draw(gtx, item)
		},
	)
}

func (list *Widget) defaultCheckBox(i *Item) {
	defer decorate.CheckBox(&i.check)

	if i.check.CheckBox != nil {
		return
	}

	if i.Callback == nil {
		i.Callback = func(this *Item) {}
	}

	if i.DisabledCallback == nil {
		i.DisabledCallback = func(this *Item) { i.Checked.Value = false }
	}

	i.check = material.CheckBox(list.Theme, &i.Checked, i.Text)
	i.check.Size = unit.Dp(list.TextSize)
	i.check.TextSize = unit.Sp(list.TextSize)
	i.check.Font.Weight = font.ExtraBold
	if list.TextSize == 0 {
		i.check.Size = unit.Dp(12)
		i.check.TextSize = unit.Sp(12)
	}
}

func (list *Widget) defaultList() {
	defer decorate.Scrollbar(&list.liststyle.ScrollbarStyle)
	defer decorate.List(&list.liststyle)

	if list.liststyle.Scrollbar != nil {
		return
	}

	cb := list.Callback
	list.Callback = func(item *Item, this *Widget) (check bool) {
		if cb == nil {
			return false
		}
		item.Checked.Value = cb(item, this)
		return false
	}

	list.liststyle = material.List(
		list.Theme,
		&widget.List{
			Scrollbar: widget.Scrollbar{},
			List: layout.List{
				Axis:      layout.Vertical,
				Alignment: layout.Start,
			},
		},
	)
	list.liststyle.AnchorStrategy = material.Overlay
	list.liststyle.Track.MajorPadding = unit.Dp(1)
	list.liststyle.Track.MinorPadding = unit.Dp(1)

	if list.WidthModifier == 0 {
		list.WidthModifier = 1
	}
}

func (list *Widget) draw(gtx layout.Context, item *Item) layout.Dimensions {
	// list.liststyle.Scrollbar.AddTrack(gtx.Ops)

	if list.liststyle.Scrollbar.IndicatorHovered() || list.liststyle.Scrollbar.TrackHovered() {
		list.liststyle.Scrollbar.AddDrag(gtx.Ops)
		cursor.Is(pointer.CursorPointer)
	}

	if item.Checked.Value {
		if item.Text == "Disabled" {
			item.check.Color = nrgba.PastelRed.Color()
		} else {
			item.check.Color = nrgba.DarkSeafoam.Color()
		}
	}

	switch {
	case item.Checked.Hovered(): //, item.Checked.Focused():
		list.hovered(gtx, item)
	default:
		list.unhovered(item)
	}

	d, ok := item.hint(gtx, list.Theme)
	if ok {
		return d
	}

	return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		dim := item.check.Layout(gtx)
		dim.Size.X = gtx.Constraints.Max.X / list.WidthModifier
		// dim.Size.Y -= 10
		return dim
	})
}

func (list *Widget) hovered(gtx layout.Context, i *Item) {
	i.hovered = true
	decorate.ColorBox(gtx, image.Pt(gtx.Constraints.Max.X, 20), nrgba.White.Alpha(5))
	cursor.Is(pointer.CursorPointer)
}

func (list *Widget) radio(item *Item) {
	if !list.Radio {
		return
	}

	for _, i := range list.Items {
		i.Checked.Value = false
		if i == item {
			i.Checked.Value = true
		}
	}
}

func (list *Widget) unhovered(i *Item) {
	if i.hovered {
		cursor.Is(pointer.CursorDefault)
	}
	i.hovered = false
}
