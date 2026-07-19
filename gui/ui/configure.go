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
	"github.com/pidgy/unitehud/system/lang"
	"github.com/pidgy/unitehud/system/process"
	"github.com/pidgy/unitehud/system/save"
	"github.com/pidgy/unitehud/system/win32"
)

// configure holds UI state for the configuration screen and its sub-windows.
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

	navButtons struct {
		menu struct {
			home,
			settings,
			save,
			hide,
			capture,
			preview,
			file,
			reset,
			lock,
			screenshot *button.Widget
		}
	}

	labels struct {
		audio struct {
			in  material.LabelStyle
			out material.LabelStyle
		}

		video struct {
			device     material.LabelStyle
			monitor    material.LabelStyle
			window     material.LabelStyle
			api        material.LabelStyle
			codec      material.LabelStyle
			monCapture material.LabelStyle
			winCapture material.LabelStyle
		}
	}

	groups struct {
		*audios
		*videos
		*areas

		ticks, threshold int
	}

	*footer

	keybinds *keys.Bind
}

// footer contains label styles for the bottom status bar.
type footer struct {
	api material.LabelStyle
	log material.LabelStyle
	cpu material.LabelStyle
	ram material.LabelStyle
	fps material.LabelStyle
	hz  material.LabelStyle
}

// configure runs the configuration window event loop and renders its layout.
func (g *GUI) configure() {
	ui := g.configureUI()

	// Uncomment next line to make configuration menu close return to main.
	defer g.nav.OnClose(func(w *button.Widget) {
		ui.navButtons.menu.home.Click(ui.navButtons.menu.home)
		is.Next(is.Closing)
	}).Then()

	defer g.nav.Remove(g.nav.Add(ui.navButtons.menu.home))
	defer g.nav.Remove(g.nav.Add(ui.navButtons.menu.settings))
	defer g.nav.Remove(g.nav.Add(ui.navButtons.menu.save))
	defer g.nav.Remove(g.nav.Add(ui.navButtons.menu.preview))
	defer g.nav.Remove(g.nav.Add(ui.navButtons.menu.hide))
	defer g.nav.Remove(g.nav.Add(ui.navButtons.menu.capture))
	defer g.nav.Remove(g.nav.Add(ui.navButtons.menu.file))
	defer g.nav.Remove(g.nav.Add(ui.navButtons.menu.reset))
	defer g.nav.Remove(g.nav.Add(ui.navButtons.menu.lock))
	defer g.nav.Remove(g.nav.Add(ui.navButtons.menu.screenshot))
	g.nav.Open()

	// g.window.Perform(system.ActionRaise)

	lastPos := g.position() // image.Pt(0, 0)

	for is.Currently(is.Configuring) {
		if ui.groups.ticks++; ui.groups.ticks > ui.groups.threshold {
			ui.groups.videos.populate()
			ui.groups.ticks = 0
		}

		switch event := g.window.NextEvent().(type) {
		case system.StageEvent:
		case system.DestroyEvent:
			ui.navButtons.menu.home.Click(ui.navButtons.menu.home)
			is.Next(is.Closing)
		case app.ViewEvent:
			g.HWND = event.HWND
		case system.FrameEvent:
			gtx := layout.NewContext(&ui.ops, event)
			// op.InvalidateOp{At: gtx.Now}.Add(gtx.Ops)
			g.dimensions.size = event.Size

			switch ui.keybinds.Event(gtx) {
			case keys.Escape():
				ui.navButtons.menu.home.Click(ui.navButtons.menu.home)
			case keys.Ctrl("S"):
				ui.navButtons.menu.save.Click(ui.navButtons.menu.save)
			case keys.Ctrl("Z"):
				ui.navButtons.menu.reset.Click(ui.navButtons.menu.reset)
			case keys.Ctrl("F"):
				g.resize()
			}

			if !g.dimensions.size.Eq(event.Size) || !g.position().Eq(lastPos) {
				g.dimensions.size = event.Size
				lastPos = g.position()

				ui.windows.settings.resize()
				ui.windows.preview.resize()
			}

			decorate.Background(gtx)

			// decorate.Label(&ui.footer.api, "API: %s", device.API(config.Current.Video.Capture.Device.API).String())
			bounds := ui.img.Bounds().Size()
			bounds2 := video.Resolution()
			apiLabel := fmt.Sprintf("Resolution: %dx%d", bounds.X, bounds.Y)
			if !bounds.Eq(bounds2) {
				apiLabel = fmt.Sprintf("Resolution: %dx%d (Scaled: %dx%d)", bounds.X, bounds.Y, bounds2.X, bounds2.Y)
			}
			decorate.Label(&ui.footer.api, "%s", apiLabel)
			decorate.Label(&ui.footer.cpu, "%s", process.Usage.CPU)
			decorate.Label(&ui.footer.ram, "%s", process.Usage.RAM)
			decorate.Label(&ui.footer.hz, "%s Hz", g.hz)
			decorate.Label(&ui.footer.fps, "%.0f FPS", device.FPS())
			decorate.LabelColor(&ui.footer.fps, nrgba.Percent(device.FPS()/float64(config.Current.Video.Capture.Device.FPS)).Color())

			g.nav.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
													if video.Is(video.Device) {
														return ui.labels.video.api.Layout(gtx)
													}
													return ui.labels.video.monCapture.Layout(gtx)
												})
											})
										}),

										ui.spacer(0, 1),

										layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
											if video.Is(video.Device) {
												return ui.groups.videos.apis.list.Layout(gtx)
											}
											return ui.groups.videos.monMethods.list.Layout(gtx)
										}),
									)
								}),

								ui.spacer(2, 0),

								layout.Flexed(7.5, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													if video.Is(video.Device) {
														return ui.labels.video.codec.Layout(gtx)
													}
													return ui.labels.video.winCapture.Layout(gtx)
												})
											})
										}),

										ui.spacer(0, 1),

										layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
											if video.Is(video.Device) {
												return ui.groups.videos.codecs.list.Layout(gtx)
											}
											return ui.groups.videos.winMethods.list.Layout(gtx)

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
			case video.Current() != video.Unknown:
				select {
				case <-ticker.C:
					img, err := video.Capture()
					if err != nil {
						g.ToastErrorf("<ini:f:capture_video> (%v)", err)
						defer video.Close()

						// No video to default to, let's bail.
						if monitor.IsActive() {
							is.Next(is.MainMenu)
						}

						if !window.IsActive() {
							config.Current.SetDefaultMonitorCapture()
						}

						break
					}

					ui.img = img

					rgba, ok := ui.img.(*image.RGBA)
					if !ok || rgba == nil {
						ui.img = splash.Invalid()
					}
				default:
				}
			default:
				ui.img = splash.Default()
			}

			g.frame(gtx, event)
		default:
			notify.Missed(event, "Configure")
		}
	}

	ui.windows.preview.close()
	ui.windows.settings.close()
	ui = nil
}

