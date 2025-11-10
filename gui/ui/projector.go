package ui

import (
	"fmt"
	"image"
	"time"
	"unsafe"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/ux/button"
	"github.com/pidgy/unitehud/gui/ux/decorate"
	"github.com/pidgy/unitehud/gui/ux/electron"
	"github.com/pidgy/unitehud/gui/ux/keys"
	"github.com/pidgy/unitehud/gui/ux/title"
	"github.com/pidgy/unitehud/media/img/splash"
	"github.com/pidgy/unitehud/media/video"
	"github.com/pidgy/unitehud/media/video/device"
	"github.com/pidgy/unitehud/media/video/fps"
	"github.com/pidgy/unitehud/system/wapi"
)

type projector struct {
	hwnd uintptr

	overlay   image.Image
	overlayOp paint.ImageOp

	nav struct {
		*title.Widget

		overlay     *button.Widget
		fps         *button.Widget
		alwaysOnTop *button.Widget
	}

	window *app.Window

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

	keybinds keys.Bind
	tag      any

	rect wapi.Rectangle

	imgDims layout.Dimensions

	ops op.Ops
}

func (g *GUI) projector(onclose func()) {
	if electron.Active() {
		return
	}
	defer onclose()

	toast := g.ToastSplash("UniteHUD Projector", "Loading...", splash.Projector()).wait()
	defer toast.close()

	ui := g.projectorUI()
	defer ui.nav.Remove(ui.nav.overlay)
	defer ui.nav.Remove(ui.nav.fps)
	defer ui.nav.Remove(ui.nav.alwaysOnTop)

	err := electron.Open(ui.dimensions.size)
	if err != nil {
		notify.Error("[UI] Failed to render overlay (%v)", err)
		return
	}
	defer electron.Close()

	defer fps.NewLoop(
		&fps.LoopOptions{
			Async: true,
			FPS:   2,
			On: func(min, max, avg time.Duration) (close bool) {
				if ui.hwnd == 0 {
					return
				}

				if !ui.nav.overlay.Radio {
					electron.Hide()
					return false
				}

				if _, ok := ui.nav.Dragging(); ok {
					electron.Hide()
					return false
				}

				go electron.Follow(ui.hwnd, ui.imgDims.Size, ui.dimensions.fullscreened)

				return false
			},
		}).Stop()

	toast.close()

	ui.window.Perform(system.ActionCenter)
	ui.window.Perform(system.ActionRaise)

	fpsLabel := material.Label(ui.nav.Calibri().Theme, 16, "FPS: 60")
	fpsLabel.Color = nrgba.Red.Color()
	fpsLabel.Font.Weight = font.SemiBold

	type FrameTiming struct {
		Start, End      time.Time
		FrameCount      int
		FramesPerSecond float64
	}

	timingWindow := time.Second
	timings := []FrameTiming{}
	frameTotal, frameCounter := 0, 0
	timingStart := time.Time{}

	for {
		if is.Now != is.MainMenu {
			ui.window.Perform(system.ActionClose)
		}

		switch event := ui.window.NextEvent().(type) {
		case system.DestroyEvent:
			notify.System("[UI] Closing Projector...")
			return
		case app.ViewEvent:
			if ui.hwnd == 0 {
				ui.hwnd = event.HWND
			}
		case system.FrameEvent:
			gtx := layout.NewContext(&ui.ops, event)

			if ui.dimensions.fullscreened {
				ui.nav.Hide = time.Since(ui.hover) > time.Second*2
			} else {
				ui.dimensions.size = event.Size
			}

			for _, e := range gtx.Events(g) {
				event, ok := e.(pointer.Event)
				if ok {
					switch event.Kind {
					case pointer.Release:
						if time.Since(ui.clicked) < time.Second/2 {
							ui.fullscreen()
							ui.clicked = time.Time{}
						} else {
							ui.clicked = time.Now()
						}
					}
				}
			}

			ui.nav.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
						layout.Stack{}.Layout(gtx,
							layout.Stacked(func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{
									Axis: layout.Horizontal,
								}.Layout(gtx,
									layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											img, err := video.Capture()
											if err != nil {
												g.ToastError(err)
												g.next(is.MainMenu)
											}

											ui.imgDims = widget.Image{
												Fit:      widget.Contain,
												Src:      paint.NewImageOp(img),
												Position: layout.Center,
											}.Layout(gtx)

											return ui.imgDims
										})
									}),
								)
							}),
							layout.Stacked(func(gtx layout.Context) layout.Dimensions {
								if ui.nav.fps.Radio {
									fpsLabel.Text = "Window: 0FPS\nDevice: 0FPS"
									if device.IsActive() {
										fpsLabel.Text = fmt.Sprintf("Window: %dFPS\nDevice: %.0fFPS", frameTotal, device.FPS())
									}

									return layout.Inset{
										Top:  unit.Dp(2),
										Left: unit.Dp(2),
									}.Layout(gtx, decorate.Label(&fpsLabel, fpsLabel.Text).Layout)
								}

								return layout.Dimensions{}
							}),
						)

						area := clip.Rect(gtx.Constraints).Push(gtx.Ops)
						pointer.InputOp{
							Tag:   g,
							Kinds: pointer.Enter | pointer.Move | pointer.Release,
						}.Add(gtx.Ops)
						area.Pop()

						return layout.Dimensions{Size: gtx.Constraints.Max}
					})
				},
			)

			switch ui.keybinds.Up(gtx, ui.tag) {
			case keys.Ctrl("W"):
				ui.window.Perform(system.ActionClose)
			case keys.Ctrl("F"), key.NameF11:
				ui.fullscreen()
			case key.NameEscape:
				if ui.dimensions.fullscreened {
					ui.fullscreen()
				}
			}

			p, ok := ui.nav.Dragging()
			if ok {
				ui.setWindowPos(p)
			}

			ui.window.Invalidate()

			if timingStart.IsZero() {
				timingStart = gtx.Now
			}

			if interval := gtx.Now.Sub(timingStart); interval >= timingWindow {
				timings = append(timings, FrameTiming{
					Start:           timingStart,
					End:             gtx.Now,
					FrameCount:      frameCounter,
					FramesPerSecond: float64(frameCounter) / interval.Seconds(),
				})
				frameTotal = frameCounter
				frameCounter = 0
				timingStart = gtx.Now
			}

			event.Frame(gtx.Ops)

			frameCounter++
		default:
			notify.Missed(event, "Projector")
		}
	}
}

