package decorate

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/core/server"
)

// Background fills the full context with the themed background color.
func Background(gtx layout.Context) {
	ColorBox(gtx, gtx.Constraints.Max, nrgba.NRGBA(config.Current.Theme.Background))
}

// BackgroundAlt fills the full context with the themed alternate background and lays out w.
func BackgroundAlt(gtx layout.Context, w layout.Widget) layout.Dimensions {
	ColorBox(gtx, gtx.Constraints.Max, nrgba.NRGBA(config.Current.Theme.BackgroundAlt))
	return layout.NW.Layout(gtx, w)
}

// BackgroundColor fills the full context with a specific color.
func BackgroundColor(gtx layout.Context, n nrgba.NRGBA) {
	ColorBox(gtx, gtx.Constraints.Max, n)
}

// BackgroundTitleBar fills the title bar area with the themed title bar color.
func BackgroundTitleBar(gtx layout.Context, size image.Point) {
	ColorBox(gtx, size, nrgba.NRGBA(config.Current.Theme.TitleBarBackground))
}

// Border draws a one-pixel border using the active/idle theme based on server readiness.
func Border(gtx layout.Context) layout.Dimensions {
	if server.Ready() {
		return ColorBox(gtx, image.Pt(gtx.Constraints.Max.X, 1), nrgba.NRGBA(config.Current.Theme.BordersActive).Alpha(255))
	}
	return ColorBox(gtx, image.Pt(gtx.Constraints.Max.X, 1), nrgba.NRGBA(config.Current.Theme.BordersIdle).Alpha(255))
}

// BorderIdle draws a one-pixel border using the idle theme color.
func BorderIdle(gtx layout.Context) layout.Dimensions {
	return ColorBox(gtx, image.Pt(gtx.Constraints.Max.X, 1), nrgba.NRGBA(config.Current.Theme.BordersIdle).Alpha(255))
}

// CheckBox applies theme colors to a checkbox style.
func CheckBox(c *material.CheckBoxStyle) {
	c.Color = config.Current.Theme.Foreground
	c.IconColor = config.Current.Theme.Foreground
}

// ColorBox paints a solid rectangle of the given size and color.
func ColorBox(gtx layout.Context, size image.Point, n nrgba.NRGBA) layout.Dimensions {
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: n.Color()}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: size}
}

// Fill paints a background color and lays out w on top.
func Fill(gtx layout.Context, bg nrgba.NRGBA, w layout.Widget) layout.Dimensions {
	ColorBox(gtx, gtx.Constraints.Max, bg)
	return layout.NW.Layout(gtx, w)
}

// Foreground sets n to the themed foreground color.
func Foreground(n *nrgba.NRGBA) {
	*n = nrgba.NRGBA(config.Current.Theme.Foreground)
}

// ForegroundAlt sets n to the themed alternate foreground color.
func ForegroundAlt(n *color.NRGBA) {
	*n = config.Current.Theme.ForegroundAlt
}

// Label formats label text and applies the themed foreground color.
func Label(l *material.LabelStyle, format string, a ...any) *material.LabelStyle {
	l.Text = format
	if len(a) > 0 {
		l.Text = fmt.Sprintf(format, a...)
	}
	l.Color = config.Current.Theme.Foreground
	return l
}

// LabelAlpha sets the label alpha.
func LabelAlpha(l *material.LabelStyle, a uint8) *material.LabelStyle {
	l.Color.A = a
	return l
}

// LabelColor sets the label color.
func LabelColor(l *material.LabelStyle, c color.NRGBA) *material.LabelStyle {
	l.Color = c
	return l
}

// Line draws a filled rectangle for a line using the provided rect and color.
func Line(gtx layout.Context, rect clip.Rect, n nrgba.NRGBA) layout.Dimensions {
	defer rect.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: n.Color()}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return ColorBox(gtx, gtx.Constraints.Max, n)
}

// List applies list styling defaults.
func List(l *material.ListStyle) {
	l.AnchorStrategy = material.Overlay
}

// Scrollbar applies themed colors to a scrollbar style.
func Scrollbar(s *material.ScrollbarStyle) {
	s.Track.Color = config.Current.Theme.ScrollbarBackground
	s.Indicator.Color = config.Current.Theme.ScrollbarForeground
	s.Indicator.HoverColor = nrgba.NRGBA(config.Current.Theme.ScrollbarForeground).Alpha(15).Color()
}

// Spacer draws a subtle themed spacer block.
func Spacer(gtx layout.Context, size image.Point) layout.Dimensions {
	return ColorBox(gtx, size, nrgba.NRGBA(config.Current.Theme.BordersIdle).Alpha(25))
}

// Underline renders w and draws a themed underline beneath it.
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

// UnderlineBorder renders w and draws a border-colored underline beneath it.
func UnderlineBorder(gtx layout.Context, w layout.Widget) layout.Dimensions {
	dims := w(gtx)

	paint.FillShape(gtx.Ops,
		nrgba.NRGBA(config.Current.Theme.BordersIdle).Alpha(127).Color(),
		clip.Stroke{
			Path:  clip.UniformRRect(image.Rect(0, dims.Size.Y, dims.Size.X, dims.Size.Y), 0).Path(gtx.Ops),
			Width: .5,
		}.Op(),
	)

	return dims
}
