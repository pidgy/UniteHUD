package gui

import (
	"fmt"
	"image"
	"os/exec"
	"time"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/audio"
	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/cursor"
	"github.com/pidgy/unitehud/fonts"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/visual/area"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/nrgba"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/splash"
	"github.com/pidgy/unitehud/video"
	"github.com/pidgy/unitehud/video/device"
	"github.com/pidgy/unitehud/video/monitor"
	"github.com/pidgy/unitehud/video/window"
	"github.com/pidgy/unitehud/video/window/electron"
)

type projected struct {
	img         image.Image
	constraints layout.Constraints

	cursor bool
	since  time.Time

	theme *material.Theme

	tag *bool

	showCaptureAreas       bool
	hideOptions, hideAudio bool

	listTextSize float32
}

func (g *GUI) projector() {
	g.fps.max = 60

	projected := projected{
		img:   splash.Invalid(),
		tag:   new(bool),
		theme: g.normal,
		since: time.Now(),

		listTextSize: float32(14),
	}

	session, err := audio.New(audio.Disabled, audio.Default)
	if err != nil {
		g.ToastErrorf(fmt.Sprintf("Failed to route audio input to output (%v)", err))
		g.next(is.MainMenu)
		return
	}
	defer session.Close()

	audios := g.audios(projected.listTextSize, session)

	areas := g.areas()

	defer g.Bar.Remove(g.Bar.Custom(&button.Button{
		Text:        "⌂",
		Font:        fonts.NishikiTeki(),
		TextSize:    unit.Sp(18),
		Released:    nrgba.PurpleBlue,
		OnHoverHint: func() { g.Bar.ToolTip("Return to main menu") },
		Click: func(this *button.Button) {
			defer this.Deactivate()

			g.next(is.MainMenu)
		},
	}))

	defer g.Bar.Remove(g.Bar.Custom(&button.Button{
		Text:        "🔊",
		Font:        fonts.NishikiTeki(),
		TextSize:    unit.Sp(14),
		Released:    nrgba.DarkRed,
		OnHoverHint: func() { g.Bar.ToolTip("Hide audio sources") },
		Click: func(this *button.Button) {
			defer this.Deactivate()

			projected.hideAudio = !projected.hideAudio
			if projected.hideAudio {
				this.OnHoverHint = func() { g.Bar.ToolTip("Show audio sources") }
				this.Text = "🔈"
			} else {
				this.OnHoverHint = func() { g.Bar.ToolTip("Hide audio sources") }
				this.Text = "🔊"
			}
		},
	}))

	defer g.Bar.Remove(g.Bar.Custom(&button.Button{
		Text:        "⇵",
		Font:        fonts.NishikiTeki(),
		TextSize:    unit.Sp(14),
		Released:    nrgba.PastelBabyBlue,
		OnHoverHint: func() { g.Bar.ToolTip("Hide sources") },
		Click: func(this *button.Button) {
			defer this.Deactivate()

			projected.hideOptions = !projected.hideOptions
			if projected.hideOptions {
				this.OnHoverHint = func() { g.Bar.ToolTip("Show sources") }
			} else {
				this.OnHoverHint = func() { g.Bar.ToolTip("Hide sources") }
			}
		},
	}))

	captureButton := &button.Button{
		Text:        "⛶",
		Font:        fonts.NishikiTeki(),
		TextSize:    unit.Sp(16),
		Released:    nrgba.Orange,
		OnHoverHint: func() { g.Bar.ToolTip("Test capture areas") },
		Click: func(this *button.Button) {
			defer this.Deactivate()
			projected.showCaptureAreas = !projected.showCaptureAreas
			if projected.showCaptureAreas {
				this.Text = "⛞"
				this.TextSize = unit.Sp(14)
			} else {
				this.Text = "⛶"
				this.TextSize = unit.Sp(16)
			}
		},
	}
	defer g.Bar.Remove(g.Bar.Custom(captureButton))

	defer g.Bar.Remove(g.Bar.Custom(&button.Button{
		Text:        "🗗",
		Font:        fonts.NishikiTeki(),
		TextSize:    unit.Sp(16),
		Released:    nrgba.Seafoam,
		OnHoverHint: func() { g.Bar.ToolTip("Preview capture areas") },
		Click: func(this *button.Button) {
			defer this.Deactivate()

			g.previewCaptures(g.areas())
		},
	}))

	saveButton := &button.Button{
		Text:        "🖫",
		Pressed:     nrgba.Transparent30,
		Released:    nrgba.CoolBlue,
		Font:        fonts.NishikiTeki(),
		TextSize:    unit.Sp(14),
		Disabled:    false,
		OnHoverHint: func() { g.Bar.ToolTip("Save configuration") },
		Click: func(this *button.Button) {
			g.ToastYesNo("Save", "Save configuration changes?",
				func() {
					defer this.Deactivate()

					server.Clear()

					this.Disabled = true

					config.Current.Scores = areas.score.Rectangle()
					config.Current.Time = areas.time.Rectangle()
					config.Current.Energy = areas.energy.Rectangle()
					config.Current.Objectives = areas.objective.Rectangle()
					config.Current.KOs = areas.ko.Rectangle()

					err := config.Current.Save()
					if err != nil {
						notify.Error("Failed to save UniteHUD configuration (%v)", err)
					}

					notify.System("Configuration saved to " + config.Current.File())

					g.Actions <- Refresh

					g.next(is.MainMenu)
				},
				func() {
					defer this.Deactivate()

					g.next(is.MainMenu)
				})
		},
	}

	defer g.Bar.Remove(g.Bar.Custom(saveButton))

	backOrSaveButton := &button.Button{
		Text:            "Back",
		OnHoverHint:     func() { g.Bar.ToolTip("Return to main menu") },
		Pressed:         nrgba.Transparent30,
		Released:        nrgba.DarkGray,
		TextSize:        unit.Sp(projected.listTextSize),
		TextInsetBottom: unit.Dp(-2),
		Size:            image.Pt(100, 20),
		BorderWidth:     unit.Sp(.1),
		Click: func(this *button.Button) {
			if this.Text == "Back" {
				defer this.Deactivate()

				g.Actions <- Refresh
				g.next(is.MainMenu)

				return
			}

			g.ToastYesNo("Save", "Save configuration changes?",
				func() {
					defer this.Deactivate()

					server.Clear()

					this.Disabled = true

					config.Current.Scores = areas.score.Rectangle()
					config.Current.Time = areas.time.Rectangle()
					config.Current.Energy = areas.energy.Rectangle()
					config.Current.Objectives = areas.objective.Rectangle()
					config.Current.KOs = areas.ko.Rectangle()

					err := config.Current.Save()
					if err != nil {
						notify.Error("Failed to save UniteHUD configuration (%v)", err)
					}

					notify.System("Configuration saved to " + config.Current.File())

					g.Actions <- Refresh
					g.next(is.MainMenu)
				},
				func() {
					defer this.Deactivate()

					g.next(is.MainMenu)
				})
		},
	}

	videos := g.videos(projected.listTextSize)
	videos.onevent = func() {
		saveButton.Disabled = false
		backOrSaveButton.Text = "Save"
	}

	resetButton := &button.Button{
		Text:            "Reset",
		OnHoverHint:     func() { g.Bar.ToolTip("Reset configuration") },
		Pressed:         nrgba.Transparent30,
		Released:        nrgba.DarkGray,
		TextSize:        unit.Sp(projected.listTextSize),
		TextInsetBottom: unit.Dp(-2),
		Size:            image.Pt(100, 20),
		BorderWidth:     unit.Sp(.1),
		Click: func(this *button.Button) {
			g.ToastYesNo("Reset", fmt.Sprintf("Reset UniteHUD %s configuration?", config.Current.Profile), func() {
				defer this.Deactivate()
				defer server.Clear()

				videos.device.list.Callback(videos.device.list.Items[0], videos.device.list)

				electron.Close()

				err = config.Current.Reset()
				if err != nil {
					notify.Error("Failed to reset %s configuration (%v)", config.Current.Profile, err)
				}

				config.Current.Reload()

				areas.energy.Min, areas.energy.Max = config.Current.Energy.Min, config.Current.Energy.Max
				areas.time.Min, areas.time.Max = config.Current.Time.Min, config.Current.Time.Max
				areas.score.Min, areas.score.Max = config.Current.Scores.Min, config.Current.Scores.Max
				areas.objective.Min, areas.objective.Max = config.Current.Objectives.Min, config.Current.Objectives.Max
				areas.ko.Min, areas.ko.Max = config.Current.KOs.Min, config.Current.KOs.Max

				// videos.window.populate(true)
				videos.device.populate(true)
				videos.monitor.populate(true)

				g.Actions <- Refresh

				g.next(is.MainMenu)

				notify.Announce("Reset UniteHUD %s configuration", config.Current.Profile)
			}, this.Deactivate)
		},
	}

	closeHUDButton := &button.Button{
		Text:        "Close HUD Overlay",
		OnHoverHint: func() { g.Bar.ToolTip("Close HUD overlay") },
		Pressed:     nrgba.Transparent30,
		Released:    nrgba.DarkGray,
		TextSize:    unit.Sp(projected.listTextSize),
		Size:        image.Pt(100, 20),
		BorderWidth: unit.Sp(.1),
		Click: func(this *button.Button) {
			g.ToastYesNo("Close HUD Overlay", "Close HUD overlay?", func() {
				defer this.Deactivate()
				electron.Close()
			}, this.Deactivate)
		},
	}

	openHUDButton := &button.Button{
		Text:        "Open HUD Overlay",
		OnHoverHint: func() { g.Bar.ToolTip("Open HUD overlay") },
		Pressed:     nrgba.Transparent30,
		Released:    nrgba.DarkGray,
		TextSize:    unit.Sp(projected.listTextSize),
		Size:        image.Pt(100, 20),
		BorderWidth: unit.Sp(.1),
		Click: func(this *button.Button) {
			g.ToastYesNo("Open HUD Overlay", "Open HUD overlay?", func() {
				defer this.Deactivate()

				electron.Close()

				err = electron.Open()
				if err != nil {
					g.ToastError(err)
					g.next(is.MainMenu)
					return
				}
			}, this.Deactivate)
		},
	}

	openConfigFileButton := &button.Button{
		Text:        "Open Configuration",
		OnHoverHint: func() { g.Bar.ToolTip("Open configuration file") },
		Pressed:     nrgba.Transparent30,
		Released:    nrgba.DarkGray,
		TextSize:    unit.Sp(projected.listTextSize),
		Size:        image.Pt(100, 20),
		BorderWidth: unit.Sp(.1),
		Click: func(b *button.Button) {
			defer b.Deactivate()

			exe := "C:\\Windows\\system32\\notepad.exe"
			err := exec.Command(exe, config.Current.File()).Run()
			if err != nil {
				notify.Error("Failed to open \"%s\" (%v)", config.Current.File(), err)
				return
			}

			// Called once window is closed.
			err = config.Load(config.Current.Profile)
			if err != nil {
				notify.Error("Failed to reload \"%s\" (%v)", config.Current.File(), err)
				return
			}

			areas.onevent()
		},
	}

	g.Perform(system.ActionRaise)
	g.Perform(system.ActionCenter)

	populateTicksThreshold := 120
	populateTicks := populateTicksThreshold

	var ops op.Ops

	for g.is == is.Projecting {
		if populateTicks++; populateTicks > populateTicksThreshold {
			videos.window.populate(false)
			videos.device.populate(false)
			videos.monitor.populate(false)
			populateTicks = 0
		}

		e := <-g.Events()
		switch e := e.(type) {
		case app.ViewEvent:
			g.HWND = e.HWND
		case system.DestroyEvent:
			g.next(is.Closing)
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)

			err = session.Error()
			if err != nil && err != audio.SessionClosed {
				g.ToastError(err)
				g.next(is.MainMenu)
				return
			}

			g.Bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				if projected.hideOptions {
					return projected.Layout(gtx, true)
				}

				return layout.Flex{
					Alignment: layout.Baseline,
					Axis:      layout.Vertical,
				}.Layout(gtx,
					layout.Flexed(0.99, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{
							Axis: layout.Horizontal,
						}.Layout(
							gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return projected.Layout(gtx, g.fullscreen())
								})
							}),
						)
					}),

					projected.spacer(0, 1),

					layout.Flexed(0.2, func(gtx layout.Context) layout.Dimensions {
						colorBox(gtx, gtx.Constraints.Max, nrgba.Background)

						return layout.Flex{
							Axis: layout.Horizontal,
						}.Layout(
							gtx,

							projected.spacer(2, 0),

							layout.Flexed(0.33, func(gtx layout.Context) layout.Dimensions {
								if projected.hideAudio {
									return layout.Dimensions{}
								}

								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										label := material.Label(projected.theme, unit.Sp(12), "Audio In (Capture)")
										label.Color = nrgba.Highlight.Color()
										label.Font.Weight = 100

										return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return label.Layout(gtx)
											})
										})
									}),

									projected.spacer(0, 1),

									layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
										return audios.in.list.Layout(gtx, projected.theme)
									}),
								)
							}),

							projected.spacer(2, 0),

							layout.Flexed(0.33, func(gtx layout.Context) layout.Dimensions {
								if projected.hideAudio {
									return layout.Dimensions{}
								}

								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										label := material.Label(projected.theme, unit.Sp(12), "Audio Out (Playback)")
										label.Color = nrgba.Highlight.Color()
										label.Font.Weight = 100

										return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return label.Layout(gtx)
											})
										})
									}),

									projected.spacer(0, 1),

									layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
										return audios.out.list.Layout(gtx, projected.theme)
									}),
								)
							}),

							projected.spacer(2, 0),

							layout.Flexed(0.33, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										label := material.Label(projected.theme, unit.Sp(12), "Video Capture Device")
										label.Color = nrgba.Highlight.Color()
										label.Font.Weight = 100

										return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return label.Layout(gtx)
											})
										})
									}),

									projected.spacer(0, 1),

									layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
										return videos.device.list.Layout(gtx, projected.theme)
									}),
								)
							}),
							/*
								projected.spacer(2, 0),

								layout.Flexed(0.33, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(
										gtx,

										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											label := material.Label(projected.theme, unit.Sp(12), "Window")
											label.Color = nrgba.Highlight.Color()
											label.Font.Weight = 100

											return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													return label.Layout(gtx)
												})
											})
										}),

										projected.spacer(0, 1),

										layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
											return videos.window.list.Layout(gtx, projected.theme)
										}),
									)
								}),
							*/
							projected.spacer(2, 0),

							layout.Flexed(0.33, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										label := material.Label(projected.theme, unit.Sp(12), "Monitor")
										label.Color = nrgba.Highlight.Color()
										label.Font.Weight = 100

										return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return label.Layout(gtx)
											})
										})
									}),

									projected.spacer(0, 1),

									layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
										return videos.monitor.list.Layout(gtx, projected.theme)
									}),
								)
							}),

							projected.spacer(2, 0),

							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								colorBox(gtx, gtx.Constraints.Max, nrgba.Background)

								if projected.showCaptureAreas {
									saveButton.Disabled = false
								}

								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Inset{Left: 5, Right: 5, Top: 2.5, Bottom: 2.5}.Layout(gtx, openConfigFileButton.Layout)
										}),

										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											if electron.IsOpen() {
												return layout.Inset{Left: 5, Right: 5, Top: 2.5, Bottom: 2.5}.Layout(gtx, closeHUDButton.Layout)
											}
											return layout.Inset{Left: 5, Right: 5, Top: 2.5, Bottom: 2.5}.Layout(gtx, openHUDButton.Layout)
										}),

										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Inset{Left: 5, Right: 5, Top: 2.5, Bottom: 2.5}.Layout(gtx, resetButton.Layout)
										}),

										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Inset{Left: 5, Right: 5, Top: 2.5, Bottom: 2.5}.Layout(gtx, backOrSaveButton.Layout)
										}),
									)
								})
							}),

							projected.spacer(2, 0),
						)
					}),

					projected.spacer(0, 1),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						colorBox(gtx, gtx.Constraints.Max, nrgba.Background)

						layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									projected.empty(2, 0),

									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										state := material.Label(projected.theme, unit.Sp(12), fmt.Sprintf("%s %s", areas.state.Text, areas.state.Subtext))
										state.Color = nrgba.Highlight.Color()
										state.Alignment = text.Start
										return state.Layout(gtx)
									}),

									projected.empty(5, 0),

									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										next := material.Label(projected.theme, unit.Sp(12), fmt.Sprintf("HUD %s", g.is.String()))
										next.Color = nrgba.Highlight.Color()
										next.Alignment = text.Start
										return next.Layout(gtx)
									}),

									projected.empty(2, 0),
								)
							})
						})

						return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									projected.empty(2, 0),

									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										label := material.Label(projected.theme, unit.Sp(12), g.cpu)
										label.Color = nrgba.Highlight.Color()
										label.Alignment = text.Start

										return label.Layout(gtx)
									}),

									projected.empty(5, 0),

									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										label := material.Label(projected.theme, unit.Sp(12), g.ram)
										label.Color = nrgba.Highlight.Color()
										label.Alignment = text.Start

										return label.Layout(gtx)
									}),

									projected.empty(5, 0),

									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										label := material.Label(projected.theme, unit.Sp(12), fmt.Sprintf("%d FPS", g.fps.frames))
										label.Color = nrgba.Highlight.Color()
										label.Alignment = text.Start

										return label.Layout(gtx)
									}),

									projected.empty(5, 0),

									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										label := material.Label(projected.theme, unit.Sp(12), fmt.Sprintf("Tick %02d", g.fps.ticks))
										label.Color = nrgba.Highlight.Color()
										label.Alignment = text.Start

										return label.Layout(gtx)
									}),

									projected.empty(2, 0),
								)
							})
						})
					}),

					projected.empty(2, 0),
				)
			})

			if projected.showCaptureAreas && projected.img != nil {
				for _, area := range []*area.Area{areas.time, areas.energy, areas.score, areas.ko, areas.objective, areas.state} {
					err := area.Layout(gtx, projected.constraints, projected.img)
					if err != nil {
						g.ToastErrorf("%s: %v. Capture area has been removed.", area.Text, err)
						notify.Error("%v", err)
						captureButton.Click(captureButton)
					}
					if area.Focus {
						cursor.Is(pointer.CursorPointer)
					}
					if area.Drag {
						cursor.Is(pointer.CursorCrosshair)
					}
				}
			}

			switch {
			case window.Lost():
				fallthrough
			default:
				projected.img = splash.Default()

			case device.IsActive(), monitor.IsDisplay(), window.IsOpen():
				projected.img, err = video.Capture()
				if err != nil {
					g.ToastError(err)
					g.next(is.MainMenu)
					return
				}
			}

			g.frame(gtx, e)
		case app.ConfigEvent:
		case key.Event:
			if e.State != key.Release {
				continue
			}

			switch e.Name {
			case "F11":
				fallthrough
			case key.NameEscape:
				g.maximize()
			}
		case pointer.Event:
		case system.StageEvent:
		}
	}
}

