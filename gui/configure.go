package gui

import (
	"fmt"
	"image"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/font"
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
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/nrgba"
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
		Ratio:      .6,
		Adjustable: true,
	}

	koArea := &area.Area{
		Text:     "KO",
		TextSize: unit.Sp(13),
		Theme:    g.normal,
		Min:      config.Current.KOs.Min,
		Max:      config.Current.KOs.Max,
		NRGBA:    area.Locked,
		Match:    g.matchKOs,
		Cooldown: time.Millisecond * 1500,

		Capture: &area.Capture{
			Option: "KO",
			File:   "ko_area.png",
			Base:   config.Current.KOs,
		},
	}

	objectiveArea := &area.Area{
		Text:     "Objectives",
		TextSize: unit.Sp(13),
		Theme:    g.normal,
		Min:      config.Current.Objectives.Min,
		Max:      config.Current.Objectives.Max,
		NRGBA:    area.Locked,
		Match:    g.matchObjectives,
		Cooldown: time.Second,

		Capture: &area.Capture{
			Option: "Objective",
			File:   "objective_area.png",
			Base:   config.Current.Objectives,
		},
	}

	energyArea := &area.Area{
		Text:     "Aeos",
		TextSize: unit.Sp(13),
		Theme:    g.normal,
		Min:      config.Current.Energy.Min,
		Max:      config.Current.Energy.Max,
		NRGBA:    area.Locked,
		Match:    g.matchEnergy,
		Cooldown: team.Energy.Delay,

		Capture: &area.Capture{
			Option: "Aeos",
			File:   "aeos_area.png",
			Base:   config.Current.Energy,
		},
	}

	timeArea := &area.Area{
		Text:     "Time",
		TextSize: unit.Sp(12),
		Theme:    g.normal,
		Min:      config.Current.Time.Min,
		Max:      config.Current.Time.Max,
		NRGBA:    area.Locked,
		Match:    g.matchTime,
		Cooldown: team.Time.Delay,

		Capture: &area.Capture{
			Option: "Time",
			File:   "time_area.png",
			Base:   config.Current.Time,
		},
	}

	scoreArea := &area.Area{
		Text:          "Score",
		TextAlignLeft: true,
		Theme:         g.normal,
		Min:           config.Current.Scores.Min,
		Max:           config.Current.Scores.Max,
		NRGBA:         area.Locked,
		Match:         g.matchScore,
		Cooldown:      team.Purple.Delay,

		Capture: &area.Capture{
			Option: "Score",
			File:   "score_area.png",
			Base:   config.Current.Scores,
		},
	}

	stateArea := &area.Area{
		Hidden: true,

		Text:     "State",
		Theme:    g.normal,
		NRGBA:    area.Locked,
		Cooldown: team.Game.Delay,
		Match:    g.matchState,
		Min:      image.Pt(0, 0),
		Max:      image.Pt(150, 20),

		Capture: &area.Capture{
			Option: "State",
			File:   "state_area.png",
			Base:   StateArea(),
		},
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
		Active:      true,
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
		Active:      true,
		Text:        "Preview",
		Pressed:     nrgba.Gray,
		Released:    nrgba.Transparent30,
		Size:        image.Pt(100, 30),
		SingleClick: true,
		BorderWidth: unit.Sp(1.5),
		Click: func(b *button.Button) {
			g.previewCaptures([]*area.Capture{
				{Option: "Screen", File: "screen_area.png", Base: g.Screen.Bounds()},
				scoreArea.Capture,
				energyArea.Capture,
				timeArea.Capture,
				objectiveArea.Capture,
				koArea.Capture,
				stateArea.Capture,
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
		Text:        "Default",
		Pressed:     nrgba.Gray,
		Released:    nrgba.Transparent30,
		Active:      true,
		Size:        image.Pt(100, 30),
		BorderWidth: unit.Sp(1.5),
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
		Text:        "Edit",
		Pressed:     nrgba.Gray,
		Released:    nrgba.Transparent30,
		Active:      true,
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

			reapplyAreas()
		},
	}

	saveButton := &button.Button{
		Text:     "Save",
		Pressed:  nrgba.ForestGreen,
		Released: nrgba.Transparent30,
		Active:   true,
		Size:     image.Pt(100, 30),
	}

	// Hold on to the previous configuration to overwrite memory saves.
	prev := config.Current

	cancelButton := &button.Button{
		Text:        "Cancel",
		Pressed:     nrgba.BloodOrange.Alpha(0x5F),
		Released:    nrgba.Transparent30,
		Active:      true,
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

					if config.Current.Window == config.BrowserWindow {
						err = electron.Open()
						if err != nil {
							notify.Error("Failed to open %s (%v)", config.BrowserWindow, err)
						}
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

	var populateDevices, populateScreens, populateWindows func(bool)

	screenList := &dropdown.List{
		WidthModifier: 6,
		Items:         []*dropdown.Item{},
		Callback: func(i *dropdown.Item, _ *dropdown.List) {
			defer defaultButton.Click(defaultButton)

			defer populateScreens(true)
			defer populateWindows(true)
			defer populateDevices(true)

			device.Close()
			electron.Close()

			config.Current.VideoCaptureDevice = config.NoVideoCaptureDevice

			config.Current.Window = i.Text
			if config.Current.Window == "" {
				config.Current.Window = config.MainDisplay
				return
			}
		},
	}

	prevLen := 0
	populateScreens = func(videoCaptureDisabledEvent bool) {
		if videoCaptureDisabledEvent {
			for _, item := range screenList.Items {
				item.Checked.Value = false
				if item.Text == config.Current.Window && !device.IsActive() {
					item.Checked.Value = true
				}
			}
		}

		screens := video.Screens()
		if len(screens) == prevLen && !videoCaptureDisabledEvent {
			return
		}
		prevLen = len(screens)

		items := []*dropdown.Item{}

		if videoCaptureDisabledEvent && config.Current.Window == "" {
			config.Current.Window = config.MainDisplay
		}

		for _, screen := range screens {
			items = append(items,
				&dropdown.Item{
					Text:    screen,
					Checked: widget.Bool{Value: screen == config.Current.Window && !device.IsActive()},
				},
			)
		}

		screenList.Items = items
	}

	windowList := &dropdown.List{
		WidthModifier: 2,
		Items:         []*dropdown.Item{},
		Callback: func(i *dropdown.Item, _ *dropdown.List) {
			defer defaultButton.Click(defaultButton)

			defer populateWindows(true)
			defer populateScreens(true)
			defer populateDevices(true)

			device.Close()
			electron.Close()

			config.Current.VideoCaptureDevice = config.NoVideoCaptureDevice

			config.Current.Window = i.Text
			if config.Current.Window == "" {
				config.Current.Window = config.MainDisplay
				return
			}
		},
	}

	populateWindows = func(videoCaptureDisabledEvent bool) {
		if videoCaptureDisabledEvent && config.Current.Window == "" {
			config.Current.Window = config.MainDisplay
		}

		for _, item := range windowList.Items {
			item.Checked.Value = config.Current.Window == item.Text && config.Current.VideoCaptureDevice == config.NoVideoCaptureDevice
		}

		items := []*dropdown.Item{}

		windows := video.Windows()
		if len(windows) == len(windowList.Items) && !videoCaptureDisabledEvent {
			if len(windowList.Items) == 0 {
				return
			}

			if windowList.Items[0].Checked.Value {
				return
			}

			for _, item := range windowList.Items {
				if item.Checked.Value {
					items = append([]*dropdown.Item{item}, items...)
				} else {
					items = append(items, item)
				}
			}
		} else {
			for _, win := range windows {
				item := &dropdown.Item{
					Text:    win,
					Checked: widget.Bool{Value: win == config.Current.Window},
				}
				if item.Checked.Value {
					items = append([]*dropdown.Item{item}, items...)
				} else {
					items = append(items, item)
				}
			}
		}

		windowList.Items = items
	}

	deviceList := &dropdown.List{}

	populateDevices = func(videoCaptureDisabledEvent bool) {
		devices := video.Devices()

		// Set the "Disabled" checkbox when device is not active.
		if len(devices)+1 == len(deviceList.Items) && !videoCaptureDisabledEvent {
			deviceList.Items[0].Checked.Value = !device.IsActive()

			for _, item := range deviceList.Items {
				item.Checked.Value = false
				if config.Current.VideoCaptureDevice == item.Value {
					item.Checked.Value = true
				}
			}

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
		Callback: func(i *dropdown.Item, _ *dropdown.List) {
			defer defaultButton.Click(defaultButton)

			electron.Close()
			video.Close()
			// Can this be Disabled? Fixes concurrency error in device.go Close.
			// time.Sleep(time.Second)

			config.Current.VideoCaptureDevice = i.Value

			if i.Text == "Disabled" {
				i.Checked = widget.Bool{Value: true}
			}

			populateDevices(i.Text == "Disabled")
			populateWindows(true)
			populateScreens(true)

			go func() {
				err = video.Open()
				if err != nil {
					g.ToastErrorForce(err)

					config.Current.Window = config.MainDisplay
					config.Current.VideoCaptureDevice = config.NoVideoCaptureDevice

					populateWindows(true)
					populateDevices(true)
					populateScreens(true)

					return
				}

				config.Current.LostWindow = ""
			}()
		},
	}

	platformList := &dropdown.List{
		WidthModifier: 8,
		Items: []*dropdown.Item{
			{
				Text:    strings.Title(config.PlatformSwitch),
				Checked: widget.Bool{Value: config.Current.Platform == config.PlatformSwitch},
			},
			{
				Text:    strings.Title(config.PlatformMobile),
				Checked: widget.Bool{Value: config.Current.Platform == config.PlatformMobile},
			},
			{
				Text:    strings.Title(config.PlatformBluestacks),
				Checked: widget.Bool{Value: config.Current.Platform == config.PlatformBluestacks},
			},
		},
		Callback: func(i *dropdown.Item, l *dropdown.List) {
			for _, item := range l.Items {
				if item != i {
					item.Checked.Value = false
					continue
				}
				item.Checked.Value = true

				config.Current.Platform = strings.ToLower(item.Text)

				err := config.Current.Save()
				if err != nil {
					notify.Error("Failed to load %s profile configuration", config.Current.Profile)
					return
				}

				err = config.Load(config.Current.Profile)
				if err != nil {
					notify.Error("Failed to load %s profile configuration", config.Current.Profile)
					return
				}

				time.AfterFunc(time.Second, func() {
					err := config.Current.Save()
					if err != nil {
						notify.Error("Failed to save %s profile configuration", config.Current.Profile)
						return
					}
				})
			}
		},
	}

	profileList := &dropdown.List{
		WidthModifier: platformList.WidthModifier,
		Radio:         true,
		Items: []*dropdown.Item{
			{
				Text: strings.Title(config.ProfilePlayer),
				Checked: widget.Bool{
					Value: config.Current.Profile == config.ProfilePlayer,
				},
			},
			{
				Text: strings.Title(config.ProfileBroadcaster),
				Checked: widget.Bool{
					Value: config.Current.Profile == config.ProfileBroadcaster,
				},
			},
		},
		Callback: func(i *dropdown.Item, _ *dropdown.List) {
			if config.Current.Profile == strings.ToLower(i.Text) {
				return
			}

			electron.Close()

			config.Current.Profile = strings.ToLower(i.Text)

			err := config.Load(config.Current.Profile)
			if err != nil {
				notify.Error("Failed to load %s profile configuration", config.Current.Profile)
				return
			}

			if config.Current.Window == config.BrowserWindow {
				err = electron.Open()
				if err != nil {
					notify.Error("Failed to open %s (%v)", config.BrowserWindow, err)
				}
			}

			populateWindows(true)
			populateDevices(true)
			populateScreens(true)

			notify.System("Profile set to %s mode", i.Text)

			time.AfterFunc(time.Second, func() {
				err := config.Current.Save()
				if err != nil {
					notify.Error("Failed to save %s profile configuration", config.Current.Profile)
					return
				}
			})
		},
	}

	resetButton := &button.Button{
		Text:        "Reset",
		Active:      true,
		Pressed:     nrgba.DarkRed,
		Released:    nrgba.Transparent30,
		BorderWidth: unit.Sp(1.5),
		Size:        image.Pt(100, 30),
		Click: func(b *button.Button) {
			g.ToastYesNo("Reset", fmt.Sprintf("Reset UniteHUD %s configuration?", config.Current.Profile), func() {
				defer b.Deactivate()
				defer server.Clear()

				deviceList.Callback(deviceList.Items[0], deviceList)

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
				populateScreens(true)

				g.Actions <- Refresh

				next = "main"

				notify.Announce("Reset UniteHUD %s configuration", config.Current.Profile)
			}, b.Deactivate)
		},
	}
	openBrowserButton := &button.Button{
		Text:        "Browser",
		Active:      true,
		Pressed:     nrgba.Gray,
		Released:    nrgba.Transparent30,
		BorderWidth: unit.Sp(1.5),
		Size:        image.Pt(100, 30),
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
		Text:     "Browser",
		Active:   true,
		Pressed:  nrgba.PaleRed,
		Released: nrgba.Transparent30,
		Size:     image.Pt(100, 30),
		Click: func(b *button.Button) {
			g.ToastYesNo("Cancel", fmt.Sprintf("Close %s?", config.BrowserWindow), func() {
				defer b.Deactivate()

				electron.Close()
				config.Current.Window = config.MainDisplay
			}, b.Deactivate)
		},
	}

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
			populateWindows(false)
			populateDevices(false)
			populateScreens(false)
			populateTicks = 0
		}

		e := <-g.Events()
		switch e := e.(type) {
		case app.ConfigEvent:
		case system.DestroyEvent:
			return "", nil
		case system.FrameEvent:
			app.Title(g.Title(fmt.Sprintf("(%s %s)", g.cpu, g.ram)))

			gtx := layout.NewContext(&ops, e)

			background := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
			paint.ColorOp{Color: nrgba.Background.Color()}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			background.Pop()

			g.Bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return split.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return g.Screen.Layout(gtx)
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

											if config.Current.Window == config.BrowserWindow {
												layout.Inset{
													Top:   unit.Dp(5),
													Left:  unit.Dp(325),
													Right: unit.Dp(10),
												}.Layout(
													gtx,
													func(gtx layout.Context) layout.Dimensions {
														return closeBrowserButton.Layout(gtx)
													},
												)
											} else {
												layout.Inset{
													Top:  unit.Dp(5),
													Left: unit.Dp(220),
												}.Layout(
													gtx,
													func(gtx layout.Context) layout.Dimensions {
														return openBrowserButton.Layout(gtx)
													},
												)
											}

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
														return platformList.Layout(gtx, g.normal)
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
														return profileList.Layout(gtx, g.normal)
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
														return screenList.Layout(gtx, g.normal)
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
														return windowList.Layout(gtx, g.normal)
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
														return deviceList.Layout(gtx, g.normal)
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
														return timeArea.Button.Layout(gtx)
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
											if device.IsActive() || screen.IsDisplay() {
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
			cursor := pointer.CursorDefault
			for _, area := range []*area.Area{timeArea, energyArea, scoreArea, koArea, objectiveArea, stateArea} {
				err := area.Layout(gtx, g.Screen.Dims, g.Screen.Image)
				if err != nil {
					g.ToastError(err)
				}
				if area.Focus {
					cursor = pointer.CursorPointer
				}
				if area.Drag {
					cursor = pointer.CursorCrosshair
				}
			}
			_ = cursor
			// g.Window.SetCursorName(cursor)

			e.Frame(gtx.Ops)
		}
	}

	return next, nil
}

func StateArea() image.Rectangle {
	b := Window.Screen.Bounds()
	return image.Rect(b.Max.X/3, 0, b.Max.X-b.Max.X/3, b.Max.Y)
}
