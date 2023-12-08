package ui

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

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/nrgba"
	"github.com/pidgy/unitehud/core/server"
	"github.com/pidgy/unitehud/gui/cursor"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/visual/area"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/decorate"
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/media/audio"
	"github.com/pidgy/unitehud/media/img/splash"
	"github.com/pidgy/unitehud/media/video"
	"github.com/pidgy/unitehud/media/video/device"
	"github.com/pidgy/unitehud/media/video/monitor"
	"github.com/pidgy/unitehud/media/video/window"
)

type footer struct {
	api,
	log,
	cpu,
	ram,
	fps,
	hz material.LabelStyle
}

type projected struct {
	img         image.Image
	constraints image.Rectangle
	inset       image.Point

	cursor bool
	since  time.Time

	showCaptureAreas bool
	hideOptions      bool

	listTextSize float32

	windows struct {
		*settings
		*preview
	}

	buttons struct {
		menu struct {
			home,
			settings,
			save,
			hide,
			capture,
			preview,
			file,
			reset *button.Widget
		}
	}

	labels struct {
		audio struct {
			in,
			out material.LabelStyle
		}

		video struct {
			device,
			monitor,
			api material.LabelStyle
		}
	}

	groups struct {
		*audios
		*videos
		*areas

		ticks,
		threshold int
	}

	*footer
}

func (g *GUI) projector() {
	ui := g.projectorUI()
	ui.buttons.menu.settings.Click(ui.buttons.menu.settings)

	defer g.header.Remove(g.header.Add(ui.buttons.menu.home))
	defer g.header.Remove(g.header.Add(ui.buttons.menu.save))
	defer g.header.Remove(g.header.Add(ui.buttons.menu.settings))
	defer g.header.Remove(g.header.Add(ui.buttons.menu.preview))
	defer g.header.Remove(g.header.Add(ui.buttons.menu.hide))
	defer g.header.Remove(g.header.Add(ui.buttons.menu.capture))
	defer g.header.Remove(g.header.Add(ui.buttons.menu.file))
	defer g.header.Remove(g.header.Add(ui.buttons.menu.reset))

	g.window.Perform(system.ActionRaise)

	var lastpos image.Point

	g.header.Open()

	var ops op.Ops

	for is.Now == is.Projecting {
		if ui.groups.ticks++; ui.groups.ticks > ui.groups.threshold {
			ui.groups.videos.window.populate(false)
			ui.groups.videos.device.populate(false)
			ui.groups.videos.monitor.populate(false)

			ui.groups.ticks = 0
		}

		switch event := g.window.NextEvent().(type) {
		case system.StageEvent:
		case app.ConfigEvent:
		case system.DestroyEvent:
			ui.buttons.menu.home.Click(ui.buttons.menu.home)
			g.next(is.Closing)
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, event)
			op.InvalidateOp{At: gtx.Now}.Add(gtx.Ops)

			if !g.dimensions.size.Eq(event.Size) || !g.position().Eq(lastpos) {
				g.dimensions.size = event.Size
				lastpos = g.position()

				ui.windows.settings.resize()

				ui.windows.preview.resize()
			}

			decorate.Background(gtx)
			decorate.Label(&ui.footer.api, "API: %s", device.APIName(device.API(config.Current.Video.Capture.Device.API)))
			decorate.Label(&ui.footer.cpu, g.performance.cpu)
			decorate.Label(&ui.footer.ram, g.performance.ram)
			decorate.Label(&ui.footer.hz, "%s Hz", g.hz)
			decorate.Label(&ui.footer.fps, "%d FPS", device.FPS())

			g.header.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				if ui.hideOptions {
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
										return ui.Layout(gtx, g.dimensions.fullscreen)
									})
								}),
							)
						}),

						ui.spacer(0, 1),

						ui.foot(gtx, ui.footer),
					)
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
									return ui.Layout(gtx, g.dimensions.fullscreen)
								})
							}),
						)
					}),

					ui.spacer(0, 1),

					layout.Flexed(0.2, func(gtx layout.Context) layout.Dimensions {
						return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{
								Axis: layout.Horizontal,
							}.Layout(gtx,
								ui.spacer(2, 0),

								layout.Flexed(15, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{
										Axis: layout.Vertical,
									}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return layout.Inset{
													Top: unit.Dp(5),
												}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													return ui.labels.audio.in.Layout(gtx)
												})
											})
										}),

										ui.spacer(0, 1),

										layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
											return ui.groups.audios.in.list.Layout(gtx)
										}),
									)
								}),

								ui.spacer(2, 0),

								layout.Flexed(15, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{
										Axis: layout.Vertical,
									}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return layout.Inset{
													Top: unit.Dp(5),
												}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													return ui.labels.audio.out.Layout(gtx)
												})
											})
										}),

										ui.spacer(0, 1),

										layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
											return ui.groups.audios.out.list.Layout(gtx)
										}),
									)
								}),

								ui.spacer(2, 0),

								layout.Flexed(15, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													return ui.labels.video.device.Layout(gtx)
												})
											})
										}),

										ui.spacer(0, 1),

										layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
											return ui.groups.videos.device.list.Layout(gtx)
										}),
									)
								}),

								ui.spacer(2, 0),

								layout.Flexed(10, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													return ui.labels.video.monitor.Layout(gtx)
												})
											})
										}),

										ui.spacer(0, 1),

										layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
											return ui.groups.videos.monitor.list.Layout(gtx)
										}),
									)
								}),

								ui.spacer(2, 0),

								layout.Flexed(15, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													return ui.labels.video.api.Layout(gtx)
												})
											})
										}),

										ui.spacer(0, 1),

										layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
											return ui.groups.videos.apis.list.Layout(gtx)
										}),
									)
								}),

								ui.spacer(3, 0),
							)
						})
					}),

					ui.foot(gtx, ui.footer),

					ui.empty(2, 0),
				)
			})

			if ui.showCaptureAreas && ui.img != nil {
				for _, area := range []*area.Widget{
					ui.groups.areas.time,
					ui.groups.areas.energy,
					ui.groups.areas.score,
					ui.groups.areas.ko,
					ui.groups.areas.objective,
					ui.groups.areas.state,
				} {
					err := area.Layout(gtx, g.header.Collection, ui.constraints, ui.img, ui.inset)
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
				ui.img = splash.Default()
			case device.IsActive(), monitor.IsDisplay(), window.IsOpen():
				var err error

				ui.img, err = video.Capture()
				if err != nil {
					g.ToastError(err)
					g.next(is.MainMenu)
					return
				}

				rgba, ok := ui.img.(*image.RGBA)
				if ok && rgba == nil {
					ui.img = splash.Device()
				}
			}

			for _, e := range gtx.Events(g) {
				switch event := e.(type) {
				case key.Event:
					if event.State != key.Release {
						continue
					}

					switch event.Name {
					case key.NameEscape:
						ui.buttons.menu.home.Click(ui.buttons.menu.home)
					}
				}
			}

			area := clip.Rect(gtx.Constraints).Push(gtx.Ops)
			key.InputOp{
				Tag:  g,
				Keys: key.Set(key.NameEscape),
			}.Add(gtx.Ops)
			area.Pop()

			g.frame(gtx, event)
		default:
			notify.Missed(event, "Projector")
		}
	}

	ui.windows.preview.close()
	ui.windows.settings.close()
}

