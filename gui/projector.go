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
	constraints image.Rectangle
	inset       image.Point

	cursor bool
	since  time.Time

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
		since: time.Now(),

		listTextSize: float32(14),
	}

	settings := &settings{
		parent: g,
		closed: true,
		width:  350,
		height: 700,
	}
	defer settings.close()

	preview := &preview{
		parent: g,
		closed: true,
		width:  350,
		height: 700,
	}
	defer preview.close()

	session, err := audio.New(audio.Disabled, audio.Default)
	if err != nil {
		g.ToastErrorf(fmt.Sprintf("Failed to route audio input to output (%v)", err))
		g.next(is.MainMenu)
		return
	}
	defer session.Close()

	audios := g.audios(projected.listTextSize, session)

	areas := g.areas(g.Bar.Collection)

	defer g.Bar.Remove(g.Bar.Add(&button.Button{
		Text:            "⚙",
		TextSize:        unit.Sp(18),
		TextInsetBottom: -2,
		Font:            g.Bar.Collection.NishikiTeki(),
		OnHoverHint:     func() { g.Bar.ToolTip("Open advanced settings") },
		Pressed:         nrgba.Transparent30,
		Released:        nrgba.Slate,
		BorderWidth:     unit.Sp(.1),
		Click: func(this *button.Button) {
			defer this.Deactivate()

			if settings.close() {
				this.Text = "⚙"
				this.OnHoverHint = func() { g.Bar.ToolTip("View advanced settings") }
			} else {
				this.Text = "🔧"
				this.OnHoverHint = func() { g.Bar.ToolTip("Close advanced settings") }

				go settings.open(func() {
					defer settings.close()

					this.Text = "⚙"
					this.OnHoverHint = func() { g.Bar.ToolTip("View advanced settings") }
				})
			}
		},
	}))

	defer g.Bar.Remove(g.Bar.Add(&button.Button{
		Text:            "🖫",
		Font:            g.Bar.Collection.NishikiTeki(),
		Released:        nrgba.OfficeBlue,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Disabled:        false,
		OnHoverHint:     func() { g.Bar.ToolTip("Save configuration") },
		Click: func(this *button.Button) {
			g.ToastYesNo("Save", "Save configuration changes?",
				func() {
					defer this.Deactivate()

					server.Clear()

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
				},
			)
		},
	}))

	defer g.Bar.Remove(g.Bar.Add(&button.Button{
		Text:            "⇵",
		Font:            g.Bar.Collection.NishikiTeki(),
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Released:        nrgba.Gray,
		OnHoverHint:     func() { g.Bar.ToolTip("Hide sources") },
		Click: func(this *button.Button) {
			defer this.Deactivate()

			projected.hideOptions = !projected.hideOptions
			if projected.hideOptions {
				this.Text = "⇈"
				this.OnHoverHint = func() { g.Bar.ToolTip("Show sources") }
			} else {
				this.Text = "⇵"
				this.OnHoverHint = func() { g.Bar.ToolTip("Hide sources") }
			}
		},
	}))

	defer g.Bar.Remove(g.Bar.Add(&button.Button{
		Text:            "⛶",
		Font:            g.Bar.Collection.NishikiTeki(),
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Released:        nrgba.DarkSeafoam,
		OnHoverHint:     func() { g.Bar.ToolTip("Test capture areas") },
		Click: func(this *button.Button) {
			defer this.Deactivate()
			projected.showCaptureAreas = !projected.showCaptureAreas
			if projected.showCaptureAreas {
				this.Text = "⛞"
			} else {
				this.Text = "⛶"
			}
		},
	}))

	defer g.Bar.Remove(g.Bar.Add(&button.Button{
		Text:            "🗗",
		Font:            g.Bar.Collection.NishikiTeki(),
		TextSize:        unit.Sp(17),
		TextInsetBottom: -1,
		Released:        nrgba.BloodOrange,
		OnHoverHint:     func() { g.Bar.ToolTip("Preview capture areas") },
		Click: func(this *button.Button) {
			defer this.Deactivate()

			if preview.close() {
				this.Text = "🗗"
				this.OnHoverHint = func() { g.Bar.ToolTip("Preview capture areas") }
			} else {
				this.Text = "🗖"
				this.OnHoverHint = func() { g.Bar.ToolTip("Close capture area preview") }

				go preview.open(areas, func() {
					defer preview.close()

					this.Text = "🗗"
					this.OnHoverHint = func() { g.Bar.ToolTip("Preview capture areas") }
				})
			}
		},
	}))

	defer g.Bar.Remove(g.Bar.Add(&button.Button{
		Text:            "🗚",
		Font:            g.Bar.Collection.NishikiTeki(),
		Released:        nrgba.CoolBlue,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Disabled:        false,
		OnHoverHint:     func() { g.Bar.ToolTip("Open configuration file") },
		Click: func(this *button.Button) {
			defer this.Deactivate()

			config.Current.HUDOverlay = false

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
	}))

	cached := config.Current

	backButton := &button.Button{
		Text:            "Back",
		Font:            g.Bar.Collection.Calibri(),
		OnHoverHint:     func() { g.Bar.ToolTip("Return to main menu") },
		Pressed:         nrgba.Transparent30,
		Released:        nrgba.DarkGray,
		TextSize:        unit.Sp(projected.listTextSize),
		TextInsetBottom: unit.Dp(-2),
		Size:            image.Pt(115, 20),
		BorderWidth:     unit.Sp(.1),
		Click: func(this *button.Button) {
			defer this.Deactivate()

			config.Current.Scores = areas.score.Rectangle()
			config.Current.Time = areas.time.Rectangle()
			config.Current.Energy = areas.energy.Rectangle()
			config.Current.Objectives = areas.objective.Rectangle()
			config.Current.KOs = areas.ko.Rectangle()

			if cached.Eq(&config.Current) {
				g.Actions <- Refresh
				g.next(is.MainMenu)
				return
			}

			g.ToastYesNo(
				"Save",
				"Save configuration changes?",
				func() {
					defer this.Deactivate()

					server.Clear()

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

					server.Clear()
					video.Close()

					config.Current = cached

					g.Actions <- Refresh
					g.next(is.MainMenu)
				},
			)
		},
	}

	videos := g.videos(projected.listTextSize)
	videos.onevent = func() {
		// ...
	}

	resetButton := &button.Button{
		Text:            "Reset",
		Font:            g.Bar.Collection.Calibri(),
		OnHoverHint:     func() { g.Bar.ToolTip("Reset configuration") },
		Pressed:         nrgba.Transparent30,
		Released:        nrgba.DarkGray,
		TextSize:        unit.Sp(projected.listTextSize),
		TextInsetBottom: unit.Dp(-2),
		Size:            image.Pt(115, 20),
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
		Font:        g.Bar.Collection.Calibri(),
		OnHoverHint: func() { g.Bar.ToolTip("Close HUD overlay") },
		Pressed:     nrgba.Transparent30,
		Released:    nrgba.DarkGray,
		TextSize:    unit.Sp(projected.listTextSize),
		Size:        image.Pt(115, 20),
		BorderWidth: unit.Sp(.1),
		Click: func(this *button.Button) {
			g.ToastYesNo("Close HUD Overlay", "Close HUD overlay?", func() {
				defer this.Deactivate()

				electron.Close()

				config.Current.HUDOverlay = false

			}, this.Deactivate)
		},
	}

	openHUDButton := &button.Button{
		Text:        "Open HUD Overlay",
		Font:        g.Bar.Collection.Calibri(),
		OnHoverHint: func() { g.Bar.ToolTip("Open HUD overlay") },
		Pressed:     nrgba.Transparent30,
		Released:    nrgba.DarkGray,
		TextSize:    unit.Sp(projected.listTextSize),
		Size:        image.Pt(115, 20),
		BorderWidth: unit.Sp(.1),
		Click: func(this *button.Button) {
			g.ToastYesNo("Open HUD Overlay", "Open HUD overlay?", func() {
				defer this.Deactivate()

				electron.Close()

				config.Current.HUDOverlay = true

				err = electron.Open()
				if err != nil {
					g.ToastError(err)
					g.next(is.MainMenu)
					return
				}

			}, this.Deactivate)
		},
	}

	audioInLabel := material.Label(g.Bar.Collection.Calibri().Theme, unit.Sp(12), "Audio In (Capture)")
	audioInLabel.Color = nrgba.Highlight.Color()
	audioInLabel.Font.Weight = 100

	audioOutLabel := material.Label(g.Bar.Collection.Calibri().Theme, unit.Sp(12), "Audio Out (Playback)")
	audioOutLabel.Color = nrgba.Highlight.Color()
	audioOutLabel.Font.Weight = 100

	videoCaptureLabel := material.Label(g.Bar.Collection.Calibri().Theme, unit.Sp(12), "Video Capture Device")
	videoCaptureLabel.Color = nrgba.Highlight.Color()
	videoCaptureLabel.Font.Weight = 100

	monitorLabel := material.Label(g.Bar.Collection.Calibri().Theme, unit.Sp(12), "Monitor")
	monitorLabel.Color = nrgba.Highlight.Color()
	monitorLabel.Font.Weight = 100

	stateLabel := material.Label(g.Bar.Collection.Calibri().Theme, unit.Sp(12), "")
	stateLabel.Color = nrgba.Highlight.Color()
	stateLabel.Alignment = text.Start

	isLabel := material.Label(g.Bar.Collection.Calibri().Theme, unit.Sp(12), "")
	isLabel.Color = nrgba.Highlight.Color()
	isLabel.Alignment = text.Start

	cpuLabel := material.Label(g.Bar.Collection.Calibri().Theme, unit.Sp(12), "")
	cpuLabel.Color = nrgba.Highlight.Color()
	cpuLabel.Alignment = text.Start

	ramLabel := material.Label(g.Bar.Collection.Calibri().Theme, unit.Sp(12), "")
	ramLabel.Color = nrgba.Highlight.Color()
	ramLabel.Alignment = text.Start

	fpsLabel := material.Label(g.Bar.Collection.Calibri().Theme, unit.Sp(12), "")
	fpsLabel.Color = nrgba.Highlight.Color()
	fpsLabel.Alignment = text.Start

	tickLabel := material.Label(g.Bar.Collection.Calibri().Theme, unit.Sp(12), "")
	tickLabel.Color = nrgba.Highlight.Color()
	tickLabel.Alignment = text.Start

	populateTicksThreshold := 120
	populateTicks := populateTicksThreshold

	videos.window.populate(false)
	videos.device.populate(false)
	videos.monitor.populate(false)

	g.Perform(system.ActionRaise)

	var lastpos image.Point

	var ops op.Ops

	for g.is == is.Projecting {
		if populateTicks++; populateTicks > populateTicksThreshold {
			videos.window.populate(false)
			videos.device.populate(false)
			videos.monitor.populate(false)

			// go func() {
			//	nameq := make(chan string)

			// 	wapi.TopWindow(nameq)

			// 	t := time.NewTimer(time.Second)
			// 	select {
			// 	case <-t.C:
			// 		println("AHASDAHDSASHDASH")
			// 		g.minimized = true
			// 	case title := <-nameq:
			// 		if !t.Stop() {
			// 			<-t.C
			// 		}
			// 		g.minimized = title != g.Bar.Title
			// 		println("here", g.minimized)
			// 	}
			// }()

			populateTicks = 0
		}

		switch e := (<-g.Events()).(type) {
		case system.StageEvent:
		case app.ConfigEvent:
		case system.DestroyEvent:
			g.next(is.Closing)
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)
			if !g.size.Eq(e.Size) || !g.position().Eq(lastpos) {
				g.size = e.Size
				lastpos = g.position()

				if settings.window != nil {
					settings.resize = true
				}

				if preview.window != nil {
					preview.resize = true
				}
			}

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
									return projected.Layout(gtx, g.fullscreen)
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
										return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return audioInLabel.Layout(gtx)
											})
										})
									}),

									projected.spacer(0, 1),

									layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
										return audios.in.list.Layout(gtx, g.Bar.Collection.Calibri().Theme)
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
										return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return audioOutLabel.Layout(gtx)
											})
										})
									}),

									projected.spacer(0, 1),

									layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
										return audios.out.list.Layout(gtx, g.Bar.Collection.Calibri().Theme)
									}),
								)
							}),

							projected.spacer(2, 0),

							layout.Flexed(0.33, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return videoCaptureLabel.Layout(gtx)
											})
										})
									}),

									projected.spacer(0, 1),

									layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
										return videos.device.list.Layout(gtx, g.Bar.Collection.Calibri().Theme)
									}),
								)
							}),
							/*
								projected.spacer(2, 0),

								layout.Flexed(0.33, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(
										gtx,

										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											label := material.Label(g.Bar.Collection.Calibri().Theme, unit.Sp(12), "Window")
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
											return videos.window.list.Layout(gtx, g.Bar.Collection.Calibri().Theme)
										}),
									)
								}),
							*/
							projected.spacer(2, 0),

							layout.Flexed(0.33, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return monitorLabel.Layout(gtx)
											})
										})
									}),

									projected.spacer(0, 1),

									layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
										return videos.monitor.list.Layout(gtx, g.Bar.Collection.Calibri().Theme)
									}),
								)
							}),

							projected.spacer(2, 0),

							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								colorBox(gtx, gtx.Constraints.Max, nrgba.Background)

								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{
										Axis: layout.Vertical,
									}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Inset{Left: 5, Right: 5, Top: 2.5, Bottom: 2.5}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												// Empty button.
												return layout.Spacer{Width: unit.Dp(openHUDButton.Size.X), Height: unit.Dp(openHUDButton.Size.Y)}.Layout(gtx)
											})
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
											return layout.Inset{Left: 5, Right: 5, Top: 2.5, Bottom: 2.5}.Layout(gtx, backButton.Layout)
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
										stateLabel.Text = fmt.Sprintf("%s %s", areas.state.Text, areas.state.Subtext)
										return stateLabel.Layout(gtx)
									}),

									projected.empty(5, 0),

									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										isLabel.Text = fmt.Sprintf("HUD %s", g.is.String())
										return isLabel.Layout(gtx)
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
										cpuLabel.Text = g.cpu
										return cpuLabel.Layout(gtx)
									}),

									projected.empty(5, 0),

									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										ramLabel.Text = g.ram
										return ramLabel.Layout(gtx)
									}),

									projected.empty(5, 0),

									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										fpsLabel.Text = fmt.Sprintf("%d FPS", g.fps.frames)
										return fpsLabel.Layout(gtx)
									}),

									projected.empty(5, 0),

									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										tickLabel.Text = fmt.Sprintf("Tick %02d", g.fps.ticks)
										return tickLabel.Layout(gtx)
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
				for _, area := range []*area.Area{
					areas.time,
					areas.energy,
					areas.score,
					areas.ko,
					areas.objective,
					areas.state,
				} {
					err := area.Layout(gtx, g.Bar.Collection, projected.constraints, projected.img, projected.inset)
					if err != nil {
						g.ToastErrorf("%s %v", area.Capture.Option, err)
						area.Reset()
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
		case key.Event:
			if e.State != key.Release {
				continue
			}

			switch e.Name {
			case key.NameEscape:
				g.next(is.Closing)
			}
		default:
			notify.Debug("Event missed: %T (Loading Window)", e)
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
			//cursor.Is(pointer.CursorNone)
		}
	}

	push := rect.Push(gtx.Ops)
	pointer.InputOp{
		Tag:   p.tag,
		Types: pointer.Move | pointer.Enter | pointer.Leave,
	}.Add(gtx.Ops)
	push.Pop()

	scaleX := float32(gtx.Constraints.Max.X) / float32(p.img.Bounds().Dx())
	scaleY := float32(gtx.Constraints.Max.Y) / float32(p.img.Bounds().Dy())
	scale := (scaleX + scaleY) / 2

	dims := widget.Image{
		Fit:      widget.Contain,
		Src:      paint.NewImageOp(p.img),
		Scale:    scale,
		Position: layout.Center,
	}.Layout(gtx)

	// Set the boundaries to be the exact dimensions of the image within projector window.
	diffX := (gtx.Constraints.Max.X - dims.Size.X)
	diffY := (gtx.Constraints.Max.Y - dims.Size.Y)
	if !p.hideOptions {
		diffX /= 2
		diffY /= 2
	}

	p.constraints = image.Rectangle{
		Min: image.Pt(diffX, diffY),
		Max: image.Pt(gtx.Constraints.Max.X-diffX, gtx.Constraints.Max.Y-diffY),
	}

	p.inset = image.Pt(
		gtx.Constraints.Max.X-int(float32(p.img.Bounds().Dx())*scale),
		gtx.Constraints.Max.Y-int(float32(p.img.Bounds().Dy())*scale),
	)

	return dims
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
