package ui

import (
	"fmt"
	"image"
	"time"
	"unsafe"

	"gioui.org/app"
	"gioui.org/font"
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
	"github.com/pidgy/unitehud/media/video/fps"
	"github.com/pidgy/unitehud/media/video/monitor"
	"github.com/pidgy/unitehud/system/process"
	"github.com/pidgy/unitehud/system/wapi"
)

// projector defines projector behavior and state.
type projector struct {
	hwnd uintptr

	overlay   image.Image
	overlayOp paint.ImageOp

	nav struct {
		*title.Widget

		overlay,
		stats,
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

	keybinds *keys.Bind

	rect wapi.Rect

	imgDims layout.Dimensions

	ops op.Ops
}

// projector opens the overlay projector window and runs its event loop.
func (g *GUI) projector(onclose func()) {
	if electron.Active() {
		return
	}
	defer onclose()

	toast := g.ToastSplash("UniteHUD Projector", "Loading...", splash.Projector()).wait()
	defer toast.close()

	ui := g.projectorUI()
	defer ui.nav.Remove(ui.nav.overlay)
	defer ui.nav.Remove(ui.nav.stats)
	defer ui.nav.Remove(ui.nav.alwaysOnTop)

	notify.System("[UI] Opening Projector...")

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

	statsLabel := material.Label(ui.nav.Calibri().Theme, 16, "FPS: 60")
	statsLabel.Font.Weight = font.ExtraBold
	decorate.LabelColor(&statsLabel, nrgba.White.Color())

	// FrameTiming defines FrameTiming behavior and state.
	type FrameTiming struct {
		Start, End      time.Time
		FrameCount      int
		FramesPerSecond float64
	}

	timingWindow := time.Second
	timings := []FrameTiming{}
	frameTotal, frameCounter := 0, 0
	timingStart := time.Time{}
	capResolution := "0x0"

	for {
		if !is.Currently(is.MainMenu) {
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
												is.Next(is.MainMenu)
											}

											ui.imgDims = widget.Image{
												Fit:      widget.Contain,
												Src:      paint.NewImageOp(img),
												Position: layout.Center,
											}.Layout(gtx)

											capResolution = fmt.Sprintf("%dx%d", img.Bounds().Dx(), img.Bounds().Dy())
											return ui.imgDims
										})
									}),
								)
							}),
							layout.Stacked(func(gtx layout.Context) layout.Dimensions {
								if ui.nav.stats.Radio {
									statsLabel.Text = ""
									mi, err := wapi.GetMonitorInfoFromWindow(wapi.Window(ui.hwnd))
									if err != nil {
										statsLabel.Text += fmt.Sprintf("Error: %v\n", err)
									}
									index, err := wapi.GetMonitorIndexFromMonitorInfo(mi)
									if err != nil {
										statsLabel.Text += fmt.Sprintf("Error: %v\n", err)
									}
									statsLabel.Text += fmt.Sprintf("Monitor: %s\n", monitor.NameFromIndex(index))
									statsLabel.Text += fmt.Sprintf("ㅤRefresh: %sHz\n", g.hz)
									statsLabel.Text += fmt.Sprintf("ㅤProjector: %dFPS\n", frameTotal)
									statsLabel.Text += fmt.Sprintf("ㅤCanvas: %dx%d\n", ui.imgDims.Size.X, ui.imgDims.Size.Y)

									statsLabel.Text += fmt.Sprintf("Device: %s\n", video.Name())
									statsLabel.Text += fmt.Sprintf("ㅤFPS: %.0fFPS\n", video.FPS())
									statsLabel.Text += fmt.Sprintf("ㅤResolution: %s\n", video.Resolution())
									statsLabel.Text += fmt.Sprintf("ㅤScaled: %s\n", capResolution)

									statsLabel.Text += "System\n"
									statsLabel.Text += fmt.Sprintf("ㅤ%s\n", process.Usage.CPU)
									statsLabel.Text += fmt.Sprintf("ㅤ%s\n", process.Usage.RAM)
									statsLabel.Text += fmt.Sprintf("ㅤ%s\n", process.Usage.Threads)

									return decorate.ColorBox(gtx, layout.Inset{
										Top:  unit.Dp(2),
										Left: unit.Dp(2),
									}.Layout(gtx, statsLabel.Layout).Size, nrgba.BackgroundAlt.Alpha(200))
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

			switch ui.keybinds.Event(gtx) {
			case keys.Ctrl("W"):
				ui.window.Perform(system.ActionClose)
			case keys.Ctrl("F"), keys.F11():
				ui.fullscreen()
			case keys.Escape():
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

// fullscreen toggles projector fullscreen mode and updates power settings.
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
		notify.Warn("[UI] Projector <ini:f:set> thread execution state (%v)", err)
	}
}

// projectorUI initializes the projector window state and controls.
func (g *GUI) projectorUI() *projector {
	ui := &projector{
		keybinds: keys.New().Bind(keys.NoMod, keys.Escape(), keys.F11()).Bind(keys.CtrlMod, "W"),
		rect:     wapi.Rect{},
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
		OnHoverHint:     ui.nav.Tip,
		Hint:            "Hide UniteHUD Overlay HUD",
		Released:        nrgba.Transparent80,
		Pressed:         nrgba.SilverPurple,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Radio:           true,

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			if this.Radio {
				this.Hint = "Show UniteHUD Overlay HUD"
				this.Text = "⛶"
				this.Radio = false
			} else {
				this.Hint = "Hide UniteHUD Overlay HUD"
				this.Text = "⛶×"
				this.Radio = true
			}
		},
	}
	ui.nav.Add(ui.nav.overlay)

	ui.nav.stats = &button.Widget{
		Text:            "fps",
		Font:            ui.nav.NishikiTeki(),
		OnHoverHint:     ui.nav.Tip,
		Hint:            "Show FPS values on UniteHUD Overlay HUD",
		Released:        nrgba.Transparent80,
		Pressed:         nrgba.DreamyBlue,
		TextSize:        unit.Sp(12),
		TextInsetBottom: -1,

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			if this.Radio {
				this.Hint = "Show FPS values on UniteHUD Overlay HUD"
				this.Radio = false
				this.Pressed = nrgba.DreamyBlue
				this.Released = nrgba.Transparent80
			} else {
				this.Hint = "Hide FPS values on UniteHUD Overlay HUD"
				this.Radio = true
				this.Pressed = nrgba.Transparent80
				this.Released = nrgba.DreamyBlue.Alpha(80)
			}
		},
	}
	ui.nav.Add(ui.nav.stats)

	ui.nav.alwaysOnTop = &button.Widget{
		Text:            "📌",
		Font:            ui.nav.NishikiTeki(),
		OnHoverHint:     ui.nav.Tip,
		Hint:            "Show UniteHUD Overlay HUD above all windows",
		Released:        nrgba.Transparent80,
		Pressed:         nrgba.Lilac,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			if this.Radio {
				this.Hint = "Show UniteHUD Overlay HUD above all windows"
				this.Text = "📌"
				this.Radio = false
				wapi.SetWindowNotAlwaysOnTop(ui.hwnd)
			} else {
				this.Hint = "Hide UniteHUD Overlay HUD under active windows"
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

// setWindowPos sets the related state.
func (ui *projector) setWindowPos(shift image.Point) {
	if ui.dimensions.fullscreened || ui.hwnd == 0 || ui.dimensions.moving {
		notify.Warn("[UI] <ini:f:set> overlay position (hwnd:%d, fullscreen:%t, moving:%t)", ui.hwnd, ui.dimensions.fullscreened, ui.dimensions.moving)
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