func (g *GUI) projectorUI() *projected {
	ui := &projected{
		img:   splash.Invalid(),
		since: time.Now(),

		listTextSize: float32(14),
	}

	ui.groups.areas = g.areas(g.header.Collection)
	ui.groups.audios = g.audios(ui.listTextSize)
	ui.groups.videos = g.videos(ui.listTextSize)
	ui.groups.videos.onevent = func() {
		// ...
	}
	ui.groups.threshold = 120
	ui.groups.ticks = ui.groups.threshold

	// ui.buttons.reset = &button.Widget{
	// 	Text:            "Reset",
	// 	Font:            g.header.Collection.Calibri(),
	// 	Pressed:         nrgba.Transparent80,
	// 	Released:        nrgba.DarkGray,
	// 	TextSize:        unit.Sp(ui.listTextSize),
	// 	TextInsetBottom: unit.Dp(-2),
	// 	Size:            image.Pt(115, 20),
	// 	BorderWidth:     unit.Sp(.5),
	// 	OnHoverHint:     func() { g.header.Tip("Reset configuration") },
	// 	Click: func(this *button.Widget) {
	// 		g.ToastYesNo("Reset", fmt.Sprintf("Reset %s configuration?", config.Current.Profile),
	// 			OnToastYes(func() {
	// 				defer this.Deactivate()
	// 				defer server.Clear()

	// 				ui.groups.videos.device.list.Callback(ui.groups.videos.device.list.Items[0], ui.groups.videos.device.list)

	// 				err := config.Current.Reset()
	// 				if err != nil {
	// 					notify.Error("Failed to reset %s configuration (%v)", config.Current.Profile, err)
	// 				}

	// 				config.Current.Reload()

	// 				ui.groups.areas.energy.Min, ui.groups.areas.energy.Max = config.Current.XY.Energy.Min, config.Current.XY.Energy.Max
	// 				ui.groups.areas.time.Min, ui.groups.areas.time.Max = config.Current.XY.Time.Min, config.Current.XY.Time.Max
	// 				ui.groups.areas.score.Min, ui.groups.areas.score.Max = config.Current.XY.Scores.Min, config.Current.XY.Scores.Max
	// 				ui.groups.areas.objective.Min, ui.groups.areas.objective.Max = config.Current.XY.Objectives.Min, config.Current.XY.Objectives.Max
	// 				ui.groups.areas.ko.Min, ui.groups.areas.ko.Max = config.Current.XY.KOs.Min, config.Current.XY.KOs.Max

	// 				// ui.groups.videos.window.populate(true)
	// 				ui.groups.videos.device.populate(true)
	// 				ui.groups.videos.monitor.populate(true)

	// 				g.Actions <- Refresh

	// 				g.next(is.MainMenu)

	// 				notify.Announce("Reset UniteHUD %s configuration", config.Current.Profile)
	// 			}),
	// 			OnToastNo(this.Deactivate),
	// 		)
	// 	},
	// }

	ui.buttons.menu.home = &button.Widget{
		Text:            "🏠",
		Font:            g.header.Collection.NishikiTeki(),
		Pressed:         nrgba.Discord.Alpha(100),
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Disabled:        false,
		OnHoverHint:     func() { g.header.Tip("Return to the Main Menu") },
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			config.Current.XY.Scores = ui.groups.areas.score.Rectangle()
			config.Current.XY.Time = ui.groups.areas.time.Rectangle()
			config.Current.XY.Energy = ui.groups.areas.energy.Rectangle()
			config.Current.XY.Objectives = ui.groups.areas.objective.Rectangle()
			config.Current.XY.KOs = ui.groups.areas.ko.Rectangle()

			if config.Cached().Eq(config.Current) {
				g.Actions <- Refresh
				g.next(is.MainMenu)
				return
			}

			g.ToastYesNo("Save", "Save configuration changes?",
				OnToastYes(func() {
					defer this.Deactivate()

					server.Clear()

					err := config.Current.Save()
					if err != nil {
						notify.Error("📺 Failed to save UniteHUD configuration (%v)", err)
					}

					g.Actions <- Refresh
					g.next(is.MainMenu)
				}),
				OnToastNo(func() {
					defer this.Deactivate()

					server.Clear()

					config.Current = config.Cached()

					audio.Restart()

					g.Actions <- Refresh
					g.next(is.MainMenu)
				}),
			)
		},
	}

	ui.buttons.menu.settings = &button.Widget{
		Text:            "⚙",
		TextSize:        unit.Sp(18),
		TextInsetBottom: -2,
		Font:            g.header.Collection.NishikiTeki(),
		OnHoverHint:     func() { g.header.Tip("Open advanced settings") },
		Pressed:         nrgba.Lilac,
		BorderWidth:     unit.Sp(.1),
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			if ui.windows.settings.open() {
				ui.windows.settings.close()
				return
			}

			ui.windows.settings = g.settings(func() {
				ui.windows.settings = nil

				this.Text = "⚙"
				this.OnHoverHint = func() { g.header.Tip("Open advanced settings") }
			})

			this.Text = "⚙×"
			this.OnHoverHint = func() { g.header.Tip("Close advanced settings") }
		},
	}

	ui.buttons.menu.preview = &button.Widget{
		Text:            "🗗",
		Font:            g.header.Collection.NishikiTeki(),
		TextSize:        unit.Sp(17),
		TextInsetBottom: -1,
		Pressed:         nrgba.BloodOrange,
		OnHoverHint:     func() { g.header.Tip("Preview capture areas") },
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			if ui.windows.preview.open() {
				ui.windows.preview.close()
				return
			}

			ui.windows.preview = g.preview(ui.groups.areas, func() {
				ui.windows.preview = nil

				this.Text = "🗗"
				this.OnHoverHint = func() { g.header.Tip("Preview capture areas") }
			})

			this.Text = "🗗×"
			this.OnHoverHint = func() { g.header.Tip("Close capture area preview") }
		},
	}

	ui.buttons.menu.save = &button.Widget{
		Text:            "🖫",
		Font:            g.header.Collection.NishikiTeki(),
		Pressed:         nrgba.OfficeBlue,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Disabled:        false,
		OnHoverHint:     func() { g.header.Tip("Save configuration") },
		Click: func(this *button.Widget) {
			g.ToastYesNo("Save", "Save configuration changes?",
				OnToastYes(func() {
					defer this.Deactivate()

					server.Clear()

					config.Current.XY.Scores = ui.groups.areas.score.Rectangle()
					config.Current.XY.Time = ui.groups.areas.time.Rectangle()
					config.Current.XY.Energy = ui.groups.areas.energy.Rectangle()
					config.Current.XY.Objectives = ui.groups.areas.objective.Rectangle()
					config.Current.XY.KOs = ui.groups.areas.ko.Rectangle()

					err := config.Current.Save()
					if err != nil {
						notify.Error("📺 Failed to save UniteHUD configuration (%v)", err)
					}

					notify.System("📺 Configuration saved to " + config.Current.File())
				}),
				OnToastNo(this.Deactivate),
			)
		},
	}

	ui.buttons.menu.hide = &button.Widget{
		Text:            "⇊",
		Font:            g.header.Collection.NishikiTeki(),
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Pressed:         nrgba.Gray,
		OnHoverHint:     func() { g.header.Tip("Hide sources") },
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			ui.hideOptions = !ui.hideOptions
			if ui.hideOptions {
				this.Text = "⇈"
				this.OnHoverHint = func() { g.header.Tip("Show sources") }
			} else {
				this.Text = "⇊"
				this.OnHoverHint = func() { g.header.Tip("Hide sources") }
			}
		},
	}

	ui.buttons.menu.capture = &button.Widget{
		Text:            "⛶",
		Font:            g.header.Collection.NishikiTeki(),
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Pressed:         nrgba.DarkSeafoam,
		OnHoverHint:     func() { g.header.Tip("Test capture areas") },
		Click: func(this *button.Widget) {
			defer this.Deactivate()
			ui.showCaptureAreas = !ui.showCaptureAreas
			if ui.showCaptureAreas {
				this.Text = "⛶×"
			} else {
				this.Text = "⛶"
			}
		},
	}

	ui.buttons.menu.file = &button.Widget{
		Text:            "📝",
		Font:            g.header.Collection.NishikiTeki(),
		Pressed:         nrgba.CoolBlue,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Disabled:        false,
		OnHoverHint:     func() { g.header.Tip("Open configuration file") },
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			exe := "C:\\Windows\\system32\\notepad.exe"
			err := exec.Command(exe, config.Current.File()).Run()
			if err != nil {
				notify.Error("📺 Failed to open \"%s\" (%v)", config.Current.File(), err)
				return
			}

			// Called once window is closed.
			err = config.Load(config.Current.Profile)
			if err != nil {
				notify.Error("📺 Failed to reload \"%s\" (%v)", config.Current.File(), err)
				return
			}

			ui.groups.areas.onevent()
		},
	}

	ui.buttons.menu.reset = &button.Widget{
		Text:            "💣",
		Font:            g.header.Collection.NishikiTeki(),
		Pressed:         nrgba.PaleRed,
		TextSize:        unit.Sp(17),
		TextInsetBottom: -1,
		Disabled:        false,
		OnHoverHint:     func() { g.header.Tip("Reset configuration") },
		Click: func(this *button.Widget) {
			g.ToastYesNo("Reset", fmt.Sprintf("Reset %s configuration?", config.Current.Profile),
				OnToastYes(func() {
					defer this.Deactivate()
					defer server.Clear()

					ui.groups.videos.device.list.Callback(ui.groups.videos.device.list.Items[0], ui.groups.videos.device.list)

					err := config.Current.Reset()
					if err != nil {
						notify.Error("📺 Failed to reset %s configuration (%v)", config.Current.Profile, err)
					}

					config.Current.Reload()

					audio.Restart()

					ui.groups.areas.energy.Min, ui.groups.areas.energy.Max = config.Current.XY.Energy.Min, config.Current.XY.Energy.Max
					ui.groups.areas.time.Min, ui.groups.areas.time.Max = config.Current.XY.Time.Min, config.Current.XY.Time.Max
					ui.groups.areas.score.Min, ui.groups.areas.score.Max = config.Current.XY.Scores.Min, config.Current.XY.Scores.Max
					ui.groups.areas.objective.Min, ui.groups.areas.objective.Max = config.Current.XY.Objectives.Min, config.Current.XY.Objectives.Max
					ui.groups.areas.ko.Min, ui.groups.areas.ko.Max = config.Current.XY.KOs.Min, config.Current.XY.KOs.Max

					// ui.groups.videos.window.populate(true)
					ui.groups.videos.device.populate(true)
					ui.groups.videos.monitor.populate(true)

					g.Actions <- Refresh

					g.next(is.MainMenu)

					notify.Announce("📺 Reset UniteHUD %s configuration", config.Current.Profile)
				}),
				OnToastNo(this.Deactivate),
			)
		},
	}

	ui.labels.audio.in = material.Label(g.header.Collection.Calibri().Theme, unit.Sp(12), "Audio Input")
	ui.labels.audio.in.Color = nrgba.Highlight.Color()
	ui.labels.audio.in.Font.Weight = 100

	ui.labels.audio.out = material.Label(g.header.Collection.Calibri().Theme, unit.Sp(12), "Audio Output")
	ui.labels.audio.out.Color = nrgba.Highlight.Color()
	ui.labels.audio.out.Font.Weight = 100

	ui.labels.video.device = material.Label(g.header.Collection.Calibri().Theme, unit.Sp(12), "Video Capture Device")
	ui.labels.video.device.Color = nrgba.Highlight.Color()
	ui.labels.video.device.Font.Weight = 100

	ui.labels.video.monitor = material.Label(g.header.Collection.Calibri().Theme, unit.Sp(12), "Monitor")
	ui.labels.video.monitor.Color = nrgba.Highlight.Color()
	ui.labels.video.monitor.Font.Weight = 100

	ui.labels.video.api = material.Label(g.header.Collection.Calibri().Theme, unit.Sp(12), "Video Capture API")
	ui.labels.video.api.Color = nrgba.Highlight.Color()
	ui.labels.video.api.Font.Weight = 100

	ui.footer = &footer{
		api: material.Label(g.header.Collection.Calibri().Theme, unit.Sp(12), ""),
		log: material.Label(g.header.Collection.Calibri().Theme, unit.Sp(12), ""),
		cpu: material.Label(g.header.Collection.Calibri().Theme, unit.Sp(12), ""),
		ram: material.Label(g.header.Collection.Calibri().Theme, unit.Sp(12), ""),
		fps: material.Label(g.header.Collection.Calibri().Theme, unit.Sp(12), ""),
		hz:  material.Label(g.header.Collection.Calibri().Theme, unit.Sp(12), ""),
	}

	ui.footer.api.Color = nrgba.Highlight.Color()
	ui.footer.api.Alignment = text.Start
	ui.footer.cpu.Color = nrgba.Highlight.Color()
	ui.footer.cpu.Alignment = text.Start
	ui.footer.ram.Color = nrgba.Highlight.Color()
	ui.footer.ram.Alignment = text.Start
	ui.footer.fps.Color = nrgba.Highlight.Color()
	ui.footer.fps.Alignment = text.Start
	ui.footer.hz.Color = nrgba.Highlight.Color()
	ui.footer.hz.Alignment = text.Start

	ui.groups.videos.window.populate(false)
	ui.groups.videos.device.populate(false)
	ui.groups.videos.monitor.populate(false)
	ui.groups.videos.apis.populate(false)

	return ui
}

