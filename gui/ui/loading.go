package ui

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const.

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

	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/gui/cursor"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/ux/decorate"
	"github.com/pidgy/unitehud/media/img/splash"
	"github.com/pidgy/unitehud/system/tray"
)

// loading defines loading behavior and state.
type loading struct {
	message string
	tick    <-chan time.Time
	ops     op.Ops
}

// loading shows the splash loading window and processes its UI events.
func (g *GUI) loading() {
	is.Next(is.Loading)

	ui := &loading{
		message: "Loading...",
		tick:    time.NewTicker(time.Millisecond * 250).C,
	}

	go ui.while()

	width := 720
	height := 440

	g.window.Option(
		app.Size(unit.Dp(width), unit.Dp(height)),
		app.MinSize(unit.Dp(width), unit.Dp(height)),
		// app.MaxSize(unit.Dp(width), unit.Dp(height)),
		// app.MinSize(unit.Dp(width), unit.Dp(height)),
	)

	cursor.Is(pointer.CursorProgress)

	dims := layout.Dimensions{}
	inset := layout.Inset{}

	messageLabel := material.Label(g.nav.Calibri().Theme, unit.Sp(18.5), ui.message)
	messageLabel.Alignment = text.Middle
	messageLabel.Font.Weight = 50

	g.window.Perform(system.ActionCenter)
	g.window.Perform(system.ActionRaise)

	img := widget.Image{
		Src:   paint.NewImageOp(splash.Loading()),
		Scale: .9,
	}

	for is.Currently(is.Loading) {
		switch event := g.window.NextEvent().(type) {
		case app.ViewEvent:
			if event.HWND != 0 {
				g.HWND = event.HWND
				tray.SetHWND(g.HWND)
			}
		case system.DestroyEvent:
			is.Next(is.Closing)
			return
		case system.FrameEvent:
			gtx := layout.NewContext(&ui.ops, event)

			decorate.BackgroundColor(gtx, nrgba.Splash)

			cursor.Draw(gtx)

			layout.Center.Layout(gtx, img.Layout)

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

			g.window.Perform(system.ActionCenter)
			g.window.Perform(system.ActionRaise)
		default:
			notify.Missed(event, "Loading")
		}
	}
}

// while updates the loading message on each tick while loading is active.
func (l *loading) while() {
	i := 0

	for ; is.Currently(is.Loading); <-l.tick {
		l.message, i = notify.Iter(i)
	}
}
