package gui

import (
	"fmt"
	"image"
	"image/color"
	"os/exec"
	"runtime"
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

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/gui/visual/area"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/dropdown"
	"github.com/pidgy/unitehud/gui/visual/split"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/video"
	"github.com/pidgy/unitehud/video/device"
	"github.com/pidgy/unitehud/video/screen"
	"github.com/pidgy/unitehud/video/window/electron"
)

func (g *GUI) configure() (next string, err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	defer server.SetConfig(false)

	g.Preview = true
	defer func() {
		g.Preview = false
	}()

	split := &split.Horizontal{
		Ratio: .6,
	}

	lockedAreaText := "Locked"
	lockedAreaButtonText := "\tLocked"

	koArea := &area.Area{
		Text:     "KOs",
		Subtext:  lockedAreaText,
		TextSize: unit.Sp(13),
		Min:      config.Current.KOs.Min,
		Max:      config.Current.KOs.Max,
		NRGBA:    area.Locked,
		Match:    g.matchKOs,
		Cooldown: time.Millisecond * 1500,

		Capture: &area.Capture{
			Option: "KO area",
			File:   "ko_area.png",
			Base:   config.Current.KOs,
		},

		Button: &button.Button{
			Active: false,
		},
	}

	objectiveArea := &area.Area{
		Text:     "Objectives",
		Subtext:  lockedAreaText,
		TextSize: unit.Sp(13),
		Min:      config.Current.Objectives.Min,
		Max:      config.Current.Objectives.Max,
		NRGBA:    area.Locked,
		Match:    g.matchObjectives,
		Cooldown: time.Second,

		Capture: &area.Capture{
			Option: "Objective area",
			File:   "objective_area.png",
			Base:   config.Current.Objectives,
		},

		Button: &button.Button{
			Active: false,
		},
	}

	energyArea := &area.Area{
		Text:     "Aeos",
		Subtext:  lockedAreaText,
		TextSize: unit.Sp(13),
		Min:      config.Current.Energy.Min,
		Max:      config.Current.Energy.Max,
		NRGBA:    area.Locked,
		Match:    g.matchEnergy,
		Cooldown: team.Energy.Delay,

		Capture: &area.Capture{
			Option: "Aeos area",
			File:   "aeos_area.png",
			Base:   config.Current.Energy,
		},
	}
	energyArea.Button = &button.Button{
		Active:   false,
		Text:     fmt.Sprintf("\t  %s", energyArea.Text),
		Pressed:  rgba.N(rgba.Night),
		Released: rgba.N(rgba.DarkGray),
		Size:     image.Pt(100, 30),
		Click: func(b *button.Button) {
			energyArea.Subtext = ""
			energyArea.NRGBA.A = 0x4F

			if !energyArea.Button.Active {
				energyArea.Subtext = lockedAreaText
				energyArea.Button.Text = lockedAreaButtonText
				energyArea.NRGBA.A = 0x9
			}
		},
	}

	timeArea := &area.Area{
		Text:     "\tTime",
		Subtext:  lockedAreaText,
		TextSize: unit.Sp(12),
		Min:      config.Current.Time.Min,
		Max:      config.Current.Time.Max,
		NRGBA:    area.Locked,
		Match:    g.matchTime,
		Cooldown: team.Time.Delay,

		Capture: &area.Capture{
			Option: "Time area",
			File:   "time_area.png",
			Base:   config.Current.Time,
		},
	}
	timeArea.Button = &button.Button{
		Active:   false,
		Text:     fmt.Sprintf("  %s", timeArea.Text),
		Pressed:  rgba.N(rgba.Night),
		Released: rgba.N(rgba.DarkGray),
		Size:     image.Pt(100, 30),
		Click: func(b *button.Button) {
			timeArea.Subtext = ""
			timeArea.NRGBA.A = 0x4F

			if !timeArea.Button.Active {
				timeArea.Subtext = lockedAreaText
				timeArea.Button.Text = lockedAreaButtonText
				timeArea.NRGBA.A = 0x9
			}
		},
	}

	scoreArea := &area.Area{
		Text:          "Score",
		Subtext:       lockedAreaText,
		TextAlignLeft: true,
		Min:           config.Current.Scores.Min,
		Max:           config.Current.Scores.Max,
		NRGBA:         area.Locked,
		Match:         g.matchScore,
		Cooldown:      team.Purple.Delay,

		Capture: &area.Capture{
			Option: "Score area",
			File:   "score_area.png",
			Base:   config.Current.Scores,
		},

		Theme: g.normal,
	}
	scoreArea.Button = &button.Button{
		Active:   false,
		Text:     fmt.Sprintf("\t %s", scoreArea.Text),
		Pressed:  rgba.N(rgba.Night),
		Released: rgba.N(rgba.DarkGray),
		Size:     image.Pt(100, 30),
		Click: func(b *button.Button) {
			scoreArea.Subtext = ""
			scoreArea.NRGBA.A = 0x4F

			if !scoreArea.Button.Active {
				scoreArea.Subtext = lockedAreaText
				scoreArea.Button.Text = lockedAreaButtonText
				scoreArea.NRGBA.A = 0x9
			}
		},
	}

	scaleText := material.H5(g.cascadia, "Scale")
	scaleText.Color = rgba.N(rgba.White)
	scaleText.Alignment = text.Middle
	scaleText.TextSize = unit.Sp(11)

	scaleValueText := material.H5(g.cascadia, "1x")
	scaleValueText.Color = rgba.N(rgba.White)
	scaleValueText.Alignment = text.Middle
	scaleValueText.TextSize = unit.Sp(11)

	scaleUpButton := &button.Button{
		Text:     "+",
		Pressed:  color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released: color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:     image.Pt(30, 30),
		TextSize: unit.Sp(12),
		Click: func(b *button.Button) {
			defer b.Deactivate()

			g.Preview = false
			defer g.buttonSpam(b)

			config.Current.Scale += .01
		},
	}

	scaleDownButton := &button.Button{
		Text:     "-",
		Pressed:  color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released: color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:     image.Pt(30, 30),
		TextSize: unit.Sp(12),
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
		Pressed:     color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released:    color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:        image.Pt(30, 20),
		TextSize:    unit.Sp(18),
		SingleClick: true,
		Click: func(b *button.Button) {
			config.Current.Shift.N++
		},
	}

	eButton := &button.Button{
		Text:          ">",
		Pressed:       color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released:      color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:          image.Pt(28, 23),
		TextSize:      unit.Sp(12),
		TextOffsetTop: -2,
		SingleClick:   true,
		Click: func(b *button.Button) {
			config.Current.Shift.E++
		},
	}

	sButton := &button.Button{
		Text:          "v",
		Pressed:       color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released:      color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:          image.Pt(30, 20),
		TextSize:      unit.Sp(12),
		TextOffsetTop: -2,
		SingleClick:   true,
		Click: func(b *button.Button) {
			config.Current.Shift.S++
		},
	}

	wButton := &button.Button{
		Text:          "<",
		Pressed:       color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released:      color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:          image.Pt(28, 23),
		TextSize:      unit.Sp(12),
		TextOffsetTop: -2,
		SingleClick:   true,
		Click: func(b *button.Button) {
			config.Current.Shift.W++
		},
	}

	shiftText := material.H5(g.cascadia, "Shift")
	shiftText.Color = rgba.N(rgba.White)
	shiftText.Alignment = text.Middle
	shiftText.TextSize = unit.Sp(11)

	assetsButton := &button.Button{
		Active:      true,
		Text:        "\t Assets",
		Pressed:     rgba.N(rgba.Gray),
		Released:    rgba.N(rgba.DarkGray),
		Size:        image.Pt(100, 30),
		SingleClick: true,
		Click: func(b *button.Button) {
			exec.Command(`explorer`, config.Current.ProfileAssets()).Run()
		},
	}

	captureButton := &button.Button{
		Active:      true,
		Text:        "\tCapture",
		Pressed:     rgba.N(rgba.Gray),
		Released:    rgba.N(rgba.DarkGray),
		Size:        image.Pt(100, 30),
		SingleClick: true,
		Click: func(b *button.Button) {
			g.ToastCapture([]*area.Capture{
				{Option: "Entire area", File: "screen_area.png", Base: g.Screen.Bounds()},
				scoreArea.Capture,
				energyArea.Capture,
				timeArea.Capture,
				objectiveArea.Capture,
				koArea.Capture,
			})
		},
	}

	reapplyAreas := func() {
		energyArea.Base = config.Current.Energy
		scoreArea.Base = config.Current.Scores
		timeArea.Base = config.Current.Time
		objectiveArea.Base = config.Current.Objectives
		koArea.Base = config.Current.KOs
	}

	defaultButton := &button.Button{
		Text:     "\tDefault",
		Pressed:  rgba.N(rgba.Gray),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
		Click: func(b *button.Button) {
			defer b.Deactivate()

			config.Current.SetDefaultAreas()

			scoreArea.Reset()
			energyArea.Reset()
			timeArea.Reset()
			koArea.Reset()
			objectiveArea.Reset()

			reapplyAreas()
		},
	}

	openConfigFileButton := &button.Button{
		Text:     "\t   Edit",
		Pressed:  rgba.N(rgba.Gray),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
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

			reapplyAreas()
		},
	}

	saveButton := &button.Button{
		Text:     "\t  Save",
		Pressed:  rgba.N(rgba.ForestGreen),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
	}

	// Hold on to the previous configuration to overwrite memory saves.
	prev := config.Current

	cancelButton := &button.Button{
		Text:     "\tCancel",
		Pressed:  rgba.N(rgba.Alpha(rgba.BloodOrange, 0x5F)),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
		Click: func(b *button.Button) {
			g.ToastYesNo("Cancel", "Discard configuration changes?",
				func() {
					defer b.Deactivate()

					server.Clear()

					if prev.Window != config.Current.Window {
						electron.Close()
					}

					b.Disabled = true
					saveButton.Disabled = true
					energyArea.Button.Disabled = true
					timeArea.Button.Disabled = true
					scoreArea.Button.Disabled = true

					config.Current = prev
					err := config.Current.Save()
					if err != nil {
						notify.Error("Failed to save UniteHUD configuration (%v)", err)
					}

					if config.Current.Window == config.BrowserWindow {
						err = electron.Open()
						if err != nil {
							notify.Error("Failed to open %s (%v)", config.BrowserWindow, err)
						}
					}

					notify.System("Configuration omitted")

					g.Actions <- Refresh

					next = "main"
				}, b.Deactivate)
		},
	}

	saveButton.Click = func(b *button.Button) {
		g.ToastYesNo("Save", "Save configuration changes?",
			func() {
				defer saveButton.Deactivate()

				server.Clear()

				cancelButton.Disabled = true
				saveButton.Disabled = true
				energyArea.Button.Disabled = true
				timeArea.Button.Disabled = true
				scoreArea.Button.Disabled = true

				config.Current.Scores = scoreArea.Rectangle()
				config.Current.Time = timeArea.Rectangle()
				config.Current.Energy = energyArea.Rectangle()
				config.Current.Objectives = objectiveArea.Rectangle()
				config.Current.KOs = koArea.Rectangle()

				err := config.Current.Save()
				if err != nil {
					notify.Error("Failed to save UniteHUD configuration (%v)", err)
				}

				notify.System("Configuration saved to " + config.Current.File())

				g.Actions <- Refresh

				next = "main"
			}, saveButton.Deactivate)
	}

	screenButton := &button.Button{
		Text:     "\tPreview",
		Pressed:  rgba.N(rgba.Gray),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
		Click: func(b *button.Button) {
			g.Preview = !g.Preview
		},
	}

	windowList := &dropdown.List{
		Items: []*dropdown.Item{},
		Callback: func(i *dropdown.Item) {
			if device.IsActive() {
				return
			}

			electron.Close()

			if i.Text == "" {
				config.Current.Window = config.MainDisplay
			} else {
				config.Current.Window = i.Text
			}
		},
		WidthModifier: 2,
	}

	populateWindows := func(videoCaptureDisabledEvent bool) {
		for _, item := range windowList.Items {
			item.Checked.Value = config.Current.Window == item.Text
		}

		windows, _, screens := video.Sources()
		if len(windows)+len(screens) == len(windowList.Items) && !videoCaptureDisabledEvent {
			return
		}

		windowList.Items = []*dropdown.Item{}

		if videoCaptureDisabledEvent && config.Current.Window == "" {
			config.Current.Window = config.MainDisplay
		}

		for _, screen := range screens {
			windowList.Items = append(windowList.Items,
				&dropdown.Item{
					Text:     screen,
					Disabled: device.IsActive(),
					Checked:  widget.Bool{Value: screen == config.Current.Window},
				},
			)
		}

		for _, win := range windows {
			windowList.Items = append(windowList.Items,
				&dropdown.Item{
					Text:     win,
					Disabled: device.IsActive(),
					Checked:  widget.Bool{Value: win == config.Current.Window},
				},
			)
		}
	}

	deviceList := &dropdown.List{
		WidthModifier: 3,
	}

	populateDevices := func(videoCaptureDisabledEvent bool) {
		_, devices, _ := video.Sources()

		// Set the "Disabled" checkbox when device is not active.
		if len(devices)+1 == len(deviceList.Items) && !videoCaptureDisabledEvent {
			deviceList.Items[0].Checked.Value = !device.IsActive()
			return
		}

		deviceList.Items = []*dropdown.Item{
			{
				Text:  "Disabled",
				Value: config.NoVideoCaptureDevice,
				Checked: widget.Bool{
					Value: device.IsActive(),
				},
			},
		}
		for _, d := range devices {
			deviceList.Items = append(deviceList.Items, &dropdown.Item{
				Text:  device.Name(d),
				Value: d,
			},
			)
		}

		for _, i := range deviceList.Items {
			i.Checked.Value = false
			if i.Value == config.Current.VideoCaptureDevice {
				i.Checked.Value = true
			}
		}
	}

	deviceList = &dropdown.List{
		Items: []*dropdown.Item{
			{
				Text:  "Disabled",
				Value: config.NoVideoCaptureDevice,
				Checked: widget.Bool{
					Value: device.IsActive(),
				},
			},
		},
		Callback: func(i *dropdown.Item) {
			electron.Close()
			video.Close()
			time.Sleep(time.Second) // XXX: Fix concurrency error in device.go Close.

			config.Current.VideoCaptureDevice = i.Value

			if i.Text == "Disabled" {
				i.Checked = widget.Bool{Value: true}
				populateDevices(true)
				populateWindows(true)
			} else {
				populateWindows(true)
			}

			go func() {
				err = video.Open()
				if err != nil {
					g.ToastErrorForce(err)

					config.Current.Window = config.MainDisplay
					config.Current.VideoCaptureDevice = config.NoVideoCaptureDevice

					populateWindows(true)
					populateDevices(true)
					return
				}

				config.Current.LostWindow = ""
			}()
		},
		WidthModifier: 1,
	}

	populateDevices(false)

	resetButton := &button.Button{
		Text:     "\t Reset",
		Pressed:  rgba.N(rgba.DarkRed),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
		Click: func(b *button.Button) {
			g.ToastYesNo("Reset", fmt.Sprintf("Reset %s configuration?", config.Current.Profile), func() {
				defer b.Deactivate()
				defer server.Clear()

				deviceList.Callback(deviceList.Items[0])

				electron.Close()

				err = config.Current.Reset()
				if err != nil {
					notify.Error("Failed to reset %s configuration (%v)", config.Current.Profile, err)
				}

				config.Current.Reload()

				energyArea.Min, energyArea.Max = config.Current.Energy.Min, config.Current.Energy.Max
				timeArea.Min, timeArea.Max = config.Current.Time.Min, config.Current.Time.Max
				scoreArea.Min, scoreArea.Max = config.Current.Scores.Min, config.Current.Scores.Max
				objectiveArea.Min, objectiveArea.Max = config.Current.Objectives.Min, config.Current.Objectives.Max
				koArea.Min, koArea.Max = config.Current.KOs.Min, config.Current.KOs.Max

				populateWindows(true)
				populateDevices(true)

				g.Actions <- Refresh

				next = "main"

				notify.Announce("Reset %s configuration", config.Current.Profile)
			}, b.Deactivate)
		},
	}
	openBrowserButton := &button.Button{
		Text:     "  Browser",
		Pressed:  rgba.N(rgba.Gray),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
		Click: func(b *button.Button) {
			defer b.Deactivate()

			err := g.ToastInput(
				fmt.Sprintf("%s URL", config.BrowserWindow),
				"https://youtube.com/watch?v=t2kzUcEQa3g",
				fmt.Sprintf("Disable %s auto-hide?", config.BrowserWindow),
				func(url string, option bool) {
					electron.Close()
					video.Close()
					time.Sleep(time.Second) // XXX: Fix concurrency error in device.go Close.

					if device.IsActive() {
						config.Current.VideoCaptureDevice = config.NoVideoCaptureDevice
						populateDevices(true)
					} else {
						populateDevices(false)
					}

					config.Current.BrowserWindowURL = url
					config.Current.DisableBrowserFormatting = option

					err = electron.Open()
					if err != nil {
						g.ToastErrorForce(err)
						return
					}

					config.Current.Window = config.BrowserWindow
				})
			if err != nil {
				notify.Error("Failed to open url dialog (%v)", err)
			}
		},
	}

	closeBrowserButton := &button.Button{
		Text:     "  Browser",
		Pressed:  rgba.N(rgba.PaleRed),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
		Click: func(b *button.Button) {
			g.ToastYesNo("Cancel", fmt.Sprintf("Close %s?", config.BrowserWindow), func() {
				defer b.Deactivate()

				electron.Close()
				config.Current.Window = config.MainDisplay
			}, b.Deactivate)
		},
	}

	header := material.H5(g.cascadia, Title(""))
	header.Color = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	header.Alignment = text.Middle
	header.Font.Weight = text.ExtraBold

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

	for next == "" {
		if !g.open {
			time.Sleep(time.Millisecond * 10)
			continue
		}

		populateWindows(false)
		populateDevices(false)

		e := <-g.Events()
		switch e := e.(type) {
		case app.ConfigEvent:
		case system.DestroyEvent:
			return "", nil
		case system.FrameEvent:
			g.Window.Option(app.Title(Title(fmt.Sprintf("(%s %s)", g.cpu, g.ram))))

			gtx := layout.NewContext(&ops, e)
			pointer.CursorNameOp{Name: pointer.CursorGrab}.Add(gtx.Ops)

			background := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
			paint.ColorOp{Color: rgba.N(rgba.Background)}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			background.Pop()

			split.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{
						Top:  unit.Px(0),
						Left: unit.Px(0),
					}.Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							return layout.NW.Layout(gtx, g.Screen.Layout)
						},
					)
				},
				func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(5)).Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							return fill(gtx, rgba.N(rgba.BackgroundAlt),
								func(gtx layout.Context) layout.Dimensions {
									{
										layout.Inset{
											Left:  unit.Px(10),
											Top:   unit.Px(101),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return saveButton.Layout(gtx)
											},
										)

										layout.Inset{
											Left:  unit.Px(115),
											Top:   unit.Px(101),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return cancelButton.Layout(gtx)
											},
										)

										if config.Current.Window == config.BrowserWindow {
											layout.Inset{
												Top:   unit.Px(5),
												Left:  unit.Px(325),
												Right: unit.Px(10),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													return closeBrowserButton.Layout(gtx)
												},
											)
										} else {
											layout.Inset{
												Top:   unit.Px(5),
												Left:  unit.Px(325),
												Right: unit.Px(10),
											}.Layout(
												gtx,
												func(gtx layout.Context) layout.Dimensions {
													return openBrowserButton.Layout(gtx)
												},
											)
										}

										layout.Inset{
											Top:   unit.Px(37),
											Left:  unit.Px(325),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return defaultButton.Layout(gtx)
											},
										)

										layout.Inset{
											Left:  unit.Px(220),
											Top:   unit.Px(101),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return resetButton.Layout(gtx)
											},
										)

										layout.Inset{
											Left:  unit.Px(325),
											Top:   unit.Px(69),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return openConfigFileButton.Layout(gtx)
											},
										)

										layout.Inset{
											Left:  unit.Px(325),
											Top:   unit.Px(101),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return screenButton.Layout(gtx)
											},
										)
									}

									// Capture video.
									{
										layout.Inset{
											Left: unit.Px(float32(gtx.Constraints.Max.X - 519)),
											Top:  unit.Px(3),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												windowListTitle := material.Label(g.cascadia, unit.Px(14), "Window")
												windowListTitle.Color = rgba.N(rgba.Slate)
												return windowListTitle.Layout(gtx)
											},
										)

										layout.Inset{
											Left:   unit.Px(float32(gtx.Constraints.Max.X - 520)),
											Top:    unit.Px(20),
											Bottom: unit.Px(3),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return widget.Border{
													Color: color.NRGBA{R: 100, G: 100, B: 100, A: 50},
													Width: unit.Px(2),
												}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													return windowList.Layout(gtx, g.normal)
												},
												)
											},
										)
									}

									{
										layout.Inset{
											Left: unit.Px(float32(gtx.Constraints.Max.X - 249)),
											Top:  unit.Px(3),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												deviceListTitle := material.Label(g.cascadia, unit.Px(14), "Video Capture Device")
												deviceListTitle.Color = rgba.N(rgba.Slate)
												return deviceListTitle.Layout(gtx)
											},
										)

										layout.Inset{
											Left:   unit.Px(float32(gtx.Constraints.Max.X - 250)),
											Top:    unit.Px(20),
											Bottom: unit.Px(3),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return widget.Border{
													Color: color.NRGBA{R: 100, G: 100, B: 100, A: 50},
													Width: unit.Px(2),
												}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													return deviceList.Layout(gtx, g.normal)
												},
												)
											},
										)
									}

									// Time area rectangle buttons.
									{
										layout.Inset{
											Top:  unit.Px(5),
											Left: unit.Px(220),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return timeArea.Button.Layout(gtx)
											},
										)
									}

									layout.Inset{
										Top:  unit.Px(37),
										Left: unit.Px(220),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return assetsButton.Layout(gtx)
										},
									)

									layout.Inset{
										Top:  unit.Px(69),
										Left: unit.Px(220),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return captureButton.Layout(gtx)
										},
									)

									// Energy area rectangle buttons.
									{
										layout.Inset{
											Left: unit.Px(115),
											Top:  unit.Px(5),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return energyArea.Button.Layout(gtx)
											},
										)
									}

									// Score area rectangle buttons.
									{
										if device.IsActive() || screen.IsDisplay() {
											scaleText.Color = rgba.N(rgba.Slate)
											scaleValueText.Color = rgba.N(rgba.Slate)
											scaleUpButton.Disabled = true
											scaleDownButton.Disabled = true
										} else {
											scaleText.Color = rgba.N(rgba.White)
											scaleValueText.Color = rgba.N(rgba.White)
											scaleUpButton.Disabled = false
											scaleDownButton.Disabled = false
										}

										layout.Inset{
											Left: unit.Px(10),
											Top:  unit.Px(5),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return scoreArea.Button.Layout(gtx)
											},
										)

										layout.Inset{
											Left: unit.Px(10),
											Top:  unit.Px(38),
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
											Left: unit.Px(45),
											Top:  unit.Px(38),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return scaleText.Layout(gtx)
											},
										)

										layout.Inset{
											Left: unit.Px(45),
											Top:  unit.Px(55),
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
											Left: unit.Px(80),
											Top:  unit.Px(38),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return scaleUpButton.Layout(gtx)
											},
										)
									}

									// Shift N,E,S,W
									{
										if device.IsActive() || screen.IsDisplay() {
											nButton.Disabled = true
											eButton.Disabled = true
											sButton.Disabled = true
											wButton.Disabled = true
											shiftText.Color = rgba.N(rgba.Slate)
										} else {
											nButton.Disabled = false
											eButton.Disabled = false
											sButton.Disabled = false
											wButton.Disabled = false
											shiftText.Color = rgba.N(rgba.White)
										}

										layout.Inset{
											Left: unit.Px(150),
											Top:  unit.Px(37),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return nButton.Layout(gtx)
											},
										)

										layout.Inset{
											Left: unit.Px(184),
											Top:  unit.Px(55),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return eButton.Layout(gtx)
											},
										)

										layout.Inset{
											Left: unit.Px(150),
											Top:  unit.Px(78),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return sButton.Layout(gtx)
											},
										)

										layout.Inset{
											Left: unit.Px(118),
											Top:  unit.Px(55),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return wButton.Layout(gtx)
											},
										)

										layout.Inset{
											Left: unit.Px(150),
											Top:  unit.Px(60),
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

			cursor := pointer.CursorDefault
			for _, area := range []*area.Area{timeArea, energyArea, scoreArea, koArea, objectiveArea} {
				err := area.Layout(gtx, g.Screen.Dims, g.Screen.Image)
				if err != nil {
					g.ToastError(err)
				}
				if area.Focus {
					cursor = pointer.CursorPointer
				}
				if area.Drag {
					cursor = pointer.CursorCrossHair
				}
			}
			g.Window.SetCursorName(cursor)

			e.Frame(gtx.Ops)
		}
	}

	return next, nil
}
