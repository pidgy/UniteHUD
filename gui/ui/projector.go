package ui

import (
	"image"
	"time"
	"unsafe"

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

	"github.com/pidgy/unitehud/avi/img/splash"
	"github.com/pidgy/unitehud/avi/video"
	"github.com/pidgy/unitehud/avi/video/fps"
	"github.com/pidgy/unitehud/core/fonts"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/ux/button"
	"github.com/pidgy/unitehud/gui/ux/decorate"
	"github.com/pidgy/unitehud/gui/ux/electron"
	"github.com/pidgy/unitehud/gui/ux/title"
	"github.com/pidgy/unitehud/system/wapi"
)

type projector struct {
	hwnd uintptr

	overlay   image.Image
	overlayOp paint.ImageOp

	nav struct {
		*title.Widget

		overlay *button.Widget
	}

	window *app.Window

	visibility struct {
		seen,
		hidden bool
	}

	dimensions struct {
		size,
		shift image.Point

		maximized,
		fullscreened,
		moving bool

		smoothing int
	}

	hover,
	clicked time.Time
}

func (g *GUI) projector(onclose func()) {
	if electron.Active() {
		return
	}
	defer onclose()

	toast := g.ToastSplash("UniteHUD Projector", "Loading...", splash.Projector()).wait()
	defer toast.close(g)

	ui := g.projectorUI()

	defer ui.nav.Remove(ui.nav.Add(ui.nav.overlay))

	err := electron.Open()
	if err != nil {
		notify.Error("[UI] Failed to render overlay (%v)", err)
		return
	}
	defer electron.Close()

	defer fps.NewLoop(&fps.LoopOptions{
		Async: true,
		FPS:   1,
		On: func(min, max, avg time.Duration) (close bool) {
			if ui.hwnd != 0 {
				go electron.Follow(ui.hwnd, ui.visibility.hidden)
			}
			return
		},
	}).Stop()

	toast.close(g)

	ui.window.Perform(system.ActionCenter)
	ui.window.Perform(system.ActionRaise)

	var ops op.Ops

	for {
		if is.Now != is.MainMenu {
			ui.window.Perform(system.ActionClose)
		}

		switch event := ui.window.NextEvent().(type) {
		case system.DestroyEvent:
			notify.System("[UI] Closing Projector...")
			return
		case system.StageEvent:
			if !ui.visibility.seen {
				ui.visibility.seen = true
			} else {
				ui.visibility.hidden = !ui.visibility.hidden
			}
		case app.ViewEvent:
			ui.hwnd = event.HWND
			ui.visibility.hidden = false
		case system.FrameEvent:

			gtx := layout.NewContext(&ops, event)

			if ui.dimensions.fullscreened {
				ui.nav.Hide = time.Since(ui.hover) > time.Second*2
			} else {
				ui.dimensions.size = event.Size
			}

			for _, e := range gtx.Events(g) {
				switch event := e.(type) {
				case key.Event:
					if event.State != key.Release {
						continue
					}

					switch event.Name {
					case key.NameF11:
						ui.fullscreen()
					case key.NameEscape:
						if ui.dimensions.fullscreened {
							ui.fullscreen()
						}
					default:
						if ui.dimensions.fullscreened {
							ui.nav.Hide = false
						}
					}
				case pointer.Event:
					switch event.Kind {
					case pointer.Release:
						if time.Since(ui.clicked) < time.Second/2 {
							ui.fullscreen()
							ui.clicked = time.Time{}
						} else {
							ui.clicked = time.Now()
						}
					case pointer.Move, pointer.Enter:
						// if !ui.dimensions.fullscreened {
						// 	break
						// }
						// ui.hover = time.Now()
						// ui.bar.Hide = false
					}
				}
			}

			fit := widget.Contain

			ui.nav.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
						layout.Flex{
							Axis: layout.Horizontal,
						}.Layout(gtx, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								img, err := video.Capture()
								if err != nil {
									g.ToastError(err)
									g.next(is.MainMenu)
								}

								return widget.Image{
									Fit:      fit,
									Src:      paint.NewImageOp(img),
									Position: layout.Center,
								}.Layout(gtx)
							})
						}))

						layout.Flex{
							Axis: layout.Horizontal,
						}.Layout(
							gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									if ui.overlay == nil {
										return layout.Dimensions{Size: gtx.Constraints.Max}
									}

									return widget.Image{
										Fit:      widget.Unscaled,
										Src:      paint.NewImageOp(ui.overlay),
										Position: layout.Center,
									}.Layout(gtx)
								})
							}),
						)

						area := clip.Rect(gtx.Constraints).Push(gtx.Ops)

						pointer.InputOp{
							Tag:   g,
							Kinds: pointer.Enter | pointer.Move | pointer.Release,
						}.Add(gtx.Ops)

						key.InputOp{
							Tag:  g,
							Keys: key.Set(key.NameEscape),
						}.Add(gtx.Ops)

						area.Pop()

						return layout.Dimensions{Size: gtx.Constraints.Max}
					})
				},
			)

			event.Frame(gtx.Ops)

			ui.window.Invalidate()

			p, ok := ui.nav.Dragging()
			if ok {
				ui.setWindowPos(p)
			}
		default:
			notify.Missed(event, "Projector")
		}
	}
}

