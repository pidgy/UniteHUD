package decorate

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/nrgba"
)

func CheckBox(c *material.CheckBoxStyle) {
	c.Color = config.Current.Theme.Foreground
	c.IconColor = config.Current.Theme.Foreground
}

func ColorBox(gtx layout.Context, size image.Point, n nrgba.NRGBA) layout.Dimensions {
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: n.Color()}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: size}
}

func Fill(gtx layout.Context, bg nrgba.NRGBA, w layout.Widget) layout.Dimensions {
	ColorBox(gtx, gtx.Constraints.Max, bg)
	return layout.NW.Layout(gtx, w)
}

func Foreground(n *nrgba.NRGBA) {
	*n = nrgba.NRGBA(config.Current.Theme.Foreground)
}

func Label(l *material.LabelStyle, format string, a ...interface{}) {
	l.Text = format
	if len(a) > 0 {
		l.Text = fmt.Sprintf(format, a...)
	}
	l.Color = config.Current.Theme.Foreground
}

func LabelColor(l *material.LabelStyle, c color.NRGBA) {
	l.Color = c
}

func Line(gtx layout.Context, rect clip.Rect, n nrgba.NRGBA) layout.Dimensions {
	defer rect.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: n.Color()}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return ColorBox(gtx, gtx.Constraints.Max, n)
}
