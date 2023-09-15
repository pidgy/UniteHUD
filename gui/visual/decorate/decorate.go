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

func Background(gtx layout.Context) {
	ColorBox(gtx, gtx.Constraints.Max, nrgba.NRGBA(config.Current.Theme.Background))
}

func BackgroundAlt(gtx layout.Context, w layout.Widget) layout.Dimensions {
	ColorBox(gtx, gtx.Constraints.Max, nrgba.NRGBA(config.Current.Theme.BackgroundAlt))
	return layout.NW.Layout(gtx, w)
}

func BackgroundTitleBar(gtx layout.Context, size image.Point) {
	ColorBox(gtx, size, nrgba.NRGBA(config.Current.Theme.TitleBarBackground))
}

func Border(gtx layout.Context) layout.Dimensions {
	return ColorBox(gtx, image.Pt(gtx.Constraints.Max.X, 1), nrgba.NRGBA(config.Current.Theme.Borders))
}

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

func ForegroundAlt(n *color.NRGBA) {
	*n = config.Current.Theme.ForegroundAlt
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

func List(l *material.ListStyle) {
	l.AnchorStrategy = material.Overlay
}

func Scrollbar(s *material.ScrollbarStyle) {
	s.Track.Color = config.Current.Theme.ScrollbarBackground
	s.Indicator.Color = config.Current.Theme.ScrollbarForeground
	s.Indicator.HoverColor = nrgba.NRGBA(config.Current.Theme.ScrollbarForeground).Alpha(15).Color()
}

func Spacer(gtx layout.Context, size image.Point) layout.Dimensions {
	return ColorBox(gtx, size, nrgba.NRGBA(config.Current.Theme.Borders).Alpha(25))
}

func Underline(gtx layout.Context, w layout.Widget) layout.Dimensions {
	dims := w(gtx)

	paint.FillShape(gtx.Ops,
		nrgba.NRGBA(config.Current.Theme.Foreground).Alpha(127).Color(),
		clip.Stroke{
			Path:  clip.UniformRRect(image.Rect(0, dims.Size.Y, dims.Size.X, dims.Size.Y), 0).Path(gtx.Ops),
			Width: .5,
		}.Op(),
	)

	return dims
}
