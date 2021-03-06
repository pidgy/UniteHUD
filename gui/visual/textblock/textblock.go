package textblock

import (
	"image"
	"image/color"

	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
)

type TextBlock struct {
	Text string
	list *widget.List
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
	}

	th := material.NewTheme(gofont.Collection())
	th.TextSize = unit.Sp(9)
	Fill(gtx,
		rgba.N(rgba.DarkGray),
		func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: gtx.Constraints.Max}
		},
	)

	list := material.List(th, t.list)
	list.Track.Color = color.NRGBA(rgba.Purple)
	list.Track.Color.A = 0xF
	layout.Inset{
		Bottom: unit.Px(5),
		Left:   unit.Px(5),
	}.Layout(
		gtx,
		func(gtx layout.Context) layout.Dimensions {
			return list.Layout(
				gtx,
				len(posts),
				func(gtx layout.Context, index int) layout.Dimensions {
					block := material.H5(th, posts[index].String())
					block.Color = color.NRGBA(posts[index].RGBA)
					block.Alignment = text.Alignment(text.Start)
					dim := block.Layout(gtx)
					dim.Size.X = gtx.Constraints.Max.X
					return dim
				},
			)
		},
	)

	return layout.Dimensions{Size: gtx.Constraints.Max}
}

// colorBox creates a widget with the specified dimensions and color.
func colorBox(gtx layout.Context, size image.Point, c color.NRGBA) layout.Dimensions {
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: c}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	return widget.Border{
		Color: color.NRGBA{R: 100, G: 100, B: 100, A: 50},
		Width: unit.Px(2),
	}.Layout(
		gtx,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: size}
		})
}

func Fill(gtx layout.Context, backgroundColor color.NRGBA, w layout.Widget) layout.Dimensions {
	colorBox(gtx, gtx.Constraints.Max, backgroundColor)
	return layout.NW.Layout(gtx, w)
}
