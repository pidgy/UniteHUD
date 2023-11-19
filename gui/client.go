package gui

import (
	"image"
	"time"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/pidgy/unitehud/fonts"
	"github.com/pidgy/unitehud/fps"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/decorate"
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/img/splash"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/video"
)

type client struct {
	hwnd uintptr

	img image.Image

	bar *title.Widget

	windows struct {
		parent  *GUI
		current *app.Window
	}

	dimensions struct {
		size image.Point

		maximized,
		fullscreened bool
	}

	menu struct {
		fullscreen *button.Widget
	}

	hover time.Time
}

func (g *GUI) client() {
	ui := g.clientUI()

	ui.windows.current.Perform(system.ActionRaise)

	defer fps.NewLoop(&fps.LoopOptions{
		Async: true,
		FPS:   60,
		Render: func(min, max, avg time.Duration) (close bool) {
			var err error

			ui.img, err = video.Capture()
			if err != nil {
				g.ToastError(err)
				g.next(is.MainMenu)
				return true
			}

			return false
		},
	}).Stop()

	var ops op.Ops

	for event := range ui.windows.current.Events() {
		switch e := event.(type) {
		case system.DestroyEvent:
			return
		case app.ViewEvent:
			ui.hwnd = e.HWND
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)

			if ui.dimensions.fullscreened {
				ui.bar.Hide = time.Since(ui.hover) > time.Second/2
			} else {
				ui.dimensions.size = e.Size
			}

			for _, e := range gtx.Events(g) {
				switch event := e.(type) {
				case key.Event:
					if event.State != key.Release {
						continue
					}

					switch event.Name {
					case key.NameF11:
						ui.menu.fullscreen.Click(ui.menu.fullscreen)
					case key.NameEscape:
						if !ui.dimensions.fullscreened {
							break
						}

						ui.menu.fullscreen.Click(ui.menu.fullscreen)
					default:
						ui.bar.Hide = !ui.bar.Hide
					}
				case pointer.Event:
					switch event.Type {
					case pointer.Move, pointer.Enter:
						if !ui.dimensions.fullscreened {
							break
						}
						ui.hover = time.Now()
						ui.bar.Hide = false
					}
				}
			}

			ui.bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
					dims := widget.Image{
						Fit:      widget.Fill,
						Src:      paint.NewImageOp(notify.Preview),
						Position: layout.Center,
					}.Layout(gtx)

					area := clip.Rect(gtx.Constraints).Push(gtx.Ops)

					pointer.InputOp{
						Tag:   g,
						Types: pointer.Enter | pointer.Move,
					}.Add(gtx.Ops)

					key.InputOp{
						Tag:  g,
						Keys: key.Set(key.NameEscape),
					}.Add(gtx.Ops)

					area.Pop()

					return dims
				})
			})

			ui.windows.current.Invalidate()

			e.Frame(gtx.Ops)
		}
	}
}

func (g *GUI) clientUI() *client {
	ui := &client{
		img: splash.Default(),
	}

	ui.bar = title.New(
		"Client",
		fonts.NewCollection(),
		func() { ui.windows.current.Perform(system.ActionMinimize) },
		func() {
			ui.bar.Hide = !ui.dimensions.fullscreened
			ui.dimensions.fullscreened = !ui.dimensions.fullscreened
			if ui.dimensions.fullscreened {
				ui.windows.current.Option(
					app.Fullscreen.Option(),
				)
			} else {
				ui.windows.current.Option(
					app.Windowed.Option(),
					app.Size(
						unit.Dp(ui.dimensions.size.X),
						unit.Dp(ui.dimensions.size.Y),
					),
				)
			}
		},
		func() { ui.windows.current.Perform(system.ActionClose) },
	)
	ui.bar.NoDrag = false

	ui.dimensions.size = image.Pt(1280, 720)

	ui.windows.parent = g
	ui.windows.current = app.NewWindow(
		app.Title("Client"),
		app.Size(unit.Dp(ui.dimensions.size.X), unit.Dp(ui.dimensions.size.Y)),
		app.MinSize(unit.Dp(ui.dimensions.size.X), unit.Dp(ui.dimensions.size.Y)),
		app.Decorated(false),
	)

	return ui
}
