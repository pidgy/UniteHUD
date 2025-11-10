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

	"github.com/pidgy/unitehud/av/img/splash"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/gui/cursor"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/ux/decorate"
	"github.com/pidgy/unitehud/system/tray"
)

type loading struct {
	message string
	tick    <-chan time.Time
	ops     op.Ops
}

func (g *GUI) loading() {
	ui := &loading{
		message: "Loading...",
		tick:    time.NewTicker(time.Millisecond * 250).C,
	}

	go ui.while()

	width := 720
	height := 440

	g.window.Option(
		app.Title(exe.Title),
		app.Size(unit.Dp(width), unit.Dp(height)),
		// app.MaxSize(unit.Dp(width), unit.Dp(height)),
		app.MinSize(unit.Dp(width), unit.Dp(height)),
		app.WindowMode.Option(app.Windowed),
		app.Decorated(false),
	)

	cursor.Is(pointer.CursorProgress)

	dims := layout.Dimensions{}
	inset := layout.Inset{}

	messageLabel := material.Label(g.nav.Calibri().Theme, unit.Sp(18.5), ui.message)
	messageLabel.Alignment = text.Middle
	messageLabel.Font.Weight = 50

	g.window.Perform(system.ActionCenter)
	g.window.Perform(system.ActionRaise)

	for is.Now == is.Loading {
		switch event := g.window.NextEvent().(type) {
		case app.ViewEvent:
			if event.HWND != 0 {
				g.HWND = event.HWND
				tray.SetHWND(g.HWND)
			}
		case system.DestroyEvent:
			g.next(is.Closing)
			return
		case system.FrameEvent:
			gtx := layout.NewContext(&ui.ops, event)
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
						messageLabel.Text = ui.message
						return messageLabel.Layout(gtx)
					})
				})
			})

			g.frame(gtx, event)
		default:
			notify.Missed(event, "Loading")
		}
	}
}

func (l *loading) while() {
	i := 0

	for ; is.Now == is.Loading; <-l.tick {
		l.message, i = notify.Iter(i)
	}
}
