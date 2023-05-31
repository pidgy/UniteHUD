package gui

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
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
	"gioui.org/widget/material"
	"github.com/skratchdot/open-golang/open"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/cursor"
	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/screen"
	"github.com/pidgy/unitehud/gui/visual/spinner"
	"github.com/pidgy/unitehud/gui/visual/split"
	"github.com/pidgy/unitehud/gui/visual/textblock"
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/history"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/nrgba"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/state"
	"github.com/pidgy/unitehud/stats"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/video/device"
)

func (g *GUI) main() (next string, err error) {
	g.Window.Perform(system.ActionCenter)
	g.Window.Perform(system.ActionRaise)

	split := &split.Vertical{
		Ratio: .70,
	}

	configButton := &button.Button{
		Text:        "Configure",
		Released:    nrgba.Gray,
		Pressed:     nrgba.Transparent30,
		BorderWidth: unit.Sp(1.5),
		Click: func(b *button.Button) {
			defer b.Deactivate()
			next = "configure"
		},
	}

	spinRun := spinner.Running()
	defer spinRun.Stop()

	spinStop := spinner.Stopped()
	defer spinStop.Stop()

	spinRecord := spinner.Recording()
	defer spinRecord.Stop()

	recordButton := &button.Button{
		Text:        "Record",
		Disabled:    true,
		BorderWidth: unit.Sp(1.5),
		Click: func(b *button.Button) {
			defer b.Deactivate()
			g.Actions <- Record
		},
	}

	stopButton := &button.Button{
		Text:        "Stop",
		Disabled:    true,
		Released:    nrgba.Disabled,
		BorderWidth: unit.Sp(1.5),
	}

	startButton := &button.Button{
		Text:        "Start",
		Released:    nrgba.Gray,
		Pressed:     nrgba.Transparent30,
		BorderWidth: unit.Sp(1.5),
		Click: func(b *button.Button) {
			g.Preview = false

			configButton.Active = true
			configButton.Disabled = true
			configButton.Released = nrgba.Disabled

			stopButton.Active = false
			stopButton.Disabled = false
			stopButton.Released = nrgba.Gray

			b.Active = false
			b.Disabled = true
			b.Released = nrgba.Disabled

			recordButton.Active = false
			recordButton.Disabled = false
			recordButton.Released = nrgba.Gray

			g.Actions <- Config
			g.Running = true
		},
	}

	logButton := &button.Button{
		Text:        "Logs",
		Disabled:    false,
		Released:    nrgba.Gray,
		Pressed:     nrgba.Transparent30,
		BorderWidth: unit.Sp(1.5),
		Click: func(b *button.Button) {
			defer b.Deactivate()

			g.Actions <- Log
		},
	}

	stopButton.Click = func(b *button.Button) {
		configButton.Active = false
		configButton.Disabled = false
		configButton.Released = nrgba.Gray

		stopButton.Active = false
		stopButton.Disabled = true
		stopButton.Released = nrgba.Disabled

		startButton.Active = false
		startButton.Disabled = false
		startButton.Released = nrgba.Gray

		recordButton.Active = false
		recordButton.Disabled = true
		recordButton.Released = nrgba.Disabled

		g.Actions <- Stop
		g.Running = false
	}

	updateRecordButton := func() {
		if config.Current.Record {
			recordButton.Text = "Recording"
			recordButton.Released = nrgba.DarkRed
		} else {
			recordButton.Text = "Record"
			recordButton.Released = nrgba.Gray
			if recordButton.Disabled {
				recordButton.Released = nrgba.Disabled
			}
		}
	}

	openButton := &button.Button{
		Text:        "Open",
		Released:    nrgba.Gray,
		Pressed:     nrgba.Transparent30,
		BorderWidth: unit.Sp(1.5),
		Click: func(b *button.Button) {
			defer b.Deactivate()

			g.Actions <- Open
		},
	}

	notifyFeedTextBlock, err := textblock.NewCascadiaCodeSemiBold()
	if err != nil {
		notifyFeedTextBlock = &textblock.TextBlock{}
		notify.Error("Failed to load font: (%v)", err)
	}

	statsButton := &button.Button{
		Text:     "¼",
		Released: nrgba.CoolBlue,
		Pressed:  nrgba.Transparent30,
		Size:     image.Pt(30, 16),
		TextSize: unit.Sp(12),

		BorderWidth: unit.Sp(.5),
		Click: func(b *button.Button) {
			defer b.Deactivate()

			stats.Data()

			s, ok := state.Dump()
			if !ok {
				notify.Warn(s)
			} else {
				notify.System(s)
			}
		},
	}

	historyButton := &button.Button{
		Text:     "±",
		Released: nrgba.BloodOrange,
		Pressed:  nrgba.Transparent30,
		Size:     image.Pt(30, 15),
		TextSize: unit.Sp(14),

		BorderWidth: unit.Sp(.5),
		Click: func(b *button.Button) {
			defer b.Deactivate()

			history.Dump()
		},
	}

	obsButton := &button.Button{
		Text:     "obs",
		Released: nrgba.Purple,
		Pressed:  nrgba.Purple,
		Size:     image.Pt(30, 15),
		TextSize: unit.Sp(12),

		BorderWidth: unit.Sp(.5),
		Click: func(b *button.Button) {
			defer b.Deactivate()

			drag := "Drag \"UniteHUD Client\" into any OBS scene."
			if config.Current.Profile == config.ProfileBroadcaster {
				drag = "Drag \"UniteHUD Broadcaster\" into any OBS scene."
			}

			g.ToastOK("UniteHUD Overlay", drag, func() {
				ex, err := os.Executable()
				if err != nil {
					notify.Error("Failed to open www/ directory: %v", err)
					return
				}

				dir := filepath.Dir(ex)
				err = open.Run(dir + "/www")
				if err != nil {
					notify.Error("Failed to open www/ directory: %v", err)
					return
				}
			},
			)
		},
	}

	clearButton := &button.Button{
		Text:     "cls",
		Released: nrgba.PaleRed,
		Pressed:  nrgba.DarkRed,
		Size:     image.Pt(30, 15),
		TextSize: unit.Sp(12),

		BorderWidth: unit.Sp(.5),
		Click: func(b *button.Button) {
			defer b.Deactivate()

			notify.CLS()
			notify.System("Cleared")
		},
	}

	ecoButton := &button.Button{
		Text:     "eco",
		Released: nrgba.ForestGreen,
		Pressed:  nrgba.Transparent30,
		Size:     image.Pt(30, 15),
		TextSize: unit.Sp(12),

		BorderWidth: unit.Sp(.5),
		Active:      !g.ecoMode,
		Click: func(b *button.Button) {
			g.ecoMode = !g.ecoMode
			b.Active = !g.ecoMode

			if g.ecoMode {
				notify.System("Resource saver has been enabled")
			} else {
				notify.System("Resource saver has been disabled")
			}
		},
	}

	controllerButton := &button.Button{
		Text:     "ctrl",
		Released: nrgba.DreamyBlue,
		Pressed:  nrgba.Transparent30,
		Size:     image.Pt(30, 15),
		TextSize: unit.Sp(12),

		BorderWidth: unit.Sp(.5),
		Click: func(b *button.Button) {
			defer b.Deactivate()

			if controller {
				g.ToastError(fmt.Errorf("%s is already open", controllerTitle))
				return
			}

			g.controller()
		},
	}

	previewImage := &button.Image{
		Screen: &screen.Screen{
			Border:      true,
			BorderColor: nrgba.Transparent,
		},
		Click: func(b *button.Image) {
			if !projecting {
				go g.Project()
				projecting = true
			}
		},
	}

	cursor.Is(pointer.CursorDefault)

	var ops op.Ops

	for next == "" {
		if !g.open {
			time.Sleep(time.Millisecond * 10)
			continue
		}

		if config.Current.Crashed != "" {
			g.ToastCrash(fmt.Sprintf("%s recently crashed for the following reason", title.Default), config.Current.Crashed)
			config.Current.Report("")
		}

		if g.ecoMode && state.Since() > time.Minute*30 && !stopButton.Disabled {
			notify.System("Idling for 30 minutes without activity, stopping to save resources")
			stopButton.Click(stopButton)
		}

		e := <-g.Events()
		switch e := e.(type) {
		case app.ConfigEvent:
		case system.DestroyEvent:
			return "", e.Err
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)

			app.Title(g.Title(""))

			cursor.Draw(gtx)

			if g.Bar.Drag {
				system.ActionInputOp(system.ActionMove).Add(gtx.Ops)

			}

			background := clip.Rect{
				Max: gtx.Constraints.Max,
			}.Push(gtx.Ops)
			paint.ColorOp{Color: nrgba.Transparent30.Color()}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			background.Pop()

			g.Bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return split.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return fill(
							gtx,
							nrgba.BackgroundAlt,
							func(gtx layout.Context) layout.Dimensions {
								{
									header := material.H6(g.normal, title.Default)
									header.Color = nrgba.White.Color()
									header.Alignment = text.Middle

									layout.Inset{
										Left: unit.Dp(4),
										Top:  unit.Dp(2),
									}.Layout(gtx, header.Layout)

									profileHeader := material.Caption(g.normal, fmt.Sprintf("%s // %s", strings.Title(config.Current.Profile), strings.Title(config.Current.Platform)))
									profileHeader.Color = nrgba.DreamyPurple.Color()
									profileHeader.Alignment = text.Middle
									profileHeader.Font.Weight = font.Bold

									layout.Inset{
										Left: unit.Dp(4),
										Top:  unit.Dp(27),
									}.Layout(gtx, profileHeader.Layout)

									win := config.Current.Window
									if device.IsActive() {
										win = device.Name(config.Current.VideoCaptureDevice)
									}

									windowHeader := material.Caption(g.normal, win)
									windowHeader.Color = nrgba.DarkSeafoam.Color()
									windowHeader.Alignment = text.Middle
									windowHeader.Font.Weight = font.Bold

									if config.Current.LostWindow != "" {
										windowHeader = material.Caption(g.normal, config.Current.LostWindow)
										windowHeader.Color = nrgba.PaleRed.Color()
									}

									layout.Inset{
										Left: unit.Dp(4),
										Top:  unit.Dp(45),
									}.Layout(gtx, windowHeader.Layout)
								}
								{
									cpu := material.H5(g.normal, g.cpu)
									cpu.Color = nrgba.White.Color()
									cpu.Alignment = text.Middle
									cpu.TextSize = unit.Sp(11)

									layout.Inset{
										Top:  unit.Dp(28),
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 408)),
									}.Layout(gtx, cpu.Layout)

									cpuGraph := material.H5(g.cascadia, stats.CPUData())
									cpuGraph.Color = nrgba.Gray.Color()
									cpuGraph.TextSize = unit.Sp(9)

									layout.Inset{
										Top:  unit.Dp(1),
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 450)),
									}.Layout(gtx, cpuGraph.Layout)

									ram := material.H5(g.normal, g.ram)
									ram.Color = nrgba.White.Color()
									ram.Alignment = text.Middle
									ram.TextSize = unit.Sp(11)

									layout.Inset{
										Top:  unit.Dp(28),
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 248)),
									}.Layout(gtx, ram.Layout)

									ramGraph := material.H5(g.cascadia, stats.RAMData())
									ramGraph.Color = nrgba.Gray.Color()
									ramGraph.TextSize = unit.Sp(9)

									layout.Inset{
										Top:  unit.Dp(1),
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 300)),
									}.Layout(gtx, ramGraph.Layout)

									h := fmt.Sprintf("%d/%2d", team.Energy.Holding, team.Energy.HoldingMax)
									if team.Energy.Holding < 10 {
										h = "0" + h
									}
									holding := material.H5(g.normal, h)
									holding.Color = team.Self.NRGBA.Color()
									holding.Alignment = text.Middle
									holding.TextSize = unit.Sp(13)

									layout.Inset{
										Top:  unit.Dp(50),
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 35)),
									}.Layout(gtx, holding.Layout)
								}
								{
									clients := server.Clients()

									connectedClients := material.H5(g.normal, fmt.Sprintf("OBS %d", clients))
									connectedClients.Color = nrgba.PaleRed.Color()
									if clients > 0 {
										connectedClients.Color = nrgba.Seafoam.Color()
									}
									connectedClients.Alignment = text.Middle
									connectedClients.TextSize = unit.Sp(11)

									layout.Inset{
										Top:  unit.Dp(34),
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 135)),
									}.Layout(gtx, connectedClients.Layout)
								}
								{
									symbol := material.H5(g.normal, spinStop.Next())
									symbol.Alignment = text.Middle
									symbol.TextSize = unit.Sp(14)
									symbol.Font.Weight = font.ExtraBold
									symbol.Color = nrgba.Slate.Color()

									acronym := material.H5(g.normal, "IDLE")
									acronym.Alignment = text.Middle
									acronym.TextSize = unit.Sp(11)
									acronym.Color = nrgba.Slate.Color()

									down := float32(1)
									left := 1

									if config.Current.Record {
										symbol.Text = spinRecord.Next()
										symbol.Color = nrgba.Red.Color()
										symbol.TextSize = unit.Sp(20)
										acronym.Font.Weight = font.ExtraBold

										acronym.Text = "REC"
										acronym.Color = nrgba.Red.Color()

										left = 0
										down = 0
									} else if g.Running {
										symbol.Text = spinRun.Next()
										symbol.Color = nrgba.Green.Color()

										acronym.Text = "RUN"
										acronym.Color = nrgba.Green.Color()

										left = 3
										down = .5
									}

									layout.Inset{
										Top:  unit.Dp(46 + down),
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 145 - left)),
									}.Layout(gtx, symbol.Layout)

									layout.Inset{
										Top:  unit.Dp(50),
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 135)),
									}.Layout(gtx, acronym.Layout)

									if global.DebugMode {
										dbg := material.H5(g.normal, "DBG")
										dbg.Alignment = text.Middle
										dbg.TextSize = unit.Sp(11)
										dbg.Color = nrgba.SeaBlue.Color()

										layout.Inset{
											Top:  unit.Dp(18),
											Left: unit.Dp(float32(gtx.Constraints.Max.X - 135)),
										}.Layout(gtx, dbg.Layout)
									}
								}
								{
									o, p, s := server.Scores()

									purple := material.H5(g.normal, fmt.Sprintf("%d", p))
									purple.Color = team.Purple.NRGBA.Color()
									purple.Alignment = text.Middle
									purple.TextSize = unit.Sp(13)

									layout.Inset{
										Top:  unit.Dp(2),
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 35)),
									}.Layout(gtx, purple.Layout)

									orange := material.H5(g.normal, fmt.Sprintf("%d", o))
									orange.Color = team.Orange.NRGBA.Color()
									orange.Alignment = text.Middle
									orange.TextSize = unit.Sp(13)

									layout.Inset{
										Top:  unit.Dp(18),
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 35)),
									}.Layout(gtx, orange.Layout)

									self := material.H5(g.normal, strconv.Itoa(s))
									self.Color = team.Self.NRGBA.Color()
									self.Alignment = text.Middle
									self.TextSize = unit.Sp(13)

									layout.Inset{
										Top:  unit.Dp(34),
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 35)),
									}.Layout(gtx, self.Layout)
								}
								{
									clock := material.H5(g.normal, server.Clock())
									clock.Color = nrgba.White.Color()
									clock.Alignment = text.Middle
									clock.TextSize = unit.Sp(13)
									layout.Inset{
										Top:  unit.Dp(2),
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 90)),
									}.Layout(gtx, clock.Layout)
								}
								{
									regis := server.Regielekis()

									for i, t := range regis {
										if t == team.None.Name {
											continue
										}

										r := material.H5(g.normal, "E")
										r.Color = nrgba.Regieleki.Color()
										r.Alignment = text.Middle
										r.TextSize = unit.Sp(12)
										layout.Inset{
											Top:  unit.Dp(18),
											Left: unit.Dp(float32(gtx.Constraints.Max.X-90) + float32(i*12)),
										}.Layout(gtx, r.Layout)

										u := material.H5(g.normal, "_")
										u.Color = team.Color(t).Color()
										u.Alignment = text.Middle
										u.TextSize = unit.Sp(18)
										u.Font.Weight = font.Bold
										layout.Inset{
											Top:  unit.Dp(15),
											Left: unit.Dp(float32(gtx.Constraints.Max.X-90) + float32(i*12)),
										}.Layout(gtx, u.Layout)
									}
								}
								{
									regis := server.Bottom()

									for i, t := range regis {
										r := material.H5(g.normal, strings.ToUpper(string(t.Name[4])))
										r.Color = nrgba.Objective(t.Name).Color()
										r.Alignment = text.Middle
										r.TextSize = unit.Sp(12)
										layout.Inset{
											Top:  unit.Dp(34),
											Left: unit.Dp(float32(gtx.Constraints.Max.X-90) + float32(i*12)),
										}.Layout(gtx, r.Layout)

										u := material.H5(g.normal, "_")
										u.Color = team.Color(t.Team).Color()
										u.Alignment = text.Middle
										u.TextSize = unit.Sp(18)
										u.Font.Weight = font.Bold
										layout.Inset{
											Top:  unit.Dp(31),
											Left: unit.Dp(float32(gtx.Constraints.Max.X-90) + float32(i*12)),
										}.Layout(gtx, u.Layout)
									}
								}
								{
									uptime := material.H5(g.normal, g.uptime)
									uptime.Color = nrgba.Slate.Color()
									uptime.Alignment = text.Middle
									uptime.TextSize = unit.Sp(13)

									layout.Inset{
										Top:  unit.Dp(50),
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 90)),
									}.Layout(gtx, uptime.Layout)
								}
								{
									layout.Inset{
										Top: unit.Dp(65),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return notifyFeedTextBlock.Layout(gtx, notify.Feeds())
										},
									)
								}

								return layout.Dimensions{Size: gtx.Constraints.Max}
							},
						)
					},
					func(gtx layout.Context) layout.Dimensions {
						return fill(
							gtx,
							nrgba.BackgroundAlt,
							func(gtx layout.Context) layout.Dimensions {
								{
									updateRecordButton()

									layout.Inset{
										Left: unit.Dp(float32(gtx.Constraints.Max.X - statsButton.Size.X - 2)),
										Top:  unit.Dp(float32(gtx.Constraints.Min.Y + 2)),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return statsButton.Layout(gtx)
										},
									)

									layout.Inset{
										Left: unit.Dp(float32(gtx.Constraints.Max.X - historyButton.Size.X*2 - 2)),
										Top:  unit.Dp(float32(gtx.Constraints.Min.Y + 2)),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return historyButton.Layout(gtx)
										},
									)

									layout.Inset{
										Left: unit.Dp(float32(gtx.Constraints.Max.X - obsButton.Size.X - 2)),
										Top:  unit.Dp(float32(gtx.Constraints.Min.Y + obsButton.Size.Y + 2)),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return obsButton.Layout(gtx)
										},
									)

									layout.Inset{
										Left: unit.Dp(float32(gtx.Constraints.Max.X - clearButton.Size.X*2 - 2)),
										Top:  unit.Dp(float32(gtx.Constraints.Min.Y + clearButton.Size.Y + 2)),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return clearButton.Layout(gtx)
										},
									)

									layout.Inset{
										Left: unit.Dp(float32(gtx.Constraints.Max.X - ecoButton.Size.X - 2)),
										Top:  unit.Dp(float32(gtx.Constraints.Min.Y + ecoButton.Size.Y*2 + 2)),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return ecoButton.Layout(gtx)
										},
									)

									layout.Inset{
										Left: unit.Dp(float32(gtx.Constraints.Max.X - controllerButton.Size.X*2 - 2)),
										Top:  unit.Dp(float32(gtx.Constraints.Min.Y + controllerButton.Size.Y*2 + 2)),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return controllerButton.Layout(gtx)
										},
									)
								}
								// Right-side criteria.
								{
									layout.Inset{
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 150)),
										Top:  unit.Dp(float32(gtx.Constraints.Max.Y - 370)),
									}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										previewImage.SetImage(notify.Preview)

										return layout.UniformInset(unit.Dp(10)).Layout(gtx,
											func(gtx layout.Context) layout.Dimensions {
												return previewImage.Layout(g.cascadia, gtx)
											},
										)
									})

									layout.Inset{
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 125)),
										Top:  unit.Dp(float32(gtx.Constraints.Max.Y - 265)),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return startButton.Layout(gtx)
										},
									)

									layout.Inset{
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 125)),
										Top:  unit.Dp(float32(gtx.Constraints.Max.Y - 210)),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return stopButton.Layout(gtx)
										},
									)

									layout.Inset{
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 125)),
										Top:  unit.Dp(float32(gtx.Constraints.Max.Y - 155)),
									}.Layout(gtx,
										func(gtx layout.Context) layout.Dimensions {
											return configButton.Layout(gtx)
										},
									)

									if recordButton.Disabled {
										layout.Inset{
											Left: unit.Dp(float32(gtx.Constraints.Max.X - 125)),
											Top:  unit.Dp(float32(gtx.Constraints.Max.Y - 100)),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return logButton.Layout(gtx)
											},
										)
									} else {
										layout.Inset{
											Left: unit.Dp(float32(gtx.Constraints.Max.X - 125)),
											Top:  unit.Dp(float32(gtx.Constraints.Max.Y - 100)),
										}.Layout(
											gtx,
											func(gtx layout.Context) layout.Dimensions {
												return recordButton.Layout(gtx)
											},
										)
									}

									layout.Inset{
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 125)),
										Top:  unit.Dp(float32(gtx.Constraints.Max.Y - 45)),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return openButton.Layout(gtx)
										},
									)
								}
								// Event images.
								{
									layout.Inset{
										Top:  unit.Dp(65),
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 150)),
									}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return layout.UniformInset(unit.Dp(5)).Layout(gtx,
											(&screen.Screen{
												Border:      true,
												BorderColor: team.Purple.NRGBA,
												Image:       notify.PurpleScore,
											}).Layout,
										)
									})

									layout.Inset{
										Top:  unit.Dp(127),
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 150)),
									}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return layout.UniformInset(unit.Dp(5)).Layout(gtx,
											(&screen.Screen{
												Border:      true,
												BorderColor: team.Orange.NRGBA,
												Image:       notify.OrangeScore,
											}).Layout,
										)
									})

									layout.Inset{
										Top:  unit.Dp(189),
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 68)),
									}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return layout.UniformInset(unit.Dp(5)).Layout(gtx,
											(&screen.Screen{
												Border:      true,
												BorderColor: team.Energy.NRGBA,
												Image:       notify.Energy,
											}).Layout,
										)
									})

									layout.Inset{
										Top:  unit.Dp(251),
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 68)),
									}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return layout.UniformInset(unit.Dp(5)).Layout(gtx,
											(&screen.Screen{
												Border:      true,
												BorderColor: team.Time.NRGBA,
												Image:       notify.Time,
											}).Layout,
										)
									})
								}

								return layout.Dimensions{Size: gtx.Constraints.Max}
							},
						)
					},
				)
			})

			e.Frame(gtx.Ops)
		}
	}

	return next, nil
}
