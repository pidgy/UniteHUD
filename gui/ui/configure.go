package ui

import (
	"fmt"
	"image"
	"os/exec"
	"time"

	"gioui.org/app"
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
	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/core/server"
	"github.com/pidgy/unitehud/gui/cursor"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/ux/area"
	"github.com/pidgy/unitehud/gui/ux/button"
	"github.com/pidgy/unitehud/gui/ux/decorate"
	"github.com/pidgy/unitehud/gui/ux/keys"
	"github.com/pidgy/unitehud/gui/ux/title"
	"github.com/pidgy/unitehud/media/audio"
	"github.com/pidgy/unitehud/media/img/splash"
	"github.com/pidgy/unitehud/media/video"
	"github.com/pidgy/unitehud/media/video/device"
	"github.com/pidgy/unitehud/media/video/monitor"
	"github.com/pidgy/unitehud/media/video/window"
	"github.com/pidgy/unitehud/system/process"
)

type configure struct {
	ops op.Ops

	img         image.Image
	constraints image.Rectangle
	inset       image.Point

	hidePreview bool

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
			home     *button.Widget
			settings *button.Widget
			save     *button.Widget
			hide     *button.Widget
			capture  *button.Widget
			preview  *button.Widget
			file     *button.Widget
			reset    *button.Widget
			lock     *button.Widget
		}
	}

	labels struct {
		audio struct {
			in  material.LabelStyle
			out material.LabelStyle
		}

		video struct {
			device  material.LabelStyle
			monitor material.LabelStyle
			window  material.LabelStyle
			api     material.LabelStyle
			codec   material.LabelStyle
		}
	}

	groups struct {
		*audios
		*videos
		*areas

		ticks, threshold int
	}

	*footer

	tag any
}

type footer struct {
	api material.LabelStyle
	log material.LabelStyle
	cpu material.LabelStyle
	ram material.LabelStyle
	fps material.LabelStyle
	hz  material.LabelStyle
}

