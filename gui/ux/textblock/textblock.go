package textblock

import (
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/core/fonts"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/gui/cursor"
	"github.com/pidgy/unitehud/gui/ux/decorate"
)

type Widget struct {
	Text string
	font *fonts.Style

	max int

	list  material.ListStyle
	label material.LabelStyle

	dragged bool
}

func New(s *fonts.Style, max int) (*Widget, error) {
	t := &Widget{
		max:  max,
		font: s,
		list: material.List(
			s.Theme,
			&widget.List{
				Scrollbar: widget.Scrollbar{},
				List: layout.List{
					Axis:        layout.Vertical,
					ScrollToEnd: true,
					Alignment:   layout.Baseline,
				},
			},
		),
	}

	t.font.Theme.TextSize = unit.Sp(8)
	t.label = material.H5(t.font.Theme, "")
	t.list.Track.MinorPadding = 0

	return t, nil
}

func (t *Widget) Layout(gtx layout.Context, posts []notify.Post) layout.Dimensions {
	defer t.cursor()

	if len(posts) > t.max {
		posts = posts[len(posts)-t.max:]
	}

	return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
		decorate.Border(gtx)
		decorate.Scrollbar(&t.list.ScrollbarStyle)
		decorate.List(&t.list)

		layout.Inset{
			Bottom: unit.Dp(5),
			Left:   unit.Dp(5),
		}.Layout(
			gtx,
			func(gtx layout.Context) layout.Dimensions {
				return t.list.Layout(gtx, len(posts), func(gtx layout.Context, index int) layout.Dimensions {
					t.label.Text = posts[index].String()
					t.label.Color = posts[index].Alpha(t.calculateAlpha(index, len(posts))).Color()
					t.label.Alignment = text.Start
					dim := t.label.Layout(gtx)
					dim.Size.X = gtx.Constraints.Max.X
					return dim
				})
			},
		)

		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
}

func (t *Widget) calculateAlpha(index, nposts int) uint8 {
	ratio := (float32(index) / float32(nposts))
	// if ratio < 0.1 {
	// 	ratio = 0.1
	// }
	a := (float32(257) * ratio) + 5
	if a > 255 {
		a = 255
	}
	return uint8(a)
}

func (t *Widget) cursor() {
	switch {
	case t.list.Scrollbar.Dragging():
		cursor.Is(pointer.CursorNorthSouthResize)
		t.dragged = true
	case t.list.Scrollbar.TrackHovered():
		cursor.Is(pointer.CursorPointer)
		t.dragged = true
	default:
		if t.dragged {
			t.dragged = false
			cursor.Is(pointer.CursorDefault)
		}
	}
}
