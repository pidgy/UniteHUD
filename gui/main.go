package gui

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	"gioui.org/widget/material"
	"github.com/skratchdot/open-golang/open"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/screen"
	"github.com/pidgy/unitehud/gui/visual/spinner"
	"github.com/pidgy/unitehud/gui/visual/split"
	"github.com/pidgy/unitehud/gui/visual/textblock"
	"github.com/pidgy/unitehud/history"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/state"
	"github.com/pidgy/unitehud/stats"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/update"
	"github.com/pidgy/unitehud/video/device"
)

func (g *GUI) main() (next string, err error) {
	g.Window.Raise()

	split := &split.Vertical{
		Ratio: .70,
	}

	configButton := &button.Button{
		Text:     " Configure",
		Released: rgba.N(rgba.Gray),
		Pressed:  rgba.N(rgba.DarkGray),
	}
	configButton.Click = func() {
		configButton.Active = false
		next = "configure"
	}

	spinRun := spinner.Running()
	defer spinRun.Stop()

	spinStop := spinner.Stopped()
	defer spinStop.Stop()

	spinRecord := spinner.Recording()
	defer spinRecord.Stop()

	updateButton := &button.Button{
		Text:     "\tUpdate",
		Released: rgba.N(rgba.Gray),
		Pressed:  rgba.N(rgba.DarkGray),
	}
	updateButton.Click = func() {
		defer updateButton.Deactivate()

		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", "https://unitehud.dev").Start()
		if err != nil {
			notify.Error("Failed to open https://UniteHUD.dev (%v)", err)
			return
		}
	}

	startButton := &button.Button{
		Text:     "\t  Start",
		Released: rgba.N(rgba.Gray),
		Pressed:  rgba.N(rgba.DarkGray),
	}

	stopButton := &button.Button{
		Text:     "\t  Stop",
		Disabled: true,
		Released: rgba.N(rgba.Disabled),
	}

	recordButton := &button.Button{
		Text:     "\tRecord",
		Disabled: true,
	}

	startButton.Click = func() {
		device.Hide = false

		g.Preview = false

		configButton.Active = true
		configButton.Disabled = true
		configButton.Released = rgba.N(rgba.Disabled)

		stopButton.Active = false
		stopButton.Disabled = false
		stopButton.Released = rgba.N(rgba.Gray)

		startButton.Active = false
		startButton.Disabled = true
		startButton.Released = rgba.N(rgba.Disabled)

		recordButton.Active = false
		recordButton.Disabled = false
		recordButton.Released = rgba.N(rgba.Gray)

		g.Actions <- Config
		g.Running = true
	}

	stopButton.Click = func() {
		device.Hide = true

		configButton.Active = false
		configButton.Disabled = false
		configButton.Released = rgba.N(rgba.Gray)

		stopButton.Active = false
		stopButton.Disabled = true
		stopButton.Released = rgba.N(rgba.Disabled)

		startButton.Active = false
		startButton.Disabled = false
		startButton.Released = rgba.N(rgba.Gray)

		recordButton.Active = false
		recordButton.Disabled = true
		recordButton.Released = rgba.N(rgba.Disabled)

		g.Actions <- Stop
		g.Running = false
	}

	updateRecordButton := func() {
		if config.Current.Record {
			recordButton.Text = " Recording"
			recordButton.Released = rgba.N(rgba.DarkRed)
		} else {
			recordButton.Text = "\tRecord"
			recordButton.Released = rgba.N(rgba.Gray)
			if recordButton.Disabled {
				recordButton.Released = rgba.N(rgba.Disabled)
			}
		}
	}

	recordButton.Click = func() {
		defer recordButton.Deactivate()

		g.Actions <- Record
	}

	openButton := &button.Button{
		Text:     "\t  Open",
		Released: rgba.N(rgba.Gray),
		Pressed:  rgba.N(rgba.DarkGray),
	}
	openButton.Click = func() {
		defer openButton.Deactivate()

		g.Actions <- Open
	}

	notifyFeedTextBlock, err := textblock.NewCascadiaCodeSemiBold()
	if err != nil {
		notifyFeedTextBlock = &textblock.TextBlock{}
		notify.Error("Failed to load font: (%v)", err)
	}

	statsButton := &button.Button{
		Text:           "¼",
		Released:       rgba.N(rgba.CoolBlue),
		Pressed:        rgba.N(rgba.DarkGray),
		Size:           image.Pt(30, 16),
		TextSize:       unit.Sp(12),
		TextOffsetTop:  -4,
		TextOffsetLeft: 0,
		BorderWidth:    unit.Sp(.5),
	}
	statsButton.Click = func() {
		defer statsButton.Deactivate()

		stats.Data()

		s, ok := state.Dump()
		if !ok {
			notify.Warn(s)
		} else {
			notify.System(s)
		}
	}

	historyButton := &button.Button{
		Text:           "±",
		Released:       rgba.N(rgba.BloodOrange),
		Pressed:        rgba.N(rgba.DarkGray),
		Size:           image.Pt(30, 15),
		TextSize:       unit.Sp(14),
		TextOffsetTop:  -6,
		TextOffsetLeft: 0,
		BorderWidth:    unit.Sp(.5),
	}
	historyButton.Click = func() {
		defer historyButton.Deactivate()

		history.Dump()
	}

	obsButton := &button.Button{
		Text:           "obs",
		Released:       rgba.N(rgba.Purple),
		Pressed:        rgba.N(rgba.Purple),
		Size:           image.Pt(30, 15),
		TextSize:       unit.Sp(12),
		TextOffsetTop:  -5,
		TextOffsetLeft: -5,
		BorderWidth:    unit.Sp(.5),
	}
	obsButton.Click = func() {
		defer obsButton.Deactivate()

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
	}

	clearButton := &button.Button{
		Text:           "cls",
		Released:       rgba.N(rgba.PaleRed),
		Pressed:        rgba.N(rgba.DarkRed),
		Size:           image.Pt(30, 15),
		TextSize:       unit.Sp(12),
		TextOffsetTop:  -4,
		TextOffsetLeft: -4,
		BorderWidth:    unit.Sp(.5),
	}
	clearButton.Click = func() {
		defer clearButton.Deactivate()

		notify.CLS()
		notify.System("Cleared")
	}

	ecoButton := &button.Button{
		Text:           "eco",
		Released:       rgba.N(rgba.ForestGreen),
		Pressed:        rgba.N(rgba.DarkGray),
		Size:           image.Pt(30, 15),
		TextSize:       unit.Sp(12),
		TextOffsetTop:  -4,
		TextOffsetLeft: -6,
		BorderWidth:    unit.Sp(.5),
		Active:         !g.ecoMode,
	}
	ecoButton.Click = func() {
		g.ecoMode = !g.ecoMode
		ecoButton.Active = !g.ecoMode

		if g.ecoMode {
			notify.System("Resource saver has been enabled")
		} else {
			notify.System("Resource saver has been disabled")
		}
	}

	controllerButton := &button.Button{
		Text:           "ctrl",
		Released:       rgba.N(rgba.DreamyBlue),
		Pressed:        rgba.N(rgba.DarkGray),
		Size:           image.Pt(30, 15),
		TextSize:       unit.Sp(12),
		TextOffsetTop:  -4,
		TextOffsetLeft: -5,
		BorderWidth:    unit.Sp(.5),
	}
	controllerButton.Click = func() {
		defer controllerButton.Deactivate()

		if controller {
			g.ToastError(fmt.Errorf("%s is already open", controllerTitle))
			return
		}

		g.controller()
	}

	preview := &button.Image{
		Screen: &screen.Screen{
			Border:      true,
			BorderColor: rgba.N(rgba.Gray),
		},
	}
	preview.Click = func() {
		preview.Hide = !preview.Hide
	}

	var ops op.Ops

	for next == "" {
		if !g.open {
			time.Sleep(time.Millisecond * 10)
			continue
		}

		if config.Current.Crashed != "" {
			g.ToastCrash(fmt.Sprintf("%s recently crashed for the following reason", Title("")), config.Current.Crashed)
			config.Current.Report("")
		}

		if g.ecoMode && state.Since() > time.Minute*30 && !stopButton.Disabled {
			notify.System("Idling for 30 minutes without activity, stopping to save resources")
			stopButton.Click()
		}

		e := <-g.Events()
		switch e := e.(type) {
		case app.ConfigEvent:
		case system.DestroyEvent:
			return "", e.Err
		case system.FrameEvent:
			g.Window.Option(app.Title(Title("")))

			gtx := layout.NewContext(&ops, e)
			pointer.CursorNameOp{Name: pointer.CursorGrab}.Add(gtx.Ops)

			background := clip.Rect{
				Max: gtx.Constraints.Max,
			}.Push(gtx.Ops)
			paint.ColorOp{Color: rgba.N(rgba.Gray)}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			background.Pop()

			split.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return fill(
						gtx,
						color.NRGBA{R: 25, G: 25, B: 25, A: 255},
						func(gtx layout.Context) layout.Dimensions {
							{
								header := material.H6(g.normal, Title(""))
								header.Color = rgba.N(rgba.White)
								header.Alignment = text.Middle

								layout.Inset{
									Left: unit.Px(2),
									Top:  unit.Px(2),
								}.Layout(gtx, header.Layout)

								profileHeader := material.Caption(g.normal, fmt.Sprintf("%s Mode", strings.Title(config.Current.Profile)))
								profileHeader.Color = rgba.N(rgba.DreamyPurple)
								profileHeader.Alignment = text.Middle
								profileHeader.Font.Weight = text.Bold

								layout.Inset{
									Left: unit.Px(2),
									Top:  unit.Px(27),
								}.Layout(gtx, profileHeader.Layout)

								win := config.Current.Window
								if config.Current.VideoCaptureDevice != config.NoVideoCaptureDevice {
									win = device.Name(config.Current.VideoCaptureDevice)
								}

								windowHeader := material.Caption(g.normal, win)
								windowHeader.Color = rgba.N(rgba.DarkSeafoam)
								windowHeader.Alignment = text.Middle
								windowHeader.Font.Weight = text.Bold

								if config.Current.LostWindow != "" {
									windowHeader = material.Caption(g.normal, config.Current.LostWindow)
									windowHeader.Color = rgba.N(rgba.PaleRed)
								}

								layout.Inset{
									Left: unit.Px(2),
									Top:  unit.Px(45),
								}.Layout(gtx, windowHeader.Layout)
							}
							{
								cpu := material.H5(g.normal, g.cpu)
								cpu.Color = rgba.N(rgba.White)
								cpu.Alignment = text.Middle
								cpu.TextSize = unit.Sp(11)

								layout.Inset{
									Top:  unit.Px(28),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 408)),
								}.Layout(gtx, cpu.Layout)

								cpuGraph := material.H5(g.cascadia, stats.CPUData())
								cpuGraph.Color = rgba.N(rgba.Gray)
								cpuGraph.TextSize = unit.Sp(9)

								layout.Inset{
									Top:  unit.Px(1),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 450)),
								}.Layout(gtx, cpuGraph.Layout)

								ram := material.H5(g.normal, g.ram)
								ram.Color = rgba.N(rgba.White)
								ram.Alignment = text.Middle
								ram.TextSize = unit.Sp(11)

								layout.Inset{
									Top:  unit.Px(28),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 248)),
								}.Layout(gtx, ram.Layout)

								ramGraph := material.H5(g.cascadia, stats.RAMData())
								ramGraph.Color = rgba.N(rgba.Gray)
								ramGraph.TextSize = unit.Sp(9)

								layout.Inset{
									Top:  unit.Px(1),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 300)),
								}.Layout(gtx, ramGraph.Layout)

								uptime := material.H5(g.normal, g.uptime)
								uptime.Color = rgba.N(rgba.Slate)
								uptime.Alignment = text.Middle
								uptime.TextSize = unit.Sp(12)

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 90)),
									Top:  unit.Px(34),
								}.Layout(gtx, uptime.Layout)

								h := fmt.Sprintf("%d/%2d", team.Energy.Holding, team.Energy.HoldingMax)
								if team.Energy.Holding < 10 {
									h = "0" + h
								}
								holding := material.H5(g.normal, h)
								holding.Color = rgba.N(team.Self.RGBA)
								holding.Alignment = text.Middle
								holding.TextSize = unit.Sp(13)

								layout.Inset{
									Top:  unit.Px(50),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 35)),
								}.Layout(gtx, holding.Layout)

								symbol := material.H5(g.normal, spinStop.Next())
								symbol.Alignment = text.Middle
								symbol.TextSize = unit.Sp(14)
								symbol.Font.Weight = text.ExtraBold
								symbol.Color = rgba.N(rgba.Slate)

								acronym := material.H5(g.normal, "STP")
								acronym.Alignment = text.Middle
								acronym.TextSize = unit.Sp(11)
								acronym.Color = rgba.N(rgba.Slate)

								down := float32(1)
								left := 1

								if config.Current.Record {
									symbol.Text = spinRecord.Next()
									symbol.Color = rgba.N(rgba.Red)
									symbol.TextSize = unit.Sp(20)
									acronym.Font.Weight = text.ExtraBold

									acronym.Text = "REC"
									acronym.Color = rgba.N(rgba.Red)

									left = 0
									down = 0
								} else if g.Running {
									symbol.Text = spinRun.Next()
									symbol.Color = rgba.N(rgba.Green)

									acronym.Text = "RUN"
									acronym.Color = rgba.N(rgba.Green)

									left = 3
									down = .5
								}

								layout.Inset{
									Top:  unit.Px(46 + down),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 145 - left)),
								}.Layout(gtx, symbol.Layout)

								layout.Inset{
									Top:  unit.Px(50),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 135)),
								}.Layout(gtx, acronym.Layout)

								if global.DebugMode {
									dbg := material.H5(g.normal, "DBG")
									dbg.Alignment = text.Middle
									dbg.TextSize = unit.Sp(11)
									dbg.Color = rgba.N(rgba.SeaBlue)

									layout.Inset{
										Top:  unit.Px(34),
										Left: unit.Px(float32(gtx.Constraints.Max.X - 135)),
									}.Layout(gtx, dbg.Layout)
								}
							}
							{
								o, p, s := server.Scores()

								purple := material.H5(g.normal, fmt.Sprintf("%d", p))
								purple.Color = rgba.N(team.Purple.RGBA)
								purple.Alignment = text.Middle
								purple.TextSize = unit.Sp(13)

								layout.Inset{
									Top:  unit.Px(2),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 35)),
								}.Layout(gtx, purple.Layout)

								orange := material.H5(g.normal, fmt.Sprintf("%d", o))
								orange.Color = rgba.N(team.Orange.RGBA)
								orange.Alignment = text.Middle
								orange.TextSize = unit.Sp(13)

								layout.Inset{
									Top:  unit.Px(18),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 35)),
								}.Layout(gtx, orange.Layout)

								self := material.H5(g.normal, strconv.Itoa(s))
								self.Color = rgba.N(team.Self.RGBA)
								self.Alignment = text.Middle
								self.TextSize = unit.Sp(13)

								layout.Inset{
									Top:  unit.Px(34),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 35)),
								}.Layout(gtx, self.Layout)

								clock := material.H5(g.normal, server.Clock())
								clock.Color = rgba.N(rgba.White)
								clock.Alignment = text.Middle
								clock.TextSize = unit.Sp(13)
								layout.Inset{
									Top:  unit.Px(2),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 90)),
								}.Layout(gtx, clock.Layout)
							}
							{
								clients := server.Clients()

								connectedClients := material.H5(g.normal, fmt.Sprintf("OBS %d", clients))
								connectedClients.Color = rgba.N(rgba.PaleRed)
								if clients > 0 {
									connectedClients.Color = rgba.N(rgba.Seafoam)
								}
								connectedClients.Alignment = text.Middle
								connectedClients.TextSize = unit.Sp(11)

								layout.Inset{
									Top:  unit.Px(50),
									Left: unit.Px(float32(gtx.Constraints.Max.X - 90)),
								}.Layout(gtx, connectedClients.Layout)

							}
							{
								objs := server.Regielekis()

								for i, t := range objs {
									objective := material.H5(g.normal, "R")

									objective.Color = rgba.N(team.Color(t))
									objective.Alignment = text.Middle
									objective.TextSize = unit.Sp(12)

									layout.Inset{
										Top:  unit.Px(18),
										Left: unit.Px(float32(gtx.Constraints.Max.X-90) + float32(i*12)),
									}.Layout(gtx, objective.Layout)
								}
							}

							{
								layout.Inset{
									Top: unit.Px(65),
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
						color.NRGBA{R: 25, G: 25, B: 25, A: 255},
						func(gtx layout.Context) layout.Dimensions {
							{
								updateRecordButton()

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - statsButton.Size.X - 2)),
									Top:  unit.Px(float32(gtx.Constraints.Min.Y + 2)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return statsButton.Layout(gtx)
									},
								)

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - historyButton.Size.X*2 - 2)),
									Top:  unit.Px(float32(gtx.Constraints.Min.Y + 2)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return historyButton.Layout(gtx)
									},
								)

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - obsButton.Size.X - 2)),
									Top:  unit.Px(float32(gtx.Constraints.Min.Y + obsButton.Size.Y + 2)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return obsButton.Layout(gtx)
									},
								)

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - clearButton.Size.X*2 - 2)),
									Top:  unit.Px(float32(gtx.Constraints.Min.Y + clearButton.Size.Y + 2)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return clearButton.Layout(gtx)
									},
								)

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - ecoButton.Size.X - 2)),
									Top:  unit.Px(float32(gtx.Constraints.Min.Y + ecoButton.Size.Y*2 + 2)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return ecoButton.Layout(gtx)
									},
								)

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - controllerButton.Size.X*2 - 2)),
									Top:  unit.Px(float32(gtx.Constraints.Min.Y + controllerButton.Size.Y*2 + 2)),
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
									Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
									Top:  unit.Px(float32(gtx.Constraints.Max.Y - 335)),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									// Assigned from captureWindow.
									preview.SetImage(notify.Preview)

									return preview.Layout(g.cascadia, gtx)
								},
								)

								if update.Available {
									layout.Inset{
										Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
										Top:  unit.Px(float32(gtx.Constraints.Max.Y - 390)),
									}.Layout(
										gtx,
										func(gtx layout.Context) layout.Dimensions {
											return updateButton.Layout(gtx)
										},
									)
								}

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
									Top:  unit.Px(float32(gtx.Constraints.Max.Y - 265)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return startButton.Layout(gtx)
									},
								)

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
									Top:  unit.Px(float32(gtx.Constraints.Max.Y - 210)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return stopButton.Layout(gtx)
									},
								)

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
									Top:  unit.Px(float32(gtx.Constraints.Max.Y - 155)),
								}.Layout(gtx,
									func(gtx layout.Context) layout.Dimensions {
										return configButton.Layout(gtx)
									},
								)

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
									Top:  unit.Px(float32(gtx.Constraints.Max.Y - 100)),
								}.Layout(
									gtx,
									func(gtx layout.Context) layout.Dimensions {
										return recordButton.Layout(gtx)
									},
								)

								layout.Inset{
									Left: unit.Px(float32(gtx.Constraints.Max.X - 125)),
									Top:  unit.Px(float32(gtx.Constraints.Max.Y - 45)),
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
									Top:  unit.Px(65),
									Left: unit.Px(5),
								}.Layout(gtx, (&screen.Screen{
									Border:      true,
									BorderColor: rgba.N(team.Purple.RGBA),
									Image:       notify.PurpleScore,
								}).Layout)

								layout.Inset{
									Top:  unit.Px(120),
									Left: unit.Px(5),
								}.Layout(gtx, (&screen.Screen{
									Border:      true,
									BorderColor: rgba.N(team.Orange.RGBA),
									Image:       notify.OrangeScore,
								}).Layout)

								layout.Inset{
									Top:  unit.Px(175),
									Left: unit.Px(5),
								}.Layout(gtx, (&screen.Screen{
									Border:      true,
									BorderColor: rgba.N(team.Self.RGBA),
									Image:       notify.Energy,
									ScaleX:      2,
									ScaleY:      2,
								}).Layout)

								layout.Inset{
									Top:  unit.Px(232),
									Left: unit.Px(5),
								}.Layout(gtx, (&screen.Screen{
									Border:      true,
									BorderColor: rgba.N(team.Self.RGBA),
									Image:       notify.SelfScore,
									ScaleX:      4,
									ScaleY:      4,
								}).Layout)

								layout.Inset{
									Top:  unit.Px(175),
									Left: unit.Px(68),
								}.Layout(gtx, (&screen.Screen{
									Image:       notify.Time,
									Border:      true,
									BorderColor: rgba.N(team.Time.RGBA),
									ScaleX:      2,
									ScaleY:      2,
								}).Layout)
							}

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