func (p *projected) Layout(gtx layout.Context, fullscreen bool) layout.Dimensions {
	rect := clip.Rect{
		Min: gtx.Constraints.Min.Add(image.Pt(0, title.Height)),
		Max: gtx.Constraints.Max.Sub(image.Pt(0, 5)),
	}

	for _, ev := range gtx.Events(p.tag) {
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}

		p.cursor = e.Position.Round().In(image.Rectangle(rect))
		p.since = time.Now()
	}

	if fullscreen && p.cursor {
		gtx.Constraints.Min = rect.Min

		if time.Since(p.since) > time.Second {
			cursor.Is(pointer.CursorNone)
		}
	}

	push := rect.Push(gtx.Ops)
	pointer.InputOp{
		Tag:   p.tag,
		Types: pointer.Move | pointer.Enter | pointer.Leave,
	}.Add(gtx.Ops)
	push.Pop()

	scale := float32(gtx.Constraints.Max.Y) / float32(p.img.Bounds().Dy())

	p.constraints = gtx.Constraints

	return layout.Center.Layout(gtx,
		widget.Image{
			Fit:   widget.Contain,
			Src:   paint.NewImageOp(p.img),
			Scale: scale,
		}.Layout,
	)
}

func (p *projected) empty(x, y float32) layout.FlexChild {
	return layout.Rigid(layout.Spacer{Width: unit.Dp(x), Height: unit.Dp(y)}.Layout)
}

func (p *projected) spacer(x, y float32) layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		if x != 0 {
			gtx.Constraints.Max.X = int(x)
		}
		if y != 0 {
			gtx.Constraints.Max.Y = int(y)
		}
		colorBox(gtx, gtx.Constraints.Max, nrgba.White.Alpha(5))

		return layout.Spacer{Width: unit.Dp(x), Height: unit.Dp(y)}.Layout(gtx)
	})
}