func (ui *projector) fullscreen() {
	electron.Hide()

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
		notify.Warn("[UI] Projector <ini:failed:set> thread execution state (%v)", err)
	}
}

func (g *GUI) projectorUI() *projector {
	ui := &projector{
		keybinds: keys.New().Bind(keys.NoMod, key.NameEscape, key.NameF11).Bind(key.ModCtrl, "W"),
		tag:      new(bool),
		rect:     wapi.Rectangle{},
	}

	ui.nav.Widget = title.New(
		"UniteHUD Projector",
		func() {
			// electron.Hide()
			ui.window.Perform(system.ActionMinimize)
		},
		ui.fullscreen,
		func() { ui.window.Perform(system.ActionClose) },
	)
	ui.nav.Open()

	ui.nav.overlay = &button.Widget{
		Text:            "⛶×",
		Font:            ui.nav.NishikiTeki(),
		OnHoverHint:     func() { ui.nav.Tip("Hide UniteHUD Overlay HUD") },
		Released:        nrgba.Transparent80,
		Pressed:         nrgba.SilverPurple,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Radio:           true,

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			if this.Radio {
				this.OnHoverHint = func() { ui.nav.Tip("Show UniteHUD Overlay HUD") }
				this.Text = "⛶"
				this.Radio = false
			} else {
				this.OnHoverHint = func() { ui.nav.Tip("Hide UniteHUD Overlay HUD") }
				this.Text = "⛶×"
				this.Radio = true
			}
		},
	}
	ui.nav.Add(ui.nav.overlay)

	ui.nav.fps = &button.Widget{
		Text:            "fps",
		Font:            ui.nav.NishikiTeki(),
		OnHoverHint:     func() { ui.nav.Tip("Show FPS values on UniteHUD Overlay HUD") },
		Released:        nrgba.Transparent80,
		Pressed:         nrgba.PastelGreen,
		TextSize:        unit.Sp(12),
		TextInsetBottom: -1,

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			if this.Radio {
				this.OnHoverHint = func() { ui.nav.Tip("Show FPS values on UniteHUD Overlay HUD") }
				this.Radio = false
				this.Pressed = nrgba.PastelGreen
				this.Released = nrgba.Transparent80
			} else {
				this.OnHoverHint = func() { ui.nav.Tip("Hide FPS values on UniteHUD Overlay HUD") }
				this.Radio = true
				this.Pressed = nrgba.Transparent80
				this.Released = nrgba.PastelGreen.Alpha(80)
			}
		},
	}
	ui.nav.Add(ui.nav.fps)

	ui.nav.alwaysOnTop = &button.Widget{
		Text:            "📌",
		Font:            ui.nav.NishikiTeki(),
		OnHoverHint:     func() { ui.nav.Tip("Show UniteHUD Overlay HUD above all windows") },
		Released:        nrgba.Transparent80,
		Pressed:         nrgba.Lilac,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			if this.Radio {
				this.OnHoverHint = func() { ui.nav.Tip("Show UniteHUD Overlay HUD above all windows") }
				this.Text = "📌"
				this.Radio = false
				wapi.SetWindowNotAlwaysOnTop(ui.hwnd)
			} else {
				this.OnHoverHint = func() { ui.nav.Tip("Hide UniteHUD Overlay HUD under active windows") }
				this.Text = "📌×"
				this.Radio = true
				wapi.SetWindowAlwaysOnTop(ui.hwnd)
			}
		},
	}
	ui.nav.Add(ui.nav.alwaysOnTop)

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
		notify.Warn("[UI] <ini:failed:set> overlay position (hwnd:%d, fullscreen:%t, moving:%t)", ui.hwnd, ui.dimensions.fullscreened, ui.dimensions.moving)
		return
	}

	ui.dimensions.smoothing++
	if ui.dimensions.smoothing < 3 {
		return
	}
	ui.dimensions.smoothing = 0

	go func() {
		notify.Debug("[UI] Projector: Setting window position from drag shift=%s", shift.String())

		ui.dimensions.moving = true
		defer func() { ui.dimensions.moving = false }()

		if shift.Eq(ui.dimensions.shift) {
			return
		}
		ui.dimensions.shift = shift

		wapi.GetWindowRect.Call(ui.hwnd, uintptr(unsafe.Pointer(&ui.rect)))
		pos := image.Pt(int(ui.rect.Left), int(ui.rect.Top)).Add(shift)

		defer notify.Debug("[UI] Projector: Setting window position from drag shift=%s", shift.String())

		go wapi.SetWindowPosNoSize(ui.hwnd, pos)
	}()
}