func (g *GUI) configure() {
	ui := g.configureUI()

	defer g.nav.OnClose(ui.buttons.menu.home.Click).Then()

	defer g.nav.Remove(g.nav.Add(ui.buttons.menu.home))
	defer g.nav.Remove(g.nav.Add(ui.buttons.menu.settings))
	defer g.nav.Remove(g.nav.Add(ui.buttons.menu.save))
	defer g.nav.Remove(g.nav.Add(ui.buttons.menu.preview))
	defer g.nav.Remove(g.nav.Add(ui.buttons.menu.hide))
	defer g.nav.Remove(g.nav.Add(ui.buttons.menu.capture))
	defer g.nav.Remove(g.nav.Add(ui.buttons.menu.file))
	defer g.nav.Remove(g.nav.Add(ui.buttons.menu.reset))
	defer g.nav.Remove(g.nav.Add(ui.buttons.menu.lock))
	g.nav.Open()

	g.window.Perform(system.ActionRaise)

	lastPos := image.Pt(0, 0)

	for is.Now == is.Configuring {
		if ui.groups.ticks++; ui.groups.ticks > ui.groups.threshold {
			ui.groups.videos.populate()
			ui.groups.ticks = 0
		}

		switch event := g.window.NextEvent().(type) {
		case system.DestroyEvent:
			ui.buttons.menu.home.Click(ui.buttons.menu.home)
			g.next(is.Closing)
		case app.ViewEvent:
			g.HWND = event.HWND
		case system.FrameEvent:
			gtx := layout.NewContext(&ui.ops, event)
			op.InvalidateOp{At: gtx.Now}.Add(gtx.Ops)

			if !g.dimensions.size.Eq(event.Size) || !g.position().Eq(lastPos) {
				g.dimensions.size = event.Size
				lastPos = g.position()

				ui.windows.settings.resize()
				ui.windows.preview.resize()
			}

			decorate.Background(gtx)

			// decorate.Label(&ui.footer.api, "API: %s", device.API(config.Current.Video.Capture.Device.API).String())
			decorate.Label(&ui.footer.api, "Resolution: %dx%d", ui.img.Bounds().Dx(), ui.img.Bounds().Dy())
			decorate.Label(&ui.footer.cpu, process.Usage.CPU.String())
			decorate.Label(&ui.footer.ram, process.Usage.RAM.String())
			decorate.Label(&ui.footer.hz, "%s Hz", g.hz)
			decorate.Label(&ui.footer.fps, "%.0f FPS", device.FPS())
			decorate.LabelColor(&ui.footer.fps, nrgba.Percent(device.FPS()/float64(config.Current.Video.Capture.Device.FPS)).Color())

			g.nav.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				if keys.Esc.Up(gtx, ui.tag) != keys.None {
					ui.buttons.menu.home.Click(ui.buttons.menu.home)
				}

				if ui.hideOptions {
					return layout.Flex{
						Alignment: layout.Baseline,
						Axis:      layout.Vertical,
					}.Layout(gtx,
						layout.Flexed(0.99, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{
								Axis: layout.Horizontal,
							}.Layout(gtx, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return ui.Layout(gtx, g.dimensions.fullscreen)
								})
							}))
						}),

						ui.spacer(0, 1),

						ui.foot(ui.footer),
					)
				}

				return layout.Flex{
					Alignment: layout.Baseline,
					Axis:      layout.Vertical,
				}.Layout(gtx,
					layout.Flexed(0.99, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{
							Axis: layout.Horizontal,
						}.Layout(gtx, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return ui.Layout(gtx, g.dimensions.fullscreen)
							})
						}))
					}),

					ui.spacer(0, 1),

					layout.Flexed(0.2, func(gtx layout.Context) layout.Dimensions {
						return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{
								Axis: layout.Horizontal,
							}.Layout(gtx,
								ui.spacer(2, 0),

								layout.Flexed(12, func(gtx layout.Context) layout.Dimensions {
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

								layout.Flexed(12, func(gtx layout.Context) layout.Dimensions {
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

								// ui.spacer(2, 0),

								// layout.Flexed(15, func(gtx layout.Context) layout.Dimensions {
								// 	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								// 		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								// 			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								// 				return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								// 					return ui.labels.video.window.Layout(gtx)
								// 				})
								// 			})
								// 		}),

								// 		ui.spacer(0, 1),

								// 		layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
								// 			return ui.groups.videos.window.list.Layout(gtx)
								// 		}),
								// 	)
								// }),

								ui.spacer(2, 0),

								layout.Flexed(7.5, func(gtx layout.Context) layout.Dimensions {
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
											return ui.groups.videos.monitors.list.Layout(gtx)
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
											return ui.groups.videos.devices.list.Layout(gtx)
										}),
									)
								}),

								ui.spacer(2, 0),

								layout.Flexed(15, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													return ui.labels.video.window.Layout(gtx)
												})
											})
										}),

										ui.spacer(0, 1),

										layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
											return ui.groups.videos.windows.list.Layout(gtx)
										}),
									)
								}),

								ui.spacer(2, 0),

								layout.Flexed(7.5, func(gtx layout.Context) layout.Dimensions {
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

								ui.spacer(2, 0),

								layout.Flexed(7.5, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													return ui.labels.video.codec.Layout(gtx)
												})
											})
										}),

										ui.spacer(0, 1),

										layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
											return ui.groups.videos.codecs.list.Layout(gtx)
										}),
									)
								}),

								ui.spacer(3, 0),
							)
						})
					}),

					ui.foot(ui.footer),

					ui.empty(2, 0),
				)
			})

			if ui.showCaptureAreas && ui.img != nil {
				for _, area := range []*area.Widget{
					ui.groups.areas.state,
					ui.groups.areas.pressButtonToScore,
					ui.groups.areas.time,
					ui.groups.areas.energy,
					ui.groups.areas.objective,
					ui.groups.areas.score,
					// ui.groups.areas.ko,
				} {
					err := area.Layout(gtx, g.nav.Collection, ui.constraints, ui.img, ui.inset)
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
			case ui.hidePreview:
				ui.img = splash.DeviceClickable()
			case device.IsActive(), monitor.IsDisplay(), window.IsOpen():
				img, err := video.Capture()
				if err != nil {
					g.ToastErrorf("<ini:f:capture> video (%v)", err)

					if monitor.IsDisplay() {
						defer g.next(is.MainMenu)
					}

					if window.IsOpen() {
						config.Current.Video.Capture.Window.Lost = config.Current.Video.Capture.Window.Name
					}
					config.Current.Video.Capture.Window.Name = config.MainDisplay

					video.Close()

					break
				}

				ui.img = img

				rgba, ok := ui.img.(*image.RGBA)
				if !ok || rgba == nil {
					ui.img = splash.Invalid()
				}
			case window.Lost():
				config.Current.Video.Capture.Window.Lost = ""
				ui.groups.videos.populate()
				fallthrough
			default:
				ui.img = splash.Default()
			}

			// for _, event := range gtx.Events(g) {
			// 	switch e := event.(type) {
			// 	case key.Event:
			// 		switch e.Name {
			// 		case key.NameEscape:
			// 			ui.buttons.menu.home.Click(ui.buttons.menu.home)
			// 		}
			// 	}
			// }

			// area := clip.Rect(gtx.Constraints).Push(gtx.Ops)
			// key.InputOp{
			// 	Tag:  g,
			// 	Keys: key.Set(key.NameEscape),
			// }.Add(gtx.Ops)
			// area.Pop()

			g.frame(gtx, event)
		default:
			notify.Missed(event, "Configure")
		}
	}

	ui.windows.preview.close()
	ui.windows.settings.close()
}

