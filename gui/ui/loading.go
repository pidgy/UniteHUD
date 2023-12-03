package ui

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

	"github.com/pidgy/unitehud/core/global"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/nrgba"
	"github.com/pidgy/unitehud/gui/cursor"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/visual/decorate"
	"github.com/pidgy/unitehud/media/img/splash"
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

	g.window.Option(
		app.Title(global.Title),
		app.Size(unit.Dp(width), unit.Dp(height)),
		app.MaxSize(unit.Dp(width), unit.Dp(height)),
		app.MinSize(unit.Dp(width), unit.Dp(height)),
		app.WindowMode.Option(app.Windowed),
		app.Decorated(false),
	)

	cursor.Is(pointer.CursorProgress)

	dims := layout.Dimensions{}
	inset := layout.Inset{}

	messageLabel := material.Label(g.header.Collection.Calibri().Theme, unit.Sp(18.5), l.message)
	messageLabel.Alignment = text.Middle
	messageLabel.Font.Weight = 50

	g.window.Perform(system.ActionCenter)
	g.window.Perform(system.ActionRaise)

	var ops op.Ops

	for is.Now == is.Loading {
		switch event := (<-g.window.Events()).(type) {
		case app.ViewEvent:
			g.HWND = event.HWND
		case system.DestroyEvent:
			g.next(is.Closing)
			return
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, event)
			op.InvalidateOp{}.Add(gtx.Ops)

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

			g.frame(gtx, event)

			g.window.Invalidate()
		default:
			notify.Missed(event, "Loading")
		}
	}
}

func (l *loading) while(g *GUI) {
	i := 0

	for ; is.Now == is.Loading; <-l.tick {
		l.message, i = notify.Iter(i)
	}
}
