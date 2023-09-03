package gui

import (
	"time"

	"gioui.org/app"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/cursor"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/visual/decorate"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/nrgba"
	"github.com/pidgy/unitehud/splash"
)

type loading struct {
	message string
	tick    <-chan time.Time
}

func (g *GUI) loading() {
	l := &loading{
		message: "Loading...",
		tick:    time.NewTicker(time.Millisecond * 250).C,
	}

	go l.while(g)

	width := 720
	height := 440

	g.Window.Option(
		app.Title("UniteHUD"),
		app.Size(unit.Dp(width), unit.Dp(height)),
		app.MaxSize(unit.Dp(width), unit.Dp(height)),
		app.MinSize(unit.Dp(width), unit.Dp(height)),
		app.WindowMode.Option(app.Windowed),
		app.Decorated(false),
	)

	cursor.Is(pointer.CursorProgress)

	dims := layout.Dimensions{}
	inset := layout.Inset{}

	messageLabel := material.Label(g.Bar.Collection.Calibri().Theme, unit.Sp(18.5), l.message)
	messageLabel.Alignment = text.Middle
	messageLabel.Font.Weight = 50

	g.Window.Perform(system.ActionCenter)
	g.Window.Perform(system.ActionRaise)

	var ops op.Ops

	for g.is == is.Loading {
		switch e := (<-g.Events()).(type) {
		case app.ViewEvent:
			g.HWND = e.HWND
		case system.DestroyEvent:
			g.next(is.Closing)
			return
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)

			cursor.Draw(gtx)

			layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx layout.Context) layout.Dimensions {
					return widget.Image{
						Src:   paint.NewImageOp(splash.Loading()),
						Scale: float32(splash.Loading().Bounds().Dx()) / float32(gtx.Constraints.Max.X),
						Fit:   widget.Cover,
					}.Layout(gtx)
				}),
			)

			if dims.Size.X == 0 {
				dims = messageLabel.Layout(gtx)
				decorate.LabelColor(&messageLabel, nrgba.White.Color())
				x := unit.Dp((float64(gtx.Constraints.Max.X) - float64(dims.Size.X)) / 2)
				y := unit.Dp((float64(gtx.Constraints.Max.Y) - float64(dims.Size.Y)))
				inset = layout.Inset{Left: x, Right: x, Top: y, Bottom: x}
			}

			layout.S.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(25)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {

					return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						messageLabel.Text = l.message
						return messageLabel.Layout(gtx)
					})
				})
			})

			g.frame(gtx, e)

			g.Window.Invalidate()
		default:
			notify.Debug("Event missed: %T (Loading Window)", e)
		}
	}
}

func (l *loading) while(g *GUI) {
	i := 0

	for ; g.is == is.Loading; <-l.tick {
		l.message, i = notify.Iter(i)
	}
}