func (g *GUI) configureUI() *configure {
	ui := &configure{
		img:   splash.Invalid(),
		since: time.Now(),

		hidePreview: true,

		listTextSize: float32(12),

		tag: new(bool),
	}

	ui.groups.areas = g.areas(g.nav.Collection)
	ui.groups.audios = g.audios(ui.listTextSize)
	ui.groups.videos = g.videos(ui.listTextSize)
	ui.groups.videos.onevent = func(b bool) {
		ui.hidePreview = b
	}
	ui.groups.threshold = 120
	ui.groups.ticks = ui.groups.threshold

	ui.buttons.menu.home = &button.Widget{
		Text:            "🏠",
		Font:            g.nav.NishikiTeki(),
		Pressed:         nrgba.Discord.Alpha(100),
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Disabled:        false,
		OnHoverHint:     func() { g.nav.Tip("Return to the Main Menu") },
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			config.Current.XY.Scores = ui.groups.areas.score.Rectangle()
			config.Current.XY.Time = ui.groups.areas.time.Rectangle()
			config.Current.XY.Energy = ui.groups.areas.energy.Rectangle()
			config.Current.XY.Objectives = ui.groups.areas.objective.Rectangle()
			// config.Current.XY.KOs = ui.groups.areas.ko.Rectangle()

			if config.Cached().Eq(config.Current) {
				g.next(is.MainMenu)
				return
			}

			g.ToastYesNo("Save", "Save configuration changes?",
				toastOnYes(func() {
					defer this.Deactivate()

					server.Clear()

					err := config.Current.Save()
					if err != nil {
						notify.Warn("[UI] <ini:f:save> UniteHUD configuration (%v)", err)
					}

					g.next(is.MainMenu)
				}),
				toastOnNo(func() {
					defer this.Deactivate()

					server.Clear()

					config.Current = config.Cached()

					audio.Restart()

					err := device.Restart()
					if err != nil {
						g.ToastError(err)
					}

					g.next(is.MainMenu)
				}),
			)
		},
	}

	ui.buttons.menu.settings = &button.Widget{
		Text:            "⚙",
		TextSize:        unit.Sp(18),
		TextInsetBottom: -2,
		Font:            g.nav.NishikiTeki(),
		OnHoverHint:     func() { g.nav.Tip("Open advanced settings") },
		Pressed:         nrgba.Lilac,
		BorderWidth:     unit.Sp(.1),
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			if ui.windows.settings.isOpen() {
				ui.windows.settings.close()
				return
			}

			ui.windows.settings = g.settings(func() {
				ui.windows.settings = nil

				this.Text = "⚙"
				this.OnHoverHint = func() { g.nav.Tip("Open advanced settings") }
			})

			this.Text = "⚙×"
			this.OnHoverHint = func() { g.nav.Tip("Close advanced settings") }
		},
	}

	ui.buttons.menu.preview = &button.Widget{
		Text:            "🗗",
		Font:            g.nav.NishikiTeki(),
		TextSize:        unit.Sp(17),
		TextInsetBottom: -1,
		Pressed:         nrgba.BloodOrange,
		OnHoverHint:     func() { g.nav.Tip("Preview capture areas") },
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			if ui.windows.preview.open() {
				ui.windows.preview.close()
				return
			}

			ui.hidePreview = false

			ui.windows.preview = g.preview(ui.groups.areas, func() {
				ui.windows.preview = nil

				this.Text = "🗗"
				this.OnHoverHint = func() { g.nav.Tip("Preview capture areas") }
			})

			this.Text = "🗗×"
			this.OnHoverHint = func() { g.nav.Tip("Close capture area preview") }
		},
	}

	ui.buttons.menu.save = &button.Widget{
		Text:            "🖫",
		Font:            g.nav.NishikiTeki(),
		Pressed:         nrgba.OfficeBlue,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Disabled:        false,
		OnHoverHint:     func() { g.nav.Tip("Save configuration") },
		Click: func(this *button.Widget) {
			g.ToastYesNo("Save", "Save configuration changes?",
				toastOnYes(func() {
					defer this.Deactivate()

					server.Clear()

					config.Current.XY.Scores = ui.groups.areas.score.Rectangle()
					config.Current.XY.Time = ui.groups.areas.time.Rectangle()
					config.Current.XY.Energy = ui.groups.areas.energy.Rectangle()
					config.Current.XY.Objectives = ui.groups.areas.objective.Rectangle()
					// config.Current.XY.KOs = ui.groups.areas.ko.Rectangle()

					err := config.Current.Save()
					if err != nil {
						notify.Error("[UI] <ini:f:save> UniteHUD configuration (%v)", err)
						return
					}

					notify.System("[UI] Configuration saved to " + config.Current.File())
				}),
				toastOnNo(this.Deactivate),
			)
		},
	}

	ui.buttons.menu.hide = &button.Widget{
		Text:            "⇊",
		Font:            g.nav.NishikiTeki(),
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Pressed:         nrgba.Gray,
		OnHoverHint:     func() { g.nav.Tip("Hide sources") },
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			ui.hideOptions = !ui.hideOptions
			if ui.hideOptions {
				this.Text = "⇈"
				this.OnHoverHint = func() { g.nav.Tip("Show sources") }
			} else {
				this.Text = "⇊"
				this.OnHoverHint = func() { g.nav.Tip("Hide sources") }
			}
		},
	}

	ui.buttons.menu.capture = &button.Widget{
		Text:            "⛶",
		Font:            g.nav.NishikiTeki(),
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Pressed:         nrgba.DarkSeafoam,
		OnHoverHint:     func() { g.nav.Tip("Test capture areas") },
		Click: func(this *button.Widget) {
			defer this.Deactivate()
			ui.showCaptureAreas = !ui.showCaptureAreas
			if ui.showCaptureAreas {
				this.Text = "⛶×"
				ui.hidePreview = false
			} else {
				this.Text = "⛶"
			}
		},
	}

	ui.buttons.menu.file = &button.Widget{
		Text:            "📝",
		Font:            g.nav.NishikiTeki(),
		Pressed:         nrgba.CoolBlue,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Disabled:        false,
		OnHoverHint:     func() { g.nav.Tip("Open configuration file") },
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			err := exec.Command("C:\\Windows\\system32\\notepad.exe", config.Current.File()).Run()
			if err != nil {
				notify.Error("[UI] <ini:f:open> \"%s\" (%v)", config.Current.File(), err)
				return
			}

			// Called once window is closed.
			err = config.Open()
			if err != nil {
				notify.Error("[UI] Failed to reload \"%s\" (%v)", config.Current.File(), err)
				return
			}

			err = config.Current.Save()
			if err != nil {
				notify.Error("[UI] <ini:f:save> \"%s\" (%v)", config.Current.File(), err)
				return
			}

			ui.groups.areas = g.areas(g.nav.Collection)

			if device.IsActive() {
				err := device.Restart()
				if err != nil {
					g.ToastError(err)
				}
			}
		},
	}

	ui.buttons.menu.reset = &button.Widget{
		Text:            "💣",
		Font:            g.nav.NishikiTeki(),
		Pressed:         nrgba.PaleRed,
		TextSize:        unit.Sp(17),
		TextInsetBottom: -1,
		Disabled:        false,
		OnHoverHint:     func() { g.nav.Tip("Reset configuration") },
		Click: func(this *button.Widget) {
			g.ToastYesNo("Reset", fmt.Sprintf("Reset %s configuration?", config.Current.Gaming.Device),
				toastOnYes(func() {
					defer this.Deactivate()
					defer server.Clear()

					ui.groups.videos.devices.list.Callback(ui.groups.videos.devices.list.Items[0], ui.groups.videos.devices.list)

					err := config.Current.Reset()
					if err != nil {
						notify.Warn("[UI] Failed to reset %s configuration (%v)", config.Current.Gaming.Device, err)
					}

					config.Current.Reload()

					audio.Restart()

					ui.groups.areas.energy.Min, ui.groups.areas.energy.Max = config.Current.XY.Energy.Min, config.Current.XY.Energy.Max
					ui.groups.areas.time.Min, ui.groups.areas.time.Max = config.Current.XY.Time.Min, config.Current.XY.Time.Max
					ui.groups.areas.score.Min, ui.groups.areas.score.Max = config.Current.XY.Scores.Min, config.Current.XY.Scores.Max
					ui.groups.areas.objective.Min, ui.groups.areas.objective.Max = config.Current.XY.Objectives.Min, config.Current.XY.Objectives.Max
					// ui.groups.areas.ko.Min, ui.groups.areas.ko.Max = config.Current.XY.KOs.Min, config.Current.XY.KOs.Max
					// ui.groups.videos.window.populate()
					ui.groups.videos.populate()

					g.next(is.MainMenu)

					notify.Announce("[UI] Reset UniteHUD %s configuration", config.Current.Gaming.Device)
				}),
				toastOnNo(this.Deactivate),
			)
		},
	}

	ui.buttons.menu.lock = &button.Widget{
		Text:            "🔓",
		Font:            g.nav.NishikiTeki(),
		Pressed:         nrgba.PastelYellow,
		TextSize:        unit.Sp(17),
		TextInsetBottom: -1,
		Disabled:        false,
		OnHoverHint:     func() { g.nav.Tip("Lock capture areas to prevent mouse dragging") },
		Click: func(this *button.Widget) {
			ui.groups.areas.time.Draggable = !ui.groups.areas.time.Draggable
			ui.groups.areas.energy.Draggable = !ui.groups.areas.energy.Draggable
			ui.groups.areas.objective.Draggable = !ui.groups.areas.objective.Draggable
			ui.groups.areas.pressButtonToScore.Draggable = !ui.groups.areas.pressButtonToScore.Draggable
			ui.groups.areas.score.Draggable = !ui.groups.areas.score.Draggable
			ui.groups.areas.state.Draggable = !ui.groups.areas.state.Draggable

			if ui.groups.areas.time.Draggable {
				this.Text = "🔓"
			} else {
				this.Text = "🔒"
			}
		},
	}

	ui.labels.audio.in = material.Label(g.nav.Calibri().Theme, unit.Sp(12), "Audio Input")
	ui.labels.audio.in.Color = nrgba.Highlight.Color()
	ui.labels.audio.in.Font.Weight = 100

	ui.labels.audio.out = material.Label(g.nav.Calibri().Theme, unit.Sp(12), "Audio Output")
	ui.labels.audio.out.Color = nrgba.Highlight.Color()
	ui.labels.audio.out.Font.Weight = 100

	ui.labels.video.device = material.Label(g.nav.Calibri().Theme, unit.Sp(12), "Video Capture Device")
	ui.labels.video.device.Color = nrgba.Highlight.Color()
	ui.labels.video.device.Font.Weight = 100

	ui.labels.video.monitor = material.Label(g.nav.Calibri().Theme, unit.Sp(12), "Monitor")
	ui.labels.video.monitor.Color = nrgba.Highlight.Color()
	ui.labels.video.monitor.Font.Weight = 100

	ui.labels.video.window = material.Label(g.nav.Calibri().Theme, unit.Sp(12), "Window")
	ui.labels.video.window.Color = nrgba.Highlight.Color()
	ui.labels.video.window.Font.Weight = 100

	ui.labels.video.api = material.Label(g.nav.Calibri().Theme, unit.Sp(12), "API")
	ui.labels.video.api.Color = nrgba.Highlight.Color()
	ui.labels.video.api.Font.Weight = 100

	ui.labels.video.codec = material.Label(g.nav.Calibri().Theme, unit.Sp(12), "Codec")
	ui.labels.video.codec.Color = nrgba.Highlight.Color()
	ui.labels.video.codec.Font.Weight = 100

	ui.footer = &footer{
		api: material.Label(g.nav.Calibri().Theme, unit.Sp(12), ""),
		log: material.Label(g.nav.Calibri().Theme, unit.Sp(12), ""),
		cpu: material.Label(g.nav.Calibri().Theme, unit.Sp(12), ""),
		ram: material.Label(g.nav.Calibri().Theme, unit.Sp(12), ""),
		fps: material.Label(g.nav.Calibri().Theme, unit.Sp(12), ""),
		hz:  material.Label(g.nav.Calibri().Theme, unit.Sp(12), ""),
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

	ui.groups.videos.populate()

	ui.buttons.menu.settings.Click(ui.buttons.menu.settings)

	return ui
}

