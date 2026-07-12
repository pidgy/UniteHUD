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

// Item represents a single checklist row with state and callbacks.
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

	hintLabel material.LabelStyle
}

// Widget renders a checklist with optional radio behavior.
type Widget struct {
	Items         []*Item
	Callback      func(item *Item, this *Widget)
	WidthModifier int
	Radio         bool
	TextSize      float32
	Theme         *material.Theme

	liststyle material.ListStyle
}

// hint lays out an optional hint label under the item.
func (item *Item) hint(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	if item.Hint == "" {
		return layout.Dimensions{}
	}

	if item.hintLabel.Text == "" {
		item.hintLabel = material.Label(
			theme,
			item.check.TextSize*.9,
			item.Hint,
		)
		item.hintLabel.Color = theme.ContrastFg
		item.hintLabel.Font.Style = font.Italic
		item.hintLabel.Font.Weight = font.Light
	}

	return layout.Inset{Top: 20, Left: item.check.Size * 1.5, Bottom: -20}.Layout(gtx, item.hintLabel.Layout)
}

// Default returns the first item or an empty placeholder.
func (l *Widget) Default() *Item {
	if len(l.Items) == 0 {
		return &Item{}
	}
	return l.Items[0]
}

// Layout renders the checklist and handles item interactions.
func (list *Widget) Layout(gtx layout.Context) layout.Dimensions {
	list.defaultChecklist()

	return list.liststyle.Layout(gtx, len(list.Items),
		func(gtx layout.Context, index int) layout.Dimensions {
			item := list.Items[index]
			list.defaultCheckBox(item)

			switch {
			case !item.Checked.Update(gtx):
			case item.Disabled:
				item.DisabledCallback(item)
				item.Checked.Value = !item.Checked.Value
			default:
				item.Callback(item)
				list.Callback(item, list)
				list.radio(item)
			}

			return list.draw(gtx, item)
		},
	)
}

// defaultCheckBox initializes checkbox styles and callbacks.
func (list *Widget) defaultCheckBox(i *Item) {
	defer decorate.CheckBox(&i.check)

	if i.check.CheckBox != nil {
		return
	}

	if i.Callback == nil {
		i.Callback = func(this *Item) {}
	}

	if i.DisabledCallback == nil {
		i.DisabledCallback = func(this *Item) {}
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

// defaultChecklist initializes the list style and selection behavior.
func (list *Widget) defaultChecklist() {
	defer decorate.Scrollbar(&list.liststyle.ScrollbarStyle)
	defer decorate.List(&list.liststyle)

	if list.liststyle.Scrollbar != nil {
		return
	}

	cb := list.Callback
	list.Callback = func(item *Item, this *Widget) {
		if item.Disabled {
			item.DisabledCallback(item)
			return
		}

		item.Callback(item)

		if cb != nil {
			cb(item, this)
		}
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

// draw renders a single checklist row.
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

	if item.Disabled {
		item.check.Color.A = 50
	}

	switch {
	case item.Checked.Hovered(): //, item.Checked.Focused():
		list.hovered(gtx, item)
	default:
		list.unhovered(item)
	}

	checkDims := layout.Dimensions{}
	lineDims := layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		checkDims = item.check.Layout(gtx)
		checkDims.Size.X = gtx.Constraints.Max.X / list.WidthModifier
		// dim.Size.Y -= 10
		return checkDims
	})
	hintDims := item.hint(gtx, list.Theme)

	return layout.Dimensions{Size: lineDims.Size.Add(hintDims.Size)}
}

// hovered applies hover styling to the item row.
func (list *Widget) hovered(gtx layout.Context, i *Item) {
	i.hovered = true
	decorate.ColorBox(gtx, image.Pt(gtx.Constraints.Max.X, 20), nrgba.White.Alpha(5))
	cursor.Is(pointer.CursorPointer)
}

// radio enforces single selection when enabled.
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

// unhovered clears hover state and cursor hints.
func (list *Widget) unhovered(i *Item) {
	if i.hovered {
		cursor.Is(pointer.CursorDefault)
	}
	i.hovered = false
}