func (ui *projector) fullscreen() {
	ui.dimensions.fullscreened = !ui.dimensions.fullscreened
	ui.nav.Hide = ui.dimensions.fullscreened

	t := wapi.ThreadExecutionState(0)

	if ui.dimensions.fullscreened {
		t = wapi.ThreadExecutionStateDisplayRequired

		ui.window.Option(app.Fullscreen.Option())
	} else {
		t = wapi.ThreadExecutionStateSystemRequired

		ui.window.Perform(system.ActionUnmaximize)
		ui.window.Option(app.Windowed.Option(), app.Size(unit.Dp(ui.dimensions.size.X), unit.Dp(ui.dimensions.size.Y)))
		ui.window.Perform(system.ActionCenter)
	}

	err := wapi.SetThreadExecutionState(t, wapi.ThreadExecutionStateContinuous)
	if err != nil {
		notify.Warn("[UI] Projector failed to set thread execution state (%v)", err)
	}
}

func (g *GUI) projectorUI() *projector {
	ui := &projector{}

	ui.nav.Widget = title.New(
		"UniteHUD Projector",
		fonts.NewCollection(),
		func() { ui.window.Perform(system.ActionMinimize) },
		ui.fullscreen,
		func() { ui.window.Perform(system.ActionClose) },
	)
	ui.nav.Open()

	ui.nav.overlay = &button.Widget{
		Text:            "⛶×",
		Font:            ui.nav.Collection.NishikiTeki(),
		OnHoverHint:     func() { ui.nav.Tip("Hide HUD overlay") },
		Released:        nrgba.Transparent80,
		Pressed:         nrgba.SilverPurple,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			if this.Text == "⛶×" {
				this.OnHoverHint = func() { ui.nav.Tip("Show HUD overlay") }
				this.Text = "⛶"
				electron.Hide()
			} else {
				this.OnHoverHint = func() { ui.nav.Tip("Hide HUD overlay ") }
				this.Text = "⛶×"
				electron.Show()
			}
		},
	}

	ui.dimensions.size = image.Pt(1280, 720)

	ui.window = app.NewWindow(
		app.Title("UniteHUD Projector"),
		app.Size(unit.Dp(ui.dimensions.size.X), unit.Dp(ui.dimensions.size.Y)),
		app.MinSize(unit.Dp(ui.dimensions.size.X), unit.Dp(ui.dimensions.size.Y)),
		app.Decorated(false),
	)

	return ui
}

func (ui *projector) setWindowPos(shift image.Point) {
	if ui.dimensions.fullscreened || ui.hwnd == 0 || ui.dimensions.moving {
		notify.Warn("[UI] Failed to set overlay position")
		return
	}

	ui.dimensions.smoothing++
	if ui.dimensions.smoothing < 3 {
		return
	}
	ui.dimensions.smoothing = 0

	go func() {
		ui.dimensions.moving = true
		defer func() { ui.dimensions.moving = false }()

		if shift.Eq(ui.dimensions.shift) {
			return
		}
		ui.dimensions.shift = shift

		r := &wapi.Rect{}
		wapi.GetWindowRect.Call(ui.hwnd, uintptr(unsafe.Pointer(r)))
		pos := image.Pt(int(r.Left), int(r.Top)).Add(shift)

		wapi.SetWindowPosNoSize(ui.hwnd, pos)
	}()
}
