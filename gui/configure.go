package gui

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"os/exec"
	"syscall"
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
	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/gui/visual/area"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/dropdown"
	"github.com/pidgy/unitehud/gui/visual/help"
	"github.com/pidgy/unitehud/gui/visual/split"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/video"
	"github.com/pidgy/unitehud/video/device"
)

func (g *GUI) configure() (next string, err error) {
	defer server.SetConfig(false)

	g.Preview = true
	device.Hide = false
	defer func() {
		g.Preview = false
		device.Hide = true
	}()

	split := &split.Horizontal{
		Ratio: .6,
	}

	energyAreaText := "Energy"
	energyAreaButtonText := fmt.Sprintf("\t%s", energyAreaText)
	objectiveAreaText := "Objectives"
	koAreaText := "KOs"
	timeAreaText := "\tTime"
	timeAreaButtonText := fmt.Sprintf("  %s", timeAreaText)
	scoreAreaText := "Score"
	scoreAreaButtonText := fmt.Sprintf("\t %s", scoreAreaText)
	lockedAreaText := func(a string) string { return fmt.Sprintf("%s (Locked)", a) }
	lockedAreaButtonText := "\tLocked"

	koArea := &area.Area{
		Text:     koAreaText,
		TextSize: unit.Sp(13),
		Min:      config.Current.KOs.Min.Div(2),
		Max:      config.Current.KOs.Max.Div(2),

		Button: &button.Button{
			Active: true,
		},
	}

	objectiveArea := &area.Area{
		Text:     objectiveAreaText,
		TextSize: unit.Sp(13),
		Min:      config.Current.Objectives.Min.Div(2),
		Max:      config.Current.Objectives.Max.Div(2),

		Button: &button.Button{
			Active: true,
		},
	}

	energyArea := &area.Area{
		Text:     energyAreaText,
		TextSize: unit.Sp(13),
		Min:      config.Current.Energy.Min.Div(2),
		Max:      config.Current.Energy.Max.Div(2),

		Button: &button.Button{
			Active:   true,
			Text:     energyAreaButtonText,
			Pressed:  rgba.N(rgba.Night),
			Released: rgba.N(rgba.DarkGray),
			Size:     image.Pt(100, 30),
		},
	}

	energyArea.Button.Click = func() {
		if !energyArea.Button.Active {
			energyArea.Text = lockedAreaText(energyAreaText)
			energyArea.Button.Text = lockedAreaButtonText
			energyArea.NRGBA.A = 0x9
			return
		}

		energyArea.Text = energyAreaText
		energyArea.Button.Text = energyAreaButtonText
		energyArea.NRGBA.A = 0x4F
	}

	timeArea := &area.Area{
		Text:     timeAreaText,
		TextSize: unit.Sp(13),
		Min:      config.Current.Time.Min.Div(2),
		Max:      config.Current.Time.Max.Div(2),
		Button: &button.Button{
			Active:   true,
			Text:     timeAreaButtonText,
			Pressed:  rgba.N(rgba.Night),
			Released: rgba.N(rgba.DarkGray),
			Size:     image.Pt(100, 30),
		},
	}

	timeArea.Button.Click = func() {
		if !timeArea.Button.Active {
			timeArea.Text = lockedAreaText(timeAreaText)
			timeArea.Button.Text = lockedAreaButtonText
			timeArea.NRGBA.A = 0x9
			return
		}

		timeArea.Text = timeAreaText
		timeArea.Button.Text = timeAreaButtonText
		timeArea.NRGBA.A = 0x4F
	}

	scoreArea := &area.Area{
		Text:          scoreAreaText,
		TextAlignLeft: true,
		Min:           config.Current.Scores.Min.Div(2),
		Max:           config.Current.Scores.Max.Div(2),
		Theme:         g.normal,

		Button: &button.Button{
			Active:   true,
			Text:     scoreAreaButtonText,
			Pressed:  rgba.N(rgba.Night),
			Released: rgba.N(rgba.DarkGray),
			Size:     image.Pt(100, 30),
		},
	}

	scoreArea.Button.Click = func() {
		if !scoreArea.Button.Active {
			scoreArea.Text = lockedAreaText(scoreAreaText)
			scoreArea.Button.Text = lockedAreaButtonText
			scoreArea.NRGBA.A = 0x9
			return
		}

		scoreArea.Text = scoreAreaText
		scoreArea.Button.Text = scoreAreaButtonText
		scoreArea.NRGBA.A = 0x4F
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
	}

	scaleDownButton := &button.Button{
		Text:     "-",
		Pressed:  color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released: color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:     image.Pt(30, 30),
		TextSize: unit.Sp(12),
	}

	scaleUpButton.Click = func() {
		g.Preview = false
		defer g.buttonSpam(scaleUpButton)

		scaleUpButton.Active = !scaleUpButton.Active
		config.Current.Scale += .01
	}

	scaleDownButton.Click = func() {
		g.Preview = false
		defer g.buttonSpam(scaleDownButton)

		scaleDownButton.Active = !scaleDownButton.Active
		config.Current.Scale -= .01
		if config.Current.Scale < 1 {
			config.Current.Scale = 1
		}
	}

	nButton := &button.Button{
		Text:        "^",
		Pressed:     color.NRGBA{R: 100, G: 100, B: 100, A: 0x4F},
		Released:    color.NRGBA{R: 50, G: 50, B: 0xFF, A: 0x3F},
		Size:        image.Pt(30, 20),
		TextSize:    unit.Sp(18),
		SingleClick: true,
		Click: func() {
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
		Click: func() {
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
		Click: func() {
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
		Click: func() {
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
		Click: func() {
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
		Click: func() {
			dir, err := os.Getwd()
			if err != nil {
				g.ToastOK("Error", "Failed to find current directory")
				return
			}

			for _, fr := range []struct {
				area string
				file string
				rect image.Rectangle
			}{
				{"entire area", "screen_area.png", g.Screen.Bounds()},
				{"Score area", "score_area.png", scoreArea.Rectangle()},
				{"Energy area", "balls_area.png", energyArea.Rectangle()},
				{"Time area", "time_area.png", timeArea.Rectangle()},
				{"Objective area", "objective_area.png", objectiveArea.Rectangle()},
				{"KO area", "ko_area.png", koArea.Rectangle()},
			} {
				noq := make(chan bool)
				g.toastActive = false
				g.ToastYesNo("Capture", fmt.Sprintf("Capture %s?", fr.area), func() { noq <- false }, func() { noq <- true })

				if <-noq {
					continue
				}

				img, err := video.CaptureRect(fr.rect)
				if err != nil {
					g.ToastOK("Error", fmt.Sprintf("Failed to capture %s", fr.file))
					return
				}

				matrix, err := gocv.ImageToMatRGB(img)
				if err != nil {
					g.ToastOK("Error", fmt.Sprintf("Failed to create %s", fr.file))
					return
				}
				defer matrix.Close()

				if !gocv.IMWrite(fr.file, matrix) {
					g.ToastOK("Error", fmt.Sprintf("Failed to save %s", fr.file))
					return
				}

				var sI syscall.StartupInfo
				var pI syscall.ProcessInformation
				argv := syscall.StringToUTF16Ptr(os.Getenv("windir") + "\\system32\\cmd.exe /C " +
					fmt.Sprintf("\"%s\\%s\"", dir, fr.file))

				err = syscall.CreateProcess(nil, argv, nil, nil, true, 0, nil, nil, &sI, &pI)
				if err != nil {
					g.ToastOK("Error", fmt.Sprintf("Failed to open %s", fr.file))
					return
				}
			}
		},
	}

	mapArea := &area.Area{
		Text:     "\t  Map",
		TextSize: unit.Sp(13),
		Min:      config.Current.Map.Min.Div(2),
		Max:      config.Current.Map.Max.Div(2),

		Button: &button.Button{
			Active:   true,
			Text:     "\t  tMap",
			Pressed:  rgba.N(rgba.Gray),
			Released: rgba.N(rgba.DarkGray),
			Size:     image.Pt(100, 30),
		},
	}

	resizeButton := &button.Button{
		Text:     "\tResize",
		Pressed:  rgba.N(rgba.Gray),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
	}

	resizeButton.Click = func() {
		resizeButton.Active = !resizeButton.Active

		err := video.Resize16x9()
		if err != nil {
			resizeButton.Error()
			notify.Error("%v", err)
			return
		}
	}

	defaultButton := &button.Button{
		Text:     "\tDefault",
		Pressed:  rgba.N(rgba.Gray),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
	}

	reapplyAreas := func() {
		energyArea.Min = config.Current.Energy.Min.Div(2)
		energyArea.Max = config.Current.Energy.Max.Div(2)
		scoreArea.Min = config.Current.Scores.Min.Div(2)
		scoreArea.Max = config.Current.Scores.Max.Div(2)
		mapArea.Min = config.Current.Map.Min.Div(2)
		mapArea.Max = config.Current.Map.Max.Div(2)
		timeArea.Min = config.Current.Time.Min.Div(2)
		timeArea.Max = config.Current.Time.Max.Div(2)
		objectiveArea.Min = config.Current.Objectives.Min.Div(2)
		objectiveArea.Max = config.Current.Objectives.Max.Div(2)
		koArea.Min = config.Current.KOs.Min.Div(2)
		koArea.Max = config.Current.KOs.Max.Div(2)
	}

	defaultButton.Click = func() {
		defaultButton.Active = !defaultButton.Active

		config.Current.SetDefaultAreas()

		reapplyAreas()
	}

	openConfigFileButton := &button.Button{
		Text:     "\t   Edit",
		Pressed:  rgba.N(rgba.Gray),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
	}

	openConfigFileButton.Click = func() {
		openConfigFileButton.Active = !openConfigFileButton.Active

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
	}

	saveButton := &button.Button{
		Text:     "\t  Save",
		Pressed:  rgba.N(rgba.ForestGreen),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
	}

	cancelButton := &button.Button{
		Text:     "\tCancel",
		Pressed:  rgba.N(rgba.Alpha(rgba.BloodOrange, 0x5F)),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
	}

	// Hold on to the previous configuration to overwrite memory saves.
	prev := config.Current

	cancelButton.Click = func() {
		g.ToastYesNo("Cancel", "Discard configuration changes?",
			func() {
				cancelButton.Active = !cancelButton.Active

				server.Clear()

				cancelButton.Disabled = true
				saveButton.Disabled = true
				energyArea.Button.Disabled = true
				timeArea.Button.Disabled = true
				scoreArea.Button.Disabled = true

				config.Current = prev
				err := config.Current.Save()
				if err != nil {
					notify.Error("Failed to save UniteHUD configuration (%v)")
				}

				notify.System("Configuration omitted")

				g.Actions <- Refresh

				next = "main"
			}, func() {
				cancelButton.Active = !cancelButton.Active
			},
		)
	}

	saveButton.Click = func() {
		g.ToastYesNo("Save", "Save configuration changes?",
			func() {
				saveButton.Active = !saveButton.Active

				server.Clear()

				cancelButton.Disabled = true
				saveButton.Disabled = true
				energyArea.Button.Disabled = true
				timeArea.Button.Disabled = true
				scoreArea.Button.Disabled = true

				config.Current.Scores = scoreArea.Rectangle()
				config.Current.Time = timeArea.Rectangle()
				config.Current.Energy = energyArea.Rectangle()
				config.Current.Map = mapArea.Rectangle()
				config.Current.Objectives = objectiveArea.Rectangle()
				config.Current.KOs = koArea.Rectangle()

				err := config.Current.Save()
				if err != nil {
					notify.Error("Failed to save UniteHUD configuration (%v)")
				}

				notify.System("Configuration saved to " + config.Current.File())

				g.Actions <- Refresh

				next = "main"
			}, func() {
				saveButton.Active = !saveButton.Active
			},
		)
	}

	screenButton := &button.Button{
		Text:     "\tPreview",
		Pressed:  rgba.N(rgba.Gray),
		Released: rgba.N(rgba.DarkGray),
		Active:   true,
		Size:     image.Pt(100, 30),
	}

	screenButton.Click = func() {
		g.Preview = !g.Preview
	}

	windowList := &dropdown.List{
		Items: []*dropdown.Item{
			{
				Text:     config.MainDisplay,
				Disabled: config.Current.VideoCaptureDevice != config.NoVideoCaptureDevice,
				Checked:  widget.Bool{Value: config.Current.Window == config.MainDisplay},
			},
		},
		Callback: func(i *dropdown.Item) {
			if config.Current.VideoCaptureDevice != config.NoVideoCaptureDevice {
				return
			}

			if i.Text == "" {
				config.Current.Window = config.MainDisplay
			} else {
				config.Current.Window = i.Text
			}
		},
		WidthModifier: 2,
	}

	populateWindows := func(videoCaptureDisabledEvent bool) {
		if config.Current.Window == config.MainDisplay {
			windowList.Items[0].Checked.Value = true
			for _, item := range windowList.Items {
				if item.Text != config.MainDisplay {
					item.Checked.Value = false
				}
			}
		} else {
			windowList.Items[0].Checked.Value = false
			for _, item := range windowList.Items {
				if item.Text != config.MainDisplay {
					item.Checked.Value = config.Current.Window == item.Text
				}
			}
		}

		windows, _ := video.Sources()
		if len(windows) == len(windowList.Items) && !videoCaptureDisabledEvent {
			return
		}

		windowList.Items = windowList.Items[:0]

		if videoCaptureDisabledEvent && config.Current.Window == "" {
			config.Current.Window = config.MainDisplay
		}

		for _, win := range windows {
			windowList.Items = append(windowList.Items,
				&dropdown.Item{
					Text:     win,
					Disabled: config.Current.VideoCaptureDevice != config.NoVideoCaptureDevice,
					Checked:  widget.Bool{Value: win == config.Current.Window},
				},
			)
		}

		windowList.Items[0].Disabled = config.Current.VideoCaptureDevice != config.NoVideoCaptureDevice
	}

	populateWindows(false)

	for _, i := range windowList.Items {
		if i.Text == config.Current.Window {
			i.Checked.Value = true
		}
	}

	deviceList := &dropdown.List{
		WidthModifier: 3,
	}

	populateDevices := func(videoCaptureDisabledEvent bool) {
		_, devices := video.Sources()

		if len(devices)+1 == len(deviceList.Items) && !videoCaptureDisabledEvent {
			deviceList.Items[0].Checked.Value = config.Current.VideoCaptureDevice == config.NoVideoCaptureDevice
			return
		}

		deviceList.Items = []*dropdown.Item{
			{
				Text:  "Disabled",
				Value: config.NoVideoCaptureDevice,
				Checked: widget.Bool{
					Value: config.Current.VideoCaptureDevice != config.NoVideoCaptureDevice,
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
					Value: config.Current.VideoCaptureDevice != config.NoVideoCaptureDevice,
				},
			},
		},
		Callback: func(i *dropdown.Item) {
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

			log.Debug().Int("device", i.Value).Msg("selected video capture device")

			go func() {
				err = video.Load()
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
	}

	resetButton.Click = func() {
		g.ToastYesNo("Reset", fmt.Sprintf("Reset %s configuration?", config.Current.Profile), func() {
			resetButton.Active = !resetButton.Active

			deviceList.Callback(deviceList.Items[0])

			err := config.Current.Reset()
			if err != nil {
				notify.Error("Failed to reset %s configuration (%v)", config.Current.Profile, err)
			}

			config.Current.Reload()

			energyArea.Min, energyArea.Max = config.Current.Energy.Min.Div(2), config.Current.Energy.Max.Div(2)
			timeArea.Min, timeArea.Max = config.Current.Time.Min.Div(2), config.Current.Time.Max.Div(2)
			scoreArea.Min, scoreArea.Max = config.Current.Scores.Min.Div(2), config.Current.Scores.Max.Div(2)
			objectiveArea.Min, objectiveArea.Max = config.Current.Objectives.Min.Div(2), config.Current.Objectives.Max.Div(2)
			koArea.Min, koArea.Max = config.Current.KOs.Min.Div(2), config.Current.KOs.Max.Div(2)

			populateWindows(true)
			populateDevices(true)

			g.Actions <- Refresh

			next = "main"

			notify.Announce("Reset %s configuration", config.Current.Profile)
		}, func() {
			resetButton.Active = !resetButton.Active
		},
		)
	}

	header := material.H5(g.cascadia, Title(""))
	header.Color = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	header.Alignment = text.Middle
	header.Font.Weight = text.ExtraBold

	pauseMatchingRoutines := false
	defer func() { pauseMatchingRoutines = true }()

	go g.while(func() { g.matchKOs(koArea) }, &pauseMatchingRoutines)
	go g.while(func() { g.matchObjectives(objectiveArea) }, &pauseMatchingRoutines)
	go g.while(func() { g.matchTime(timeArea) }, &pauseMatchingRoutines)

	if config.Current.Profile == config.ProfilePlayer {
		go g.while(func() { g.matchEnergy(energyArea) }, &pauseMatchingRoutines)
		go g.while(func() { g.matchScore(scoreArea) }, &pauseMatchingRoutines)
	}
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
			paint.ColorOp{Color: rgba.N(rgba.Gray)}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			background.Pop()

			split.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(5)).Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							if !screenButton.Active {
								return layout.Dimensions{Size: gtx.Constraints.Max}
							}

							return fill(
								gtx,
								color.NRGBA{R: 25, G: 25, B: 25, A: 255},
								g.Screen.Layout)
						},
					)
				},

				func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(5)).Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							return fill(gtx, color.NRGBA{R: 25, G: 25, B: 25, A: 255},
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

										layout.Inset{
											Top:   unit.Px(5),
											Left:  unit.Px(325),
											Right: unit.Px(10),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return resizeButton.Layout(gtx)
											},
										)

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
										if config.Current.VideoCaptureDevice != config.NoVideoCaptureDevice || config.Current.Window == config.MainDisplay {
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
												if config.Current.VideoCaptureDevice == config.NoVideoCaptureDevice {
													scaleValueText.Text = fmt.Sprintf("%.2fx", config.Current.Scale)
												} else {
													scaleValueText.Text = "1.00x"
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
										if config.Current.VideoCaptureDevice != config.NoVideoCaptureDevice || config.Current.Window == config.MainDisplay || config.Current.Scale == 1 {
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

			timeArea.Layout(gtx)

			// mapArea.Layout(gtx)
			switch config.Current.Profile {
			case config.ProfilePlayer:
				energyArea.Layout(gtx)
				scoreArea.Layout(gtx)
			case config.ProfileBroadcaster:
			}

			koArea.Layout(gtx)
			objectiveArea.Layout(gtx)

			e.Frame(gtx.Ops)
		}
	}

	return next, nil
}

func (g *GUI) configurationHelpDialog(h *help.Help, widget layout.Widget) (next string, err error) {
	split := &split.Vertical{Ratio: .70}

	var ops op.Ops

	header := material.H5(g.cascadia, "Help: Configuration")
	header.Color = rgba.N(rgba.White)
	header.Alignment = text.Middle

	backwardButton := &button.Button{
		Text:     " <",
		Released: rgba.N(rgba.Slate),
		Pressed:  rgba.N(rgba.DarkGray),
		Size:     image.Pt(40, 35),
	}

	backwardButton.Click = func() {
		backwardButton.Active = !backwardButton.Active
		if h.Page != 0 {
			h.Page--
		}
	}

	forwardButton := &button.Button{
		Text:     " >",
		Released: rgba.N(rgba.Slate),
		Pressed:  rgba.N(rgba.DarkGray),
		Size:     image.Pt(40, 35),
	}

	forwardButton.Click = func() {
		forwardButton.Active = !forwardButton.Active
		if h.Page != h.Pages-1 {
			h.Page++
		}
	}

	returnButton := &button.Button{
		Text:     "\t  Back",
		Released: rgba.N(rgba.Slate),
		Pressed:  rgba.N(rgba.DarkGray),
	}

	returnButton.Click = func() {
		returnButton.Active = false
		next = "configure"
	}

	for next == "" {
		if !g.open {
			time.Sleep(time.Millisecond * 10)
			continue
		}

		e := <-g.Events()
		switch e := e.(type) {
		case app.ConfigEvent:
		case system.DestroyEvent:
			return "", e.Err
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)
			pointer.CursorNameOp{Name: pointer.CursorGrab}.Add(gtx.Ops)

			background := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
			paint.ColorOp{Color: rgba.N(rgba.Gray)}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			background.Pop()

			split.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return fill(gtx,
						color.NRGBA{R: 25, G: 25, B: 25, A: 255},
						func(gtx layout.Context) layout.Dimensions {
							layout.Inset{
								Left: unit.Px(2),
								Top:  unit.Px(10),
							}.Layout(gtx, header.Layout)

							return layout.Inset{Top: unit.Px(50)}.Layout(gtx,
								func(gtx layout.Context) layout.Dimensions {
									return widget(gtx)
								},
							)
						},
					)
				},

				func(gtx layout.Context) layout.Dimensions {
					return fill(
						gtx,
						color.NRGBA{R: 25, G: 25, B: 25, A: 255},
						func(gtx layout.Context) layout.Dimensions {
							pages := material.H5(g.cascadia, fmt.Sprintf("%d / %d", h.Page+1, h.Pages))
							pages.Color = rgba.N(rgba.White)
							pages.Alignment = text.Middle
							pages.TextSize = unit.Sp(14)
							layout.Inset{
								Left: unit.Px(float32(gtx.Constraints.Max.X - 90)),
								Top:  unit.Px(float32(gtx.Constraints.Max.Y - 130)),
							}.Layout(gtx, pages.Layout)

							layout.Inset{
								Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
								Top:  unit.Px(float32(gtx.Constraints.Max.Y - 100)),
							}.Layout(gtx,
								func(gtx layout.Context) layout.Dimensions {
									return backwardButton.Layout(gtx)
								},
							)

							layout.Inset{
								Left: unit.Px(float32(gtx.Constraints.Max.X - 65)),
								Top:  unit.Px(float32(gtx.Constraints.Max.Y - 100)),
							}.Layout(gtx,
								func(gtx layout.Context) layout.Dimensions {
									return forwardButton.Layout(gtx)
								},
							)

							layout.Inset{
								Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
								Top:  unit.Px(float32(gtx.Constraints.Max.Y - 45)),
							}.Layout(gtx,
								func(gtx layout.Context) layout.Dimensions {
									return returnButton.Layout(gtx)
								},
							)

							return layout.Dimensions{Size: gtx.Constraints.Max}
						},
					)
				},
			)

			e.Frame(gtx.Ops)
		}
	}

	return next, nil
}
