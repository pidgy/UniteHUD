package gui

import (
	"fmt"
	"image"
	"os/exec"
	"time"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/split"
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/nrgba"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/video/device"
	"github.com/pidgy/unitehud/video/monitor"
	"github.com/pidgy/unitehud/video/window/electron"
)

func (g *GUI) configure() (next string, err error) {
	defer server.SetConfig(false)

	g.Preview = true
	defer func() {
		g.Preview = false
	}()

	split := &split.Horizontal{
		Ratio:      .6,
		Adjustable: true,
	}

	areas := g.areas()
	areas.onevent = func() {
		areas.score.Reset()
		areas.energy.Reset()
		areas.time.Reset()
		areas.ko.Reset()
		areas.objective.Reset()

		areas.energy.Base = config.Current.Energy
		areas.score.Base = config.Current.Scores
		areas.time.Base = config.Current.Time
		areas.objective.Base = config.Current.Objectives
		areas.ko.Base = config.Current.KOs
	}

	grp := g.videos(12)
	grp.platform.list.WidthModifier = 8
	grp.profile.list.WidthModifier = 8
	grp.monitor.list.WidthModifier = 6
	grp.window.list.WidthModifier = 2
	grp.onevent = func() {
		config.Current.SetDefaultAreas()

		areas.onevent()
	}

	scaleText := material.H5(g.cascadia, "Scale")
	scaleText.Color = nrgba.White.Color()
	scaleText.Alignment = text.Middle
	scaleText.TextSize = unit.Sp(11)

	scaleValueText := material.H5(g.cascadia, "1x")
	scaleValueText.Color = nrgba.White.Color()
	scaleValueText.Alignment = text.Middle
	scaleValueText.TextSize = unit.Sp(11)

	scaleUpButton := &button.Button{
		Text:        "+",
		Pressed:     nrgba.LightGray,
		Released:    nrgba.Night,
		Size:        image.Pt(30, 30),
		TextSize:    unit.Sp(12),
		BorderWidth: unit.Sp(1.5),
		Click: func(b *button.Button) {
			defer b.Deactivate()

			g.Preview = false
			defer g.buttonSpam(b)

			config.Current.Scale += .01
		},
	}

	scaleDownButton := &button.Button{
		Text:        "-",
		Pressed:     nrgba.LightGray,
		Released:    nrgba.Night,
		Size:        image.Pt(30, 30),
		TextSize:    unit.Sp(12),
		BorderWidth: unit.Sp(1.5),
		Click: func(b *button.Button) {
			defer b.Deactivate()

			g.Preview = false
			defer g.buttonSpam(b)

			config.Current.Scale -= .01
			if config.Current.Scale < 1 {
				config.Current.Scale = 1
			}
		},
	}

	nButton := &button.Button{
		Text:        "^",
		Pressed:     nrgba.LightGray,
		Released:    nrgba.Night,
		Size:        image.Pt(30, 20),
		TextSize:    unit.Sp(18),
		SingleClick: true,
		BorderWidth: unit.Sp(1.5),
		Click: func(b *button.Button) {
			config.Current.Shift.N++
		},
	}

	eButton := &button.Button{
		Text:     ">",
		Pressed:  nrgba.LightGray,
		Released: nrgba.Night,
		Size:     image.Pt(28, 23),
		TextSize: unit.Sp(12),

		BorderWidth: unit.Sp(1.5),
		SingleClick: true,
		Click: func(b *button.Button) {
			config.Current.Shift.E++
		},
	}

	sButton := &button.Button{
		Text:     "v",
		Pressed:  nrgba.LightGray,
		Released: nrgba.Night,
		Size:     image.Pt(30, 20),
		TextSize: unit.Sp(12),

		SingleClick: true,
		BorderWidth: unit.Sp(1.5),
		Click: func(b *button.Button) {
			config.Current.Shift.S++
		},
	}

	wButton := &button.Button{
		Text:     "<",
		Pressed:  nrgba.LightGray,
		Released: nrgba.Night,
		Size:     image.Pt(28, 23),
		TextSize: unit.Sp(12),

		SingleClick: true,
		BorderWidth: unit.Sp(1.5),
		Click: func(b *button.Button) {
			config.Current.Shift.W++
		},
	}

	shiftText := material.H5(g.cascadia, "Shift")
	shiftText.Color = nrgba.White.Color()
	shiftText.Alignment = text.Middle
	shiftText.TextSize = unit.Sp(11)

	assetsButton := &button.Button{
		Text:        "Assets",
		Pressed:     nrgba.Gray,
		Released:    nrgba.Transparent30,
		Size:        image.Pt(100, 30),
		SingleClick: true,
		BorderWidth: unit.Sp(1.5),
		Click: func(b *button.Button) {
			exec.Command(`explorer`, config.Current.ProfileAssets()).Run()
		},
	}

	previewButton := &button.Button{
		Text:        "Preview",
		Pressed:     nrgba.Gray,
		Released:    nrgba.Transparent30,
		Size:        image.Pt(100, 30),
		SingleClick: true,
		BorderWidth: unit.Sp(1.5),
		Click: func(b *button.Button) {
			g.previewCaptures(areas)
		},
	}

	defaultButton := &button.Button{
		Text:        "Default",
		Pressed:     nrgba.Gray,
		Released:    nrgba.Transparent30,
		Size:        image.Pt(100, 30),
		BorderWidth: unit.Sp(1.5),
		Click: func(b *button.Button) {
			defer b.Deactivate()

			areas.onevent()
		},
	}

	openConfigFileButton := &button.Button{
		Text:        "Edit",
		Pressed:     nrgba.Gray,
		Released:    nrgba.Transparent30,
		Size:        image.Pt(100, 30),
		BorderWidth: unit.Sp(1.5),
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

	saveButton := &button.Button{
		Text:     "Save",
		Pressed:  nrgba.ForestGreen,
		Released: nrgba.Transparent30,
		Size:     image.Pt(100, 30),
	}

	// Hold on to the previous configuration to overwrite memory saves.
	prev := config.Current

	cancelButton := &button.Button{
		Text:        "Cancel",
		Pressed:     nrgba.BloodOrange.Alpha(0x5F),
		Released:    nrgba.Transparent30,
		Size:        image.Pt(100, 30),
		BorderWidth: unit.Sp(1.5),
		Click: func(this *button.Button) {
			g.ToastYesNo("Cancel", "Discard configuration changes?",
				func() {
					defer this.Deactivate()

					server.Clear()

					if prev.Window != config.Current.Window {
						electron.Close()
					}

					this.Disabled = true
					saveButton.Disabled = true

					config.Current = prev
					err := config.Current.Save()
					if err != nil {
						notify.Error("Failed to save UniteHUD configuration (%v)", err)
					}

					notify.System("Configuration omitted")

					g.Actions <- Refresh

					next = "main"
				}, this.Deactivate)
		},
	}

	saveButton.Click = func(b *button.Button) {
		g.ToastYesNo("Save", "Save configuration changes?",
			func() {
				defer saveButton.Deactivate()

				server.Clear()

				cancelButton.Disabled = true
				saveButton.Disabled = true

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

				next = "main"
			}, saveButton.Deactivate)
	}

	resetButton := &button.Button{
		Text:        "Reset",
		Pressed:     nrgba.DarkRed,
		Released:    nrgba.Transparent30,
		BorderWidth: unit.Sp(1.5),
		Size:        image.Pt(100, 30),
		Click: func(b *button.Button) {
			g.ToastYesNo("Reset", fmt.Sprintf("Reset UniteHUD %s configuration?", config.Current.Profile), func() {
				defer b.Deactivate()
				defer server.Clear()

				grp.device.list.Callback(grp.device.list.Items[0], grp.device.list)

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

				grp.window.populate(true)
				grp.device.populate(true)
				grp.monitor.populate(true)

				g.Actions <- Refresh

				next = "main"

				notify.Announce("Reset UniteHUD %s configuration", config.Current.Profile)
			}, b.Deactivate)
		},
	}
	// openBrowserButton := &button.Button{
	// 	Text:        "Browser",
	// 	Pressed:     nrgba.Gray,
	// 	Released:    nrgba.Transparent30,
	// 	BorderWidth: unit.Sp(1.5),
	// 	Size:        image.Pt(100, 30),
	// 	Click: func(b *button.Button) {

	// 		// defer b.Deactivate()

	// 		// err := g.ToastInput(
	// 		// 	fmt.Sprintf("%s URL", config.BrowserWindow),
	// 		// 	"https://youtube.com/watch?v=t2kzUcEQa3g",
	// 		// 	fmt.Sprintf("Disable %s auto-hide?", config.BrowserWindow),
	// 		// 	func(url string, option bool) {
	// 		// 		electron.Close()
	// 		// 		video.Close()
	// 		// 		time.Sleep(time.Second) // XXX: Fix concurrency error in device.go Close.

	// 		// 		if device.IsActive() {
	// 		// 			config.Current.VideoCaptureDevice = config.NoVideoCaptureDevice
	// 		// 			grp.device.populate(true)
	// 		// 		} else {
	// 		// 			grp.device.populate(false)
	// 		// 		}

	// 		// 		config.Current.DisableBrowserFormatting = option

	// 		// 		err = electron.Open()
	// 		// 		if err != nil {
	// 		// 			g.ToastErrorForce(err)
	// 		// 			return
	// 		// 		}

	// 		// 		config.Current.Window = config.BrowserWindow
	// 		// 	})
	// 		// if err != nil {
	// 		// 	notify.Error("Failed to open url dialog (%v)", err)
	// 		// }
	// 	},
	// }

	// closeBrowserButton := &button.Button{
	// 	Text:     "Browser",
	// 	Pressed:  nrgba.PaleRed,
	// 	Released: nrgba.Transparent30,
	// 	Size:     image.Pt(100, 30),
	// 	Click: func(b *button.Button) {
	// 		g.ToastYesNo("Cancel", fmt.Sprintf("Close %s?", config.BrowserWindow), func() {
	// 			defer b.Deactivate()

	// 			electron.Close()
	// 			config.Current.Window = config.MainDisplay
	// 		}, b.Deactivate)
	// 	},
	// }

	header := material.H5(g.cascadia, title.Default)
	header.Color = nrgba.Black.Color()
	header.Alignment = text.Middle
	header.Font.Weight = font.ExtraBold

	/*
		go g.while(func() { g.matchKOs(koArea) }, &matchingRoutines)
		go g.while(func() { g.matchObjectives(objectiveArea) }, &matchingRoutines)
		go g.while(func() { g.matchTime(timeArea) }, &matchingRoutines)

		if config.Current.Profile == config.ProfilePlayer {
			go g.while(func() { g.matchEnergy(energyArea) }, &matchingRoutines)
			go g.while(func() { g.matchScore(scoreArea) }, &matchingRoutines)
		}
	*/
	// go g.run(func() { g.matchMap(mapArea) }, &kill)

	var ops op.Ops

	populateTicksThreshold := 120
	populateTicks := populateTicksThreshold

	for next == "" {
		if !g.open {
			time.Sleep(time.Millisecond * 10)
			continue
		}

		if populateTicks++; populateTicks > populateTicksThreshold {
			grp.window.populate(false)
			grp.device.populate(false)
			grp.monitor.populate(false)
			populateTicks = 0
		}

		e := <-g.Events()
		switch e := e.(type) {
		case app.ViewEvent:
			g.HWND = e.HWND
		case app.ConfigEvent:
		case system.DestroyEvent:
			return "", nil
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)

			background := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
			paint.ColorOp{Color: nrgba.Background.Color()}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			background.Pop()

			g.Bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return split.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						// img, err := video.Capture()
						// if err != nil {
						// 	g.ToastError(err)
						// 	next = "main"
						// }

						// g.Screen = &screen.Screen{
						// 	Image:         img,
						// 	VerticalScale: true,
						// 	Splash:        true,
						// }

						// return g.Screen.Layout(gtx)

						return layout.Dimensions{Size: gtx.Constraints.Max}
					},
					func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(0)).Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								return fill(gtx, nrgba.BackgroundAlt,
									func(gtx layout.Context) layout.Dimensions {
										{
											layout.Inset{
												Left:  unit.Dp(10),
												Top:   unit.Dp(101),
												Right: unit.Dp(10),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													return saveButton.Layout(gtx)
												},
											)

											layout.Inset{
												Left:  unit.Dp(115),
												Top:   unit.Dp(101),
												Right: unit.Dp(10),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													return cancelButton.Layout(gtx)
												},
											)

											// if config.Current.Window == config.BrowserWindow {
											// 	layout.Inset{
											// 		Top:   unit.Dp(5),
											// 		Left:  unit.Dp(325),
											// 		Right: unit.Dp(10),
											// 	}.Layout(
											// 		gtx,
											// 		func(gtx layout.Context) layout.Dimensions {
											// 			return closeBrowserButton.Layout(gtx)
											// 		},
											// 	)
											// } else {
											// 	layout.Inset{
											// 		Top:  unit.Dp(5),
											// 		Left: unit.Dp(220),
											// 	}.Layout(
											// 		gtx,
											// 		func(gtx layout.Context) layout.Dimensions {
											// 			return openBrowserButton.Layout(gtx)
											// 		},
											// 	)
											// }

											layout.Inset{
												Left: unit.Dp(115),
												Top:  unit.Dp(5),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													return defaultButton.Layout(gtx)
												},
											)

											layout.Inset{
												Left:  unit.Dp(220),
												Top:   unit.Dp(101),
												Right: unit.Dp(10),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													return resetButton.Layout(gtx)
												},
											)

											layout.Inset{
												Left: unit.Dp(10),
												Top:  unit.Dp(5),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													return openConfigFileButton.Layout(gtx)
												},
											)
										}

										// Platform device / Profile configuration.
										{
											left := unit.Dp(float32(gtx.Constraints.Max.X - 687))

											layout.Inset{
												Left: left,
												Top:  unit.Dp(3),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													platformListTitle := material.Label(g.cascadia, unit.Sp(14), "Platform")
													platformListTitle.Color = nrgba.Slate.Color()
													return platformListTitle.Layout(gtx)
												},
											)

											layout.Inset{
												Left: left,
												Top:  unit.Dp(20),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													return widget.Border{
														Color: nrgba.LightGray.Color(),
														Width: unit.Dp(2),
													}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
														return grp.platform.list.Layout(gtx, g.normal)
													})
												},
											)

											layout.Inset{
												Left: left,
												Top:  unit.Dp(82),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													profileListTitle := material.Label(g.cascadia, unit.Sp(14), "Profile")
													profileListTitle.Color = nrgba.Slate.Color()
													return profileListTitle.Layout(gtx)
												},
											)

											layout.Inset{
												Left: left,
												Top:  unit.Dp(99),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													return widget.Border{
														Color: nrgba.LightGray.Color(),
														Width: unit.Dp(2),
													}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
														return grp.profile.list.Layout(gtx, g.normal)
													})
												},
											)
										}

										// Screen capture list.
										{
											left := unit.Dp(float32(gtx.Constraints.Max.X - 586))

											layout.Inset{
												Left: left,
												Top:  unit.Dp(3),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													screenListTitle := material.Label(g.cascadia, unit.Sp(14), "Screen")
													screenListTitle.Color = nrgba.Slate.Color()
													return screenListTitle.Layout(gtx)
												},
											)

											layout.Inset{
												Left:   left,
												Top:    unit.Dp(20),
												Bottom: unit.Dp(3),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													return widget.Border{
														Color: nrgba.LightGray.Color(),
														Width: unit.Dp(2),
													}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
														return grp.monitor.list.Layout(gtx, g.normal)
													})
												},
											)
										}

										// Window capture list.
										{
											left := unit.Dp(float32(gtx.Constraints.Max.X - 473))

											layout.Inset{
												Left: left,
												Top:  unit.Dp(3),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													windowListTitle := material.Label(g.cascadia, unit.Sp(14), "Window")
													windowListTitle.Color = nrgba.Slate.Color()
													return windowListTitle.Layout(gtx)
												},
											)

											layout.Inset{
												Left:   left,
												Top:    unit.Dp(20),
												Bottom: unit.Dp(3),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													return widget.Border{
														Color: nrgba.LightGray.Color(),
														Width: unit.Dp(2),
													}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
														return grp.window.list.Layout(gtx, g.normal)
													})
												},
											)
										}

										// Capture device list.
										{
											left := unit.Dp(float32(gtx.Constraints.Max.X - 225))

											layout.Inset{
												Left: left,
												Top:  unit.Dp(3),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													deviceListTitle := material.Label(g.cascadia, unit.Sp(14), "Device")
													deviceListTitle.Color = nrgba.Slate.Color()
													return deviceListTitle.Layout(gtx)
												},
											)

											layout.Inset{
												Left:   left,
												Top:    unit.Dp(20),
												Bottom: unit.Dp(3),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													return widget.Border{
														Color: nrgba.LightGray.Color(),
														Width: unit.Dp(2),
													}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
														return grp.device.list.Layout(gtx, g.normal)
													})
												},
											)
										}

										// Time area rectangle buttons.
										/*
											{
												layout.Inset{
													Top: unit.Dp(5),
													Left: unit.Dp(220),
												}.Layout(
													gtx,
													func(gtx layout.Context) layout.Dimensions {
														return areas.time.Button.Layout(gtx)
													},
												)
											}
										*/

										layout.Inset{
											Top:  unit.Dp(37),
											Left: unit.Dp(220),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return assetsButton.Layout(gtx)
											},
										)

										layout.Inset{
											Top:  unit.Dp(69),
											Left: unit.Dp(220),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return previewButton.Layout(gtx)
											},
										)

										{
											if device.IsActive() {
												scaleText.Color = nrgba.Slate.Color()
												scaleValueText.Color = nrgba.Slate.Color()
												scaleUpButton.Disabled = true
												scaleDownButton.Disabled = true
											} else {
												scaleText.Color = nrgba.White.Color()
												scaleValueText.Color = nrgba.White.Color()
												scaleUpButton.Disabled = false
												scaleDownButton.Disabled = false
											}

											layout.Inset{
												Left: unit.Dp(10),
												Top:  unit.Dp(38),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													scaleDownButton.Disabled = false
													if config.Current.Scale == 1 {
														scaleDownButton.Disabled = true
													}

													return scaleDownButton.Layout(gtx)
												},
											)

											layout.Inset{
												Left: unit.Dp(45),
												Top:  unit.Dp(38),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													return scaleText.Layout(gtx)
												},
											)

											layout.Inset{
												Left: unit.Dp(45),
												Top:  unit.Dp(55),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													if device.IsActive() {
														scaleValueText.Text = "1.00x"
													} else {
														scaleValueText.Text = fmt.Sprintf("%.2fx", config.Current.Scale)
													}

													return scaleValueText.Layout(gtx)
												},
											)

											layout.Inset{
												Left: unit.Dp(80),
												Top:  unit.Dp(38),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													return scaleUpButton.Layout(gtx)
												},
											)
										}

										// Shift N,E,S,W
										{
											if device.IsActive() || monitor.IsDisplay() {
												nButton.Disabled = true
												eButton.Disabled = true
												sButton.Disabled = true
												wButton.Disabled = true
												shiftText.Color = nrgba.Slate.Color()
											} else {
												nButton.Disabled = false
												eButton.Disabled = false
												sButton.Disabled = false
												wButton.Disabled = false
												shiftText.Color = nrgba.White.Color()
											}

											layout.Inset{
												Left: unit.Dp(150),
												Top:  unit.Dp(37),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													return nButton.Layout(gtx)
												},
											)

											layout.Inset{
												Left: unit.Dp(184),
												Top:  unit.Dp(55),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													return eButton.Layout(gtx)
												},
											)

											layout.Inset{
												Left: unit.Dp(150),
												Top:  unit.Dp(78),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													return sButton.Layout(gtx)
												},
											)

											layout.Inset{
												Left: unit.Dp(118),
												Top:  unit.Dp(55),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													return wButton.Layout(gtx)
												},
											)

											layout.Inset{
												Left: unit.Dp(150),
												Top:  unit.Dp(60),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													return shiftText.Layout(gtx)
												},
											)
										}

										return layout.Dimensions{Size: gtx.Constraints.Max}
									},
								)
							},
						)
					},
				)
			})

			// cursor := pointer.CursorDefault
			// for _, area := range []*area.Area{areas.time, areas.energy, areas.score, areas.ko, areas.objective, areas.state} {
			// 	err := area.Layout(gtx, layout.Constraints{Max: g.max})
			// 	if err != nil {
			// 		g.ToastError(err)
			// 	}
			// 	if area.Focus {
			// 		cursor = pointer.CursorPointer
			// 	}
			// 	if area.Drag {
			// 		cursor = pointer.CursorCrosshair
			// 	}
			// }
			// _ = cursor
			// g.Window.SetCursorName(cursor)

			e.Frame(gtx.Ops)
		}
	}

	return next, nil
}

// buttonSpam ensures we only execute a config reload once before cooling down.
func (g *GUI) buttonSpam(b *button.Button) {
	b.LastPressed = time.Now()

	time.AfterFunc(time.Second, func() {
		if time.Since(b.LastPressed) >= time.Second {
			config.Current.Reload()
			g.Preview = true
		}
	},
	)
}
