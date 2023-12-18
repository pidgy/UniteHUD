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
	"github.com/pidgy/unitehud/avi/video/wapi"
	"github.com/pidgy/unitehud/core/fonts"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/gui/cursor"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/ux/decorate"
	"github.com/pidgy/unitehud/gui/ux/electron"
	"github.com/pidgy/unitehud/gui/ux/title"
)

type overlay struct {
	hwnd uintptr

	video,
	overlay image.Image
	overlayOp paint.ImageOp

	bar *title.Widget

	windows struct {
		parent  *GUI
		current *app.Window
	}

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

func (g *GUI) overlay(onclose func()) {
	toast := g.ToastSplash("UniteHUD Overlay", "Loading...", splash.Projector()).wait()
	defer toast.close()

	ui := g.overlayUI()

	ui.windows.current.Perform(system.ActionCenter)
	ui.windows.current.Perform(system.ActionRaise)

	err := electron.OpenWindow()
	if err != nil {
		notify.Warn("Overlay: Failed to render overlay (%v)", err)
		return
	}
	defer electron.CloseWindow()

	defer fps.NewLoop(&fps.LoopOptions{
		Async: true,
		FPS:   1,
		Render: func(min, max, avg time.Duration) (close bool) {
			if ui.hwnd != 0 {
				go electron.Follow(ui.hwnd, ui.visibility.hidden)
			}
			return
		},
	}).Stop()

	toast.close()

	defer onclose()

	var ops op.Ops

	for {
		switch event := ui.windows.current.NextEvent().(type) {
		case system.DestroyEvent:
			notify.System("Overlay: Closing...")
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
				ui.bar.Hide = time.Since(ui.hover) > time.Second*2
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
							ui.bar.Hide = false
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

			ui.bar.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
						layout.Flex{
							Axis: layout.Horizontal,
						}.Layout(
							gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
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
							}),
						)

						layout.Flex{
							Axis: layout.Horizontal,
						}.Layout(
							gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									if ui.overlay != nil {
										return widget.Image{
											Fit:      widget.Unscaled,
											Src:      paint.NewImageOp(ui.overlay),
											Position: layout.Center,
										}.Layout(gtx)
									}
									return layout.Dimensions{Size: gtx.Constraints.Max}
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
			if ui.bar.Hide {
				cursor.Is(pointer.CursorNone)
			}

			ui.windows.current.Invalidate()

			event.Frame(gtx.Ops)

			p, ok := ui.bar.Dragging()
			if ok {
				ui.setWindowPos(p)
			}
		default:
			notify.Missed(event, "Overlay")
		}
	}
}

func (ui *overlay) fullscreen() {
	ui.dimensions.fullscreened = !ui.dimensions.fullscreened
	ui.bar.Hide = ui.dimensions.fullscreened

	t := wapi.ThreadExecutionState(0)

	if ui.dimensions.fullscreened {
		t = wapi.ThreadExecutionStateDisplayRequired

		ui.windows.current.Option(app.Fullscreen.Option())
	} else {
		t = wapi.ThreadExecutionStateSystemRequired

		ui.windows.current.Perform(system.ActionUnmaximize)
		ui.windows.current.Option(app.Windowed.Option(), app.Size(unit.Dp(ui.dimensions.size.X), unit.Dp(ui.dimensions.size.Y)))
		ui.windows.current.Perform(system.ActionCenter)
	}

	err := wapi.SetThreadExecutionState(t, wapi.ThreadExecutionStateContinuous)
	if err != nil {
		notify.Warn("Overlay: Failed to set thread execution state (%v)", err)
	}
}

func (g *GUI) overlayUI() *overlay {
	ui := &overlay{
		video: splash.Projector(),
	}

	ui.bar = title.New(
		"UniteHUD",
		fonts.NewCollection(),
		func() { ui.windows.current.Perform(system.ActionMinimize) },
		ui.fullscreen,
		func() { ui.windows.current.Perform(system.ActionClose) },
	)
	ui.bar.NoDrag = false

	ui.dimensions.size = image.Pt(1280, 720)

	ui.windows.parent = g
	ui.windows.current = app.NewWindow(
		app.Title("UniteHUD Overlay"),
		app.Size(unit.Dp(ui.dimensions.size.X), unit.Dp(ui.dimensions.size.Y)),
		app.MinSize(unit.Dp(ui.dimensions.size.X), unit.Dp(ui.dimensions.size.Y)),
		app.Decorated(false),
	)

	return ui
}

func (ui *overlay) setWindowPos(shift image.Point) {
	if ui.dimensions.fullscreened || ui.hwnd == 0 || ui.dimensions.moving {
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