var ticker = time.NewTicker(66 * time.Millisecond) // ~15.15 FPS

// configureUI initializes the configuration UI model and controls.
func (g *GUI) configureUI() *configure {
	ui := &configure{
		img:   splash.Invalid(),
		since: time.Now(),

		hidePreview: true,

		listTextSize: float32(12),

		keybinds: keys.New().
			Bind(keys.NoMod, keys.Escape()).
			Bind(keys.CtrlMod, "S", "Z", "F"),
	}

	ui.groups.areas = g.areas(g.nav.Collection)
	ui.groups.audios = g.audios(ui.listTextSize)
	ui.groups.videos = g.videos(ui.listTextSize)
	ui.groups.videos.onevent = func(b bool) {
		ui.hidePreview = b
	}
	ui.groups.threshold = 120
	ui.groups.ticks = ui.groups.threshold

	ui.navButtons.menu.home = &button.Widget{
		Text:            "🏠",
		Font:            g.nav.NishikiTeki(),
		Pressed:         nrgba.Discord.Alpha(100),
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Disabled:        false,
		Hint:            "Return to the Main Menu (Esc)",
		OnHoverHint:     g.nav.Tip,
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			config.Current.XY.Scores = ui.groups.areas.score.Rectangle()
			config.Current.XY.Time = ui.groups.areas.time.Rectangle()
			config.Current.XY.Energy = ui.groups.areas.energy.Rectangle()
			config.Current.XY.Objectives = ui.groups.areas.objective.Rectangle()
			// config.Current.XY.KOs = ui.groups.areas.ko.Rectangle()

			if config.Cached().Eq(config.Current) {
				is.Next(is.MainMenu)
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

					is.Next(is.MainMenu)
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

					is.Next(is.MainMenu)
				}),
			)
		},
	}

	ui.navButtons.menu.settings = &button.Widget{
		Text:            "⚙",
		TextSize:        unit.Sp(18),
		TextInsetBottom: -2,
		Font:            g.nav.NishikiTeki(),
		Hint:            "Open advanced settings",
		OnHoverHint:     g.nav.Tip,
		Pressed:         nrgba.Lilac,
		BorderWidth:     unit.Sp(.1),
		Radio:           false,
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			if ui.windows.settings.isOpen() {
				ui.windows.settings.close()
				this.Radio = false
				return
			}
			this.Radio = true

			onclose := func() {
				ui.windows.settings = nil

				this.Text = "⚙"
				this.Hint = "Open advanced settings"
			}
			ui.windows.settings = g.settings(onclose)

			this.Text = "⚙×"
			this.Hint = "Close advanced settings"
		},
	}

	ui.navButtons.menu.preview = &button.Widget{
		Text:            "🗗",
		Font:            g.nav.NishikiTeki(),
		TextSize:        unit.Sp(17),
		TextInsetBottom: -1,
		Pressed:         nrgba.BloodOrange,
		Hint:            "Preview capture areas",
		OnHoverHint:     g.nav.Tip,
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
				this.Hint = "Preview capture areas"
			})

			this.Text = "🗗×"
			this.Hint = "Close capture area preview"
		},
	}

	ui.navButtons.menu.save = &button.Widget{
		Text:            "🖫",
		Font:            g.nav.NishikiTeki(),
		Pressed:         nrgba.OfficeBlue,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Disabled:        false,
		Hint:            "Save configuration (Ctrl+S)",
		OnHoverHint:     g.nav.Tip,
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

					notify.System("[UI] Configuration saved to %s", config.Current.File())
				}),
				toastOnNo(this.Deactivate),
			)
		},
	}

	ui.navButtons.menu.hide = &button.Widget{
		Text:            "⇊",
		Font:            g.nav.NishikiTeki(),
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Pressed:         nrgba.Gray,
		Hint:            "Hide sources",
		OnHoverHint:     g.nav.Tip,
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			ui.hideOptions = !ui.hideOptions
			if ui.hideOptions {
				this.Text = "⇈"
				this.Hint = "Show sources"
			} else {
				this.Text = "⇊"
				this.Hint = "Hide sources"
			}
		},
	}

	ui.navButtons.menu.capture = &button.Widget{
		Text:            "⛶",
		Font:            g.nav.NishikiTeki(),
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Pressed:         nrgba.DarkSeafoam,
		Hint:            "Test capture areas",
		OnHoverHint:     g.nav.Tip,
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

	ui.navButtons.menu.file = &button.Widget{
		Text:            "📝",
		Font:            g.nav.NishikiTeki(),
		Pressed:         nrgba.CoolBlue,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Disabled:        false,
		Hint:            "Open configuration file",
		OnHoverHint:     g.nav.Tip,
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

	ui.navButtons.menu.reset = &button.Widget{
		Text:            "💣",
		Font:            g.nav.NishikiTeki(),
		Pressed:         nrgba.PaleRed,
		TextSize:        unit.Sp(17),
		TextInsetBottom: -1,
		Disabled:        false,
		Hint:            "Reset configuration (Ctrl+Z)",
		OnHoverHint:     g.nav.Tip,
		Click: func(this *button.Widget) {
			g.ToastYesNo("Reset", fmt.Sprintf("<ini:i:reset> configuration for this profile? (%s)", lang.Title(config.Current.Gaming.Device)),
				toastOnYes(func() {
					server.Clear()

					prev := config.Current
					err := config.Current.Reset()
					if err != nil {
						config.Current = prev
						g.ToastErrorf("[UI] <ini:f:reset> configuration for %s (%v)", lang.Title(config.Current.Gaming.Device), err)
						return
					}

					// Hit the disable.
					ui.groups.videos.devices.list.Callback(ui.groups.videos.devices.list.Items[0], ui.groups.videos.devices.list)

					this.Deactivate()

					video.Close()

					config.Current.Reload()

					audio.Restart()

					ui.groups.areas.energy.Min, ui.groups.areas.energy.Max = config.Current.XY.Energy.Min, config.Current.XY.Energy.Max
					ui.groups.areas.time.Min, ui.groups.areas.time.Max = config.Current.XY.Time.Min, config.Current.XY.Time.Max
					ui.groups.areas.score.Min, ui.groups.areas.score.Max = config.Current.XY.Scores.Min, config.Current.XY.Scores.Max
					ui.groups.areas.objective.Min, ui.groups.areas.objective.Max = config.Current.XY.Objectives.Min, config.Current.XY.Objectives.Max
					// ui.groups.areas.ko.Min, ui.groups.areas.ko.Max = config.Current.XY.KOs.Min, config.Current.XY.KOs.Max
					ui.groups.videos.populate()

					is.Next(is.MainMenu)

					notify.Announce("[UI] <ini:i:reset> UniteHUD configuration profile for %s", lang.Title(config.Current.Gaming.Device))
				}),
				toastOnNo(this.Deactivate),
			)
		},
	}

	ui.navButtons.menu.lock = &button.Widget{
		Text:            "🔓",
		Font:            g.nav.NishikiTeki(),
		Pressed:         nrgba.Gold.Alpha(150),
		TextSize:        unit.Sp(17),
		TextInsetBottom: -1,
		Disabled:        false,
		Hint:            "Lock capture areas to prevent mouse dragging",
		OnHoverHint:     g.nav.Tip,
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

	ui.navButtons.menu.screenshot = &button.Widget{
		Text:            "📸",
		Font:            g.nav.NishikiTeki(),
		Hint:            "Take a screenshot of this window (Ctrl+V)",
		OnHoverHint:     g.nav.Tip,
		Released:        nrgba.Transparent80,
		Pressed:         nrgba.Night,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			w := win32.Window(g.HWND)

			r, err := w.RectClient()
			if err != nil {
				notify.Error("[UI] Failed to determine screenshot dimensions (%v)", err)
				g.ToastErrorf("Failed to capture screenshot (%v)", err)
				return
			}

			img, err := w.Capture(r.Image(), image.Point{})
			if err != nil {
				notify.Error("[UI] Failed to capture screenshot (%v)", err)
				g.ToastErrorf("Failed to capture screenshot (%v)", err)
				return
			}

			file := fmt.Sprintf("Screenshot_%s.png", save.KitchenTime())

			err = save.PNG(img, file)
			if err != nil {
				notify.Error("[UI] Failed to save screenshot (%v)", err)
				g.ToastErrorf("Failed to capture screenshot (%v)", err)
				return
			}

			err = save.OpenImage(file)
			if err != nil {
				notify.Error("[UI] Failed to open screenshot (%v)", err)
				g.ToastErrorf("Failed to capture screenshot (%v)", err)
				return
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

	ui.labels.video.monCapture = material.Label(g.nav.Calibri().Theme, unit.Sp(12), "Monitor Capture Method")
	ui.labels.video.monCapture.Color = nrgba.Highlight.Color()
	ui.labels.video.monCapture.Font.Weight = 100

	ui.labels.video.winCapture = material.Label(g.nav.Calibri().Theme, unit.Sp(12), "Window Capture Method")
	ui.labels.video.winCapture.Color = nrgba.Highlight.Color()
	ui.labels.video.winCapture.Font.Weight = 100

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

	// if !g.dimensions.fullscreen {
	ui.navButtons.menu.settings.Click(ui.navButtons.menu.settings)
	// }

	return ui
}

// Layout renders the live capture preview and computes capture bounds.
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

// empty returns a spacer child with the given dimensions.
func (ui *configure) empty(x, y float32) layout.FlexChild {
	return layout.Rigid(layout.Spacer{Width: unit.Dp(x), Height: unit.Dp(y)}.Layout)
}

// foot builds the footer bar showing status and performance metrics.
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

// spacer draws a decorated spacer with optional fixed width/height.
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
