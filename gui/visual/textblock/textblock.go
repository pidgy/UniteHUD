package textblock

import (
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/fonts"
	"github.com/pidgy/unitehud/gui/visual/decor"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/nrgba"
)

type TextBlock struct {
	Text       string
	list       *widget.List
	font       *fonts.Style
	style      material.ListStyle
	labelStyle material.LabelStyle
}

func New(s *fonts.Style) (*TextBlock, error) {
	return &TextBlock{
		font: s,
	}, nil
}

func (t *TextBlock) Layout(gtx layout.Context, posts []notify.Post) layout.Dimensions {
	if t.list == nil {
		t.list = &widget.List{
			Scrollbar: widget.Scrollbar{},
			List: layout.List{
				Axis:        layout.Vertical,
				ScrollToEnd: true,
				Alignment:   layout.Baseline,
			},
		}

		t.font.Theme.TextSize = unit.Sp(9)

		t.style = material.List(t.font.Theme, t.list)
		t.style.Track.Color = nrgba.Slate.Color()
		t.style.Track.Color.A = 0x0F

		t.labelStyle = material.H5(t.font.Theme, "")
	}

	decor.Fill(gtx,
		nrgba.Background,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: gtx.Constraints.Max}
		},
	)

	layout.Inset{
		Bottom: unit.Dp(5),
		Left:   unit.Dp(5),
	}.Layout(
		gtx,
		func(gtx layout.Context) layout.Dimensions {
			return t.style.Layout(
				gtx,
				len(posts),
				func(gtx layout.Context, index int) layout.Dimensions {
					t.labelStyle.Text = posts[index].String()
					t.labelStyle.Color = posts[index].Color()
					t.labelStyle.Alignment = text.Alignment(text.Start)
					dim := t.labelStyle.Layout(gtx)
					dim.Size.X = gtx.Constraints.Max.X
					return dim
				},
			)
		},
	)
	//

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
