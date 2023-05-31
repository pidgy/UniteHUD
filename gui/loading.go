package gui

import (
	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/cursor"
	"github.com/pidgy/unitehud/splash"
)

func (g *GUI) loading() {
	w := app.NewWindow(
		app.Title(config.ProjectorWindow),
		app.Size(unit.Dp(720), unit.Dp(480)),
		app.WindowMode.Option(app.Windowed),
		app.Decorated(false),
	)

	w.Perform(system.ActionCenter)
	w.Perform(system.ActionRaise)

	var ops op.Ops

	cursor.Is(pointer.CursorProgress)

	for e := range w.Events() {
		select {
		case <-g.readyq:
			w.Perform(system.ActionClose)
		default:
		}

		switch e := e.(type) {
		case app.ViewEvent:
		case system.DestroyEvent:
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)

			cursor.Draw(gtx)

			widget.Image{
				Src:   paint.NewImageOp(splash.Loading()),
				Scale: float32(splash.Loading().Bounds().Dx()) / float32(gtx.Constraints.Max.X),
				Fit:   widget.Cover,
			}.Layout(gtx)

			w.Invalidate()

			e.Frame(gtx.Ops)
		case app.ConfigEvent:
			println(e.Config.Size.String())
		case key.Event:

		case pointer.Event:

		case system.StageEvent:
		default:
		}
	}
}