func (p *projected) Layout(gtx layout.Context, fullscreen bool) layout.Dimensions {
	rect := clip.Rect{
		Min: gtx.Constraints.Min.Add(image.Pt(0, title.Height)),
		Max: gtx.Constraints.Max.Sub(image.Pt(0, 5)),
	}

	for _, ev := range gtx.Events(p) {
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
		Tag:   p,
		Kinds: pointer.Move | pointer.Enter | pointer.Leave,
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

func (p *projected) foot(gtx layout.Context, f *footer) layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.Y = gtx.Dp(25)

		decorate.BackgroundTitleBar(gtx, gtx.Constraints.Max)
		decorate.Border(gtx)

		layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					p.empty(2, 0),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return f.api.Layout(gtx)
					}),

					p.empty(2, 0),
				)
			})
		})

		layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				post := notify.Last()
				f.log.Text = post.String()
				f.log.Color = post.Color()
				return f.log.Layout(gtx)
			})
		})

		return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					p.empty(2, 0),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return f.cpu.Layout(gtx)
					}),

					p.empty(5, 0),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return f.ram.Layout(gtx)
					}),

					p.empty(5, 0),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return f.fps.Layout(gtx)
					}),

					p.empty(5, 0),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return f.hz.Layout(gtx)
					}),

					p.empty(2, 0),
				)
			})
		})
	})
}

func (p *projected) spacer(x, y float32) layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		if x != 0 {
			gtx.Constraints.Max.X = int(x)
		}

		if y != 0 {
			gtx.Constraints.Max.Y = int(y)
		}

		decorate.Spacer(gtx, gtx.Constraints.Max)

		return layout.Spacer{Width: unit.Dp(x), Height: unit.Dp(y)}.Layout(gtx)
	})
}