func (ui *configure) Layout(gtx layout.Context, fullscreen bool) layout.Dimensions {
	rect := clip.Rect{
		Min: gtx.Constraints.Min.Add(image.Pt(0, title.Height)),
		Max: gtx.Constraints.Max.Sub(image.Pt(0, 5)),
	}

	for _, ev := range gtx.Events(ui) {
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}

		if e.Kind == pointer.Release {
			ui.hidePreview = !ui.hidePreview
		}

		ui.cursor = e.Position.Round().In(image.Rectangle(rect))
		ui.since = time.Now()
	}

	if fullscreen && ui.cursor {
		gtx.Constraints.Min = rect.Min

		if time.Since(ui.since) > time.Second {
			//cursor.Is(pointer.CursorNone)
		}
	}

	push := rect.Push(gtx.Ops)
	pointer.InputOp{
		Tag:   ui,
		Kinds: pointer.Move | pointer.Enter | pointer.Leave | pointer.Release,
	}.Add(gtx.Ops)
	push.Pop()

	scaleX := float32(gtx.Constraints.Max.X) / float32(ui.img.Bounds().Dx())
	scaleY := float32(gtx.Constraints.Max.Y) / float32(ui.img.Bounds().Dy())
	scale := (scaleX + scaleY) / 2

	dims := widget.Image{
		Fit:      widget.Contain,
		Src:      paint.NewImageOp(ui.img),
		Scale:    scale,
		Position: layout.Center,
	}.Layout(gtx)

	// Set the boundaries to be the exact dimensions of the image within projector window.
	diffX := (gtx.Constraints.Max.X - dims.Size.X)
	diffY := (gtx.Constraints.Max.Y - dims.Size.Y)
	if !ui.hideOptions {
		diffX /= 2
		diffY /= 2
	}

	ui.constraints = image.Rectangle{
		Min: image.Pt(diffX, diffY),
		Max: image.Pt(gtx.Constraints.Max.X-diffX, gtx.Constraints.Max.Y-diffY),
	}

	ui.inset = image.Pt(
		gtx.Constraints.Max.X-int(float32(ui.img.Bounds().Dx())*scale),
		gtx.Constraints.Max.Y-int(float32(ui.img.Bounds().Dy())*scale),
	)

	return dims
}

func (ui *configure) empty(x, y float32) layout.FlexChild {
	return layout.Rigid(layout.Spacer{Width: unit.Dp(x), Height: unit.Dp(y)}.Layout)
}

func (ui *configure) foot(f *footer) layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.Y = gtx.Dp(25)

		decorate.BackgroundTitleBar(gtx, gtx.Constraints.Max)
		decorate.Border(gtx)

		layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					ui.empty(2, 0),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return f.api.Layout(gtx)
					}),

					ui.empty(2, 0),
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
					ui.empty(2, 0),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return f.cpu.Layout(gtx)
					}),

					ui.empty(5, 0),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return f.ram.Layout(gtx)
					}),

					ui.empty(5, 0),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return f.fps.Layout(gtx)
					}),

					ui.empty(5, 0),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return f.hz.Layout(gtx)
					}),

					ui.empty(2, 0),
				)
			})
		})
	})
}

func (ui *configure) spacer(x, y float32) layout.FlexChild {
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
