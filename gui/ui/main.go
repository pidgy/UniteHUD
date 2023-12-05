//go:build !lite

package ui

import (
	"fmt"
	"image"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/global"
	"github.com/pidgy/unitehud/core/history"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/nrgba"
	"github.com/pidgy/unitehud/core/server"
	"github.com/pidgy/unitehud/core/state"
	"github.com/pidgy/unitehud/core/stats"
	"github.com/pidgy/unitehud/core/team"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/decorate"
	"github.com/pidgy/unitehud/gui/visual/screen"
	"github.com/pidgy/unitehud/gui/visual/spinner"
	"github.com/pidgy/unitehud/gui/visual/split"
	"github.com/pidgy/unitehud/gui/visual/textblock"
	"github.com/pidgy/unitehud/media/audio"
	"github.com/pidgy/unitehud/media/video/device"
	"github.com/pidgy/unitehud/media/video/monitor"
	"github.com/pidgy/unitehud/media/video/window"
	"github.com/pidgy/unitehud/system/desktop"
	"github.com/pidgy/unitehud/system/desktop/clicked"
	"github.com/pidgy/unitehud/system/discord"
	"github.com/pidgy/unitehud/system/save"
	"github.com/skratchdot/open-golang/open"
)

var (
	once = true
)

type main struct {
	split struct {
		vertical *split.Vertical
	}

	labels struct {
		audio,
		discord,
		warning,
		window,
		cpu, cpuGraph,
		ram, ramGraph,
		holding,
		connectedClients,
		symbol,
		acronym,
		fps,
		purpleScore,
		orangeScore,
		selfScore,
		clock,
		uptime,
		version material.LabelStyle

		regielekis, regielekiUnderlines,
		regiBottoms, regiBottomUnderlines []material.LabelStyle
	}

	buttons struct {
		start,
		stop *button.Widget
		projector *button.ImageWidget
	}

	textblocks struct {
		feed *textblock.Widget
	}

	screens struct {
		purple,
		orange,
		aeos,
		time *screen.Widget
	}

	spinners struct {
		run  *spinner.Widget
		stop *spinner.Widget
	}

	menu struct {
		settings,
		client,
		stats,
		results,
		obs,
		clear,
		eco,
		logs,
		record,
		file,
		startstop,
		hideTop,
		hideRight *button.Widget
	}
}

func (g *GUI) main() {
	if once {
		once = false
		g.window.Option(
			app.Title(global.Title),
			app.Size(
				unit.Dp(g.dimensions.min.X),
				unit.Dp(g.dimensions.min.Y),
			),
			app.MinSize(
				unit.Dp(g.dimensions.min.X),
				unit.Dp(g.dimensions.min.Y),
			),
			app.MaxSize(
				unit.Dp(g.dimensions.max.X),
				unit.Dp(g.dimensions.max.Y),
			),
		)
	}

	ui := g.mainUI()

	defer ui.spinners.run.Stop()
	defer ui.spinners.stop.Stop()

	defer g.header.Remove(g.header.Add(ui.menu.startstop))
	defer g.header.Remove(g.header.Add(ui.menu.settings))
	defer g.header.Remove(g.header.Add(ui.menu.hideRight))
	defer g.header.Remove(g.header.Add(ui.menu.hideTop))
	defer g.header.Remove(g.header.Add(ui.menu.client))
	defer g.header.Remove(g.header.Add(ui.menu.stats))
	defer g.header.Remove(g.header.Add(ui.menu.results))
	defer g.header.Remove(g.header.Add(ui.menu.obs))
	defer g.header.Remove(g.header.Add(ui.menu.clear))
	defer g.header.Remove(g.header.Add(ui.menu.eco))
	defer g.header.Remove(g.header.Add(ui.menu.logs))
	defer g.header.Remove(g.header.Add(ui.menu.record))
	defer g.header.Remove(g.header.Add(ui.menu.file))

	g.window.Perform(system.ActionRaise)
	if !g.firstOpen {
		g.firstOpen = true
		g.window.Perform(system.ActionCenter)
		g.window.Perform(system.ActionUnmaximize)
	}

	g.header.Open()

	var ops op.Ops

	for is.Now == is.MainMenu {
		if !g.open {
			time.Sleep(time.Millisecond * 10)
			continue
		}

		if config.Current.Crashed != "" {
			g.ToastCrash(
				fmt.Sprintf("Previous Crash: %s", config.Current.Crashed),
				func() {
					config.Current.Report("")

					err := config.Current.Save()
					if err != nil {
						notify.Error("Failed to save configuration (%v)", err)
					}
				},
				func() { save.OpenLogDirectory() },
			)
		}

		if g.performance.eco && state.Idle() > time.Minute*30 && !ui.buttons.stop.Disabled {
			desktop.Notification("Eco Mode").
				Says("No matches detected for 30 minutes, stopping to save resources").
				When(clicked.OpenUniteHUD).
				Send()

			ui.buttons.stop.Click(ui.buttons.stop)
		}

		switch event := g.window.NextEvent().(type) {
		case app.ConfigEvent:
		case system.DestroyEvent:
			g.next(is.Closing)
			return
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, event)
			op.InvalidateOp{
				At: gtx.Now,
			}.Add(gtx.Ops)

			g.dimensions.size = event.Size

			decorate.Background(gtx)
			decorate.Label(&ui.labels.cpu, g.performance.cpu)
			decorate.Label(&ui.labels.cpuGraph, stats.CPUGraph())
			decorate.Label(&ui.labels.ram, g.performance.ram)
			decorate.Label(&ui.labels.ramGraph, stats.RAMGraph())
			decorate.Label(&ui.labels.holding, ui.labels.holding.Text)
			decorate.ForegroundAlt(&ui.labels.cpuGraph.Color)
			decorate.ForegroundAlt(&ui.labels.ramGraph.Color)

			g.header.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return ui.split.vertical.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
						if ui.menu.hideTop.Text == "⇊" {
							return ui.textblocks.feed.Layout(gtx, notify.Feeds())
						}

						warnings, nonwarnings := []string{}, []string{}
						switch {
						case config.Current.Advanced.IncreasedCaptureRate > 0:
							warnings = append(warnings, fmt.Sprintf("Match Frequency: %d%%",
								100+config.Current.Advanced.IncreasedCaptureRate))
						case config.Current.Advanced.IncreasedCaptureRate < 0:
							nonwarnings = append(warnings, fmt.Sprintf("Match Frequency: %d%%",
								100+config.Current.Advanced.IncreasedCaptureRate))
						}

						if len(warnings) > 0 {
							layout.Inset{
								Left: unit.Dp(2),
								Top:  unit.Dp(18),
							}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								ui.labels.warning.Text = fmt.Sprintf("⚠ CPU (%s)", strings.Join(warnings, ","))
								return ui.labels.warning.Layout(gtx)
							})
						}

						if len(nonwarnings) > 0 {
							ui.labels.warning.Text = fmt.Sprintf("⚠ CPU (%s)", strings.Join(nonwarnings, ","))
							ui.labels.warning.Color = nrgba.PastelGreen.Color()

							layout.Inset{
								Left: unit.Dp(2),
								Top:  unit.Dp(35),
							}.Layout(gtx, ui.labels.warning.Layout)
						}

						layout.Inset{
							Left: unit.Dp(1),
							Top:  unit.Dp(.4),
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							if config.Current.Advanced.Discord.Disabled {
								ui.labels.discord.Color.A = 127
								ui.labels.discord.Text = "👾 Discord RPC Disabled"
							} else {
								ui.labels.discord.Color.A = 200
								ui.labels.discord.Text = fmt.Sprintf("👾 %s: %s", strings.ReplaceAll(discord.Activity.Details, "UniteHUD - ", ""), discord.Activity.State)
							}
							return ui.labels.discord.Layout(gtx)
						})

						layout.Inset{
							Left: unit.Dp(2),
							Top:  unit.Dp(19),
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							ui.labels.audio.Text = audio.Label()
							return ui.labels.audio.Layout(gtx)
						})

						switch {
						case device.IsActive():
							if ui.labels.window.Text == "" || ui.labels.window.Text == config.Current.Video.Capture.Window.Name {
								ui.labels.window.Text = fmt.Sprintf("🕹️ %s", device.Name(config.Current.Video.Capture.Device.Index))
							}
						case window.IsOpen(), monitor.IsDisplay():
							ui.labels.window.Text = fmt.Sprintf("🕹️ %s", config.Current.Video.Capture.Window.Name)
						}
						if config.Current.Video.Capture.Window.Lost != "" {
							ui.labels.window.Text = config.Current.Video.Capture.Window.Lost
							ui.labels.window.Text = fmt.Sprintf("🕹️ %s", config.Current.Video.Capture.Window.Name)
							ui.labels.window.Color = nrgba.PaleRed.Color()
						}
						layout.Inset{
							Left: unit.Dp(2),
							Top:  unit.Dp(49),
						}.Layout(gtx, ui.labels.window.Layout)

						layout.Inset{
							Top:  unit.Dp(28),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 408)),
						}.Layout(gtx, ui.labels.cpu.Layout)

						layout.Inset{
							Top:  unit.Dp(1),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 450)),
						}.Layout(gtx, ui.labels.cpuGraph.Layout)

						layout.Inset{
							Top:  unit.Dp(28),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 248)),
						}.Layout(gtx, ui.labels.ram.Layout)

						layout.Inset{
							Top:  unit.Dp(1),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 300)),
						}.Layout(gtx, ui.labels.ramGraph.Layout)

						ui.labels.holding.Text = fmt.Sprintf("%02d/%02d", team.Energy.Holding, team.Energy.HoldingMax)

						layout.Inset{
							Top:  unit.Dp(50),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 35)),
						}.Layout(gtx, ui.labels.holding.Layout)

						clients := server.Clients()
						if clients > 0 {
							ui.labels.connectedClients.Color = nrgba.Seafoam.Color()
						} else {
							ui.labels.connectedClients.Color = nrgba.PaleRed.Color()
						}

						ui.labels.connectedClients.Text = fmt.Sprintf("OBS %d", clients)
						layout.Inset{
							Top:  unit.Dp(34),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 135)),
						}.Layout(gtx, ui.labels.connectedClients.Layout)

						down := float32(1)

						if g.Running {
							ui.labels.symbol.Text = ui.spinners.run.Next()
							ui.labels.symbol.Color = nrgba.Green.Color()

							ui.labels.acronym.Text = "RUN"
							ui.labels.acronym.Color = nrgba.Green.Color()
							down = .5
						} else {
							ui.labels.acronym.Color = nrgba.Slate.Color()
							ui.labels.symbol.Color = nrgba.Slate.Color()
							ui.labels.acronym.Text = "IDLE"
						}

						ui.labels.symbol.Text = ui.spinners.stop.Next()
						layout.Inset{
							Top:  unit.Dp(48 + down),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 135)),
						}.Layout(gtx, ui.labels.symbol.Layout)

						layout.Inset{
							Top:  unit.Dp(50),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 124)),
						}.Layout(gtx, ui.labels.acronym.Layout)

						layout.Inset{
							Top:  unit.Dp(2),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 135)),
						}.Layout(gtx, ui.labels.version.Layout)

						ui.labels.fps.Color = nrgba.FPS(g.fps.FPS()).Color()
						ui.labels.fps.Text = fmt.Sprintf("%s FPS", g.fps)
						layout.Inset{
							Top:  unit.Dp(18),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 135)),
						}.Layout(gtx, ui.labels.fps.Layout)

						o, p, s := server.Scores()

						ui.labels.purpleScore.Text = fmt.Sprintf("%d", p)
						layout.Inset{
							Top:  unit.Dp(2),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 35)),
						}.Layout(gtx, ui.labels.purpleScore.Layout)

						ui.labels.orangeScore.Text = fmt.Sprintf("%d", o)
						layout.Inset{
							Top:  unit.Dp(18),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 35)),
						}.Layout(gtx, ui.labels.orangeScore.Layout)

						ui.labels.selfScore.Text = fmt.Sprintf("%d", s)
						layout.Inset{
							Top:  unit.Dp(34),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 35)),
						}.Layout(gtx, ui.labels.selfScore.Layout)

						decorate.Label(&ui.labels.clock, server.Clock())
						layout.Inset{
							Top:  unit.Dp(2),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 90)),
						}.Layout(gtx, ui.labels.clock.Layout)

						for i, t := range server.Regielekis() {
							ui.labels.regielekis[i].Color = team.None.Color()
							if t != team.None.Name {
								ui.labels.regielekis[i].Color = nrgba.Regieleki.Color()
							}

							ui.labels.regielekiUnderlines[i].Color = team.Color(t).Color()
						}

						for i := range ui.labels.regielekis {
							layout.Inset{
								Top:  unit.Dp(18),
								Left: unit.Dp(float32(gtx.Constraints.Max.X-90) + float32(i*12)),
							}.Layout(gtx, ui.labels.regielekis[i].Layout)

							layout.Inset{
								Top:  unit.Dp(15),
								Left: unit.Dp(float32(gtx.Constraints.Max.X-90) + float32(i*12)),
							}.Layout(gtx, ui.labels.regielekiUnderlines[i].Layout)
						}

						b := server.Bottom()
						for i := range ui.labels.regiBottoms {
							ui.labels.regiBottoms[i].Color = team.None.Color()
							ui.labels.regiBottoms[i].Text = "R"
							ui.labels.regiBottomUnderlines[i].Color = ui.labels.regiBottoms[i].Color

							if i < len(b) {
								t := b[i]
								ui.labels.regiBottoms[i].Text = strings.ToUpper(string(t.Name[4]))
								ui.labels.regiBottoms[i].Color = nrgba.Objective(t.Name).Color()
								ui.labels.regiBottomUnderlines[i].Color = team.Color(t.Team).Color()
							}

							layout.Inset{
								Top:  unit.Dp(34),
								Left: unit.Dp(float32(gtx.Constraints.Max.X-90) + float32(i*12)),
							}.Layout(gtx, ui.labels.regiBottoms[i].Layout)

							layout.Inset{
								Top:  unit.Dp(31),
								Left: unit.Dp(float32(gtx.Constraints.Max.X-90) + float32(i*12)),
							}.Layout(gtx, ui.labels.regiBottomUnderlines[i].Layout)
						}

						ui.labels.uptime.Text = g.performance.uptime

						layout.Inset{
							Top:  unit.Dp(50),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 90)),
						}.Layout(gtx, ui.labels.uptime.Layout)

						layout.Inset{
							Top: unit.Dp(65),
						}.Layout(
							gtx,
							func(gtx layout.Context) layout.Dimensions {
								return ui.textblocks.feed.Layout(gtx, notify.Feeds())
							},
						)

						return layout.Dimensions{Size: gtx.Constraints.Max}
					},
					)
				},
					func(gtx layout.Context) layout.Dimensions {
						if ui.menu.hideRight.Text == "⇇" {
							return layout.Dimensions{}
						}

						return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
							// Right-side criteria.
							{
								layout.Inset{
									Top: unit.Dp(float32(gtx.Constraints.Max.Y - int(float32(ui.buttons.start.Size.Y)*1.5))),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Flex{Axis: layout.Horizontal}.Layout(
												gtx,
												layout.Flexed(.5, layout.Spacer{Width: unit.Dp(25)}.Layout),
												layout.Rigid(ui.buttons.start.Layout),
												layout.Flexed(.5, layout.Spacer{Width: unit.Dp(25)}.Layout),
												layout.Rigid(ui.buttons.stop.Layout),
												layout.Flexed(.5, layout.Spacer{Width: unit.Dp(25)}.Layout),
											)
										}),
									)
								})
							}

							{
								dims := layout.Inset{
									Top: unit.Dp(60),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									ui.buttons.projector.SetImage(notify.Preview)
									return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return ui.buttons.projector.Layout(g.header.Collection.Cascadia().Theme, gtx)
									})
								})

								layout.Inset{
									Top:  unit.Dp(dims.Size.Y + 57),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 150)),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									ui.screens.purple.Image = notify.PurpleScore
									return layout.UniformInset(unit.Dp(5)).Layout(gtx, ui.screens.purple.Layout)
								})

								layout.Inset{
									Top:  unit.Dp(dims.Size.Y + 119),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 150)),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									ui.screens.orange.Image = notify.OrangeScore
									return layout.UniformInset(unit.Dp(5)).Layout(gtx, ui.screens.orange.Layout)
								})

								layout.Inset{
									Top:  unit.Dp(dims.Size.Y + 181),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 68)),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									ui.screens.aeos.Image = notify.Energy
									return layout.UniformInset(unit.Dp(5)).Layout(gtx, ui.screens.aeos.Layout)
								})

								layout.Inset{
									Top:  unit.Dp(dims.Size.Y + 243),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 68)),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									ui.screens.time.Image = notify.Time
									return layout.UniformInset(unit.Dp(5)).Layout(gtx, ui.screens.time.Layout)
								})
							}

							return layout.Dimensions{Size: gtx.Constraints.Max}
						},
						)
					},
				)
			})

			for _, e := range gtx.Events(g) {
				switch event := e.(type) {
				case key.Event:
					if event.State != key.Release {
						continue
					}

					switch event.Modifiers {
					case key.ModCtrl:
						switch event.Name {
						case "C":
							g.next(is.Projecting)
						case "F":
							g.header.Resize()
						case "S":
							btn := ui.buttons.start
							if g.Running {
								btn = ui.buttons.stop
							}
							btn.Click(btn)
						case "W":
							g.next(is.Closing)
						}
					default:

					}
				}
			}

			area := clip.Rect(gtx.Constraints).Push(gtx.Ops)
			key.InputOp{
				Tag:  g,
				Keys: key.Set(""),
			}.Add(gtx.Ops)
			area.Pop()

			g.frame(gtx, event)
		default:
			notify.Missed(event, "Main")
		}
	}
}

func (g *GUI) mainUI() *main {
	ui := &main{}

	var err error

	ui.split.vertical = split.NewVertical(1)

	ui.buttons.stop = &button.Widget{
		Text:            "Stop",
		Font:            g.header.Collection.Calibri(),
		OnHoverHint:     func() { g.header.Tip("Stop capturing events (Ctrl+s)") },
		Disabled:        true,
		Released:        nrgba.Disabled,
		BorderWidth:     unit.Sp(1.5),
		Size:            image.Pt(60, 25),
		TextInsetBottom: -2,
		Click: func(this *button.Widget) {
			this.Deactivate()
			this.Disabled = true
			this.Released = nrgba.Disabled

			ui.buttons.start.Deactivate()
			ui.buttons.start.Disabled = false
			ui.buttons.start.Released = nrgba.PastelGreen.Alpha(150)

			g.Actions <- Stop
			g.Running = false
			g.Preview = true

			ui.menu.startstop.Text = "▶"
			ui.menu.startstop.OnHoverHint = ui.buttons.start.OnHoverHint
			ui.menu.startstop.Pressed = nrgba.PastelGreen
		},
	}

	ui.buttons.start = &button.Widget{
		Text:            "Start",
		Font:            g.header.Collection.Calibri(),
		OnHoverHint:     func() { g.header.Tip("Start capturing events (Ctrl+s)") },
		Released:        nrgba.PastelGreen.Alpha(150),
		Pressed:         nrgba.Transparent80,
		BorderWidth:     unit.Sp(1.5),
		Size:            ui.buttons.stop.Size,
		TextInsetBottom: ui.buttons.stop.TextInsetBottom,
		Click: func(this *button.Widget) {
			g.Preview = false

			ui.buttons.stop.Deactivate()
			ui.buttons.stop.Disabled = false
			ui.buttons.stop.Released = nrgba.PastelRed.Alpha(150)

			this.Deactivate()
			this.Disabled = true
			this.Released = nrgba.Disabled

			g.Actions <- Config
			g.Running = true

			ui.menu.startstop.Text = "⏹"
			ui.menu.startstop.OnHoverHint = ui.buttons.stop.OnHoverHint
			ui.menu.startstop.Pressed = nrgba.PastelRed
		},
	}

	ui.textblocks.feed, err = textblock.New(g.header.Collection.Cascadia(), notify.MaxPosts)
	if err != nil {
		ui.textblocks.feed = &textblock.Widget{}
		notify.Error("Failed to load font: (%v)", err)
	}

	ui.buttons.projector = &button.ImageWidget{
		HintEvent: func() { g.header.Tip("Open a projector window") },

		Widget: &screen.Widget{
			Border:      true,
			BorderColor: nrgba.Transparent,
		},
		Click: func(this *button.ImageWidget) {
			if !ui.buttons.stop.Disabled {
				ui.buttons.stop.Click(ui.buttons.stop)
			}

			g.next(is.Projecting)
		},
	}

	ui.labels.audio = material.Caption(g.header.Collection.NotoSans().Theme, audio.Label())
	ui.labels.audio.Color = nrgba.Slate.Color()
	ui.labels.audio.Alignment = text.Middle
	ui.labels.audio.Font.Weight = font.ExtraBold

	ui.labels.discord = material.Caption(g.header.Collection.NotoSans().Theme, "👾 Discord Disabled")
	ui.labels.discord.Color = nrgba.Discord.Color()
	ui.labels.audio.Alignment = text.Middle
	ui.labels.discord.Font.Weight = font.ExtraBold

	ui.labels.warning = material.Caption(g.header.Collection.NotoSans().Theme, "⚠ CPU")
	ui.labels.warning.Color = nrgba.Yellow.Alpha(200).Color()
	ui.labels.audio.Alignment = text.Middle
	ui.labels.warning.Font.Weight = font.ExtraBold

	ui.labels.window = material.Caption(g.header.Collection.Calibri().Theme, "")
	ui.labels.window.Color = nrgba.DarkSeafoam.Color()
	ui.labels.window.Alignment = text.Middle
	ui.labels.window.Font.Weight = font.ExtraBold
	ui.labels.window.TextSize = unit.Sp(14)

	ui.labels.cpu = material.H5(g.header.Collection.Calibri().Theme, "")
	ui.labels.cpu.Alignment = text.Middle
	ui.labels.cpu.TextSize = unit.Sp(14)

	ui.labels.cpuGraph = material.H5(g.header.Collection.Cascadia().Theme, "")
	ui.labels.cpuGraph.Color = nrgba.Gray.Color()
	ui.labels.cpuGraph.TextSize = unit.Sp(9)

	ui.labels.ram = material.H5(g.header.Collection.Calibri().Theme, "")
	ui.labels.ram.Alignment = text.Middle
	ui.labels.ram.TextSize = unit.Sp(14)

	ui.labels.ramGraph = material.H5(g.header.Collection.Cascadia().Theme, "")
	ui.labels.ramGraph.Color = nrgba.Gray.Color()
	ui.labels.ramGraph.TextSize = unit.Sp(9)

	ui.labels.holding = material.H5(g.header.Collection.Calibri().Theme, "")
	ui.labels.holding.Color = nrgba.Gold.Color()
	ui.labels.holding.Alignment = text.Middle
	ui.labels.holding.TextSize = unit.Sp(14)

	ui.labels.connectedClients = material.H5(g.header.Collection.Calibri().Theme, "")
	ui.labels.connectedClients.Alignment = text.Middle
	ui.labels.connectedClients.TextSize = unit.Sp(14)

	ui.labels.symbol = material.H5(g.header.Collection.Calibri().Theme, "")
	ui.labels.symbol.Alignment = text.Middle
	ui.labels.symbol.TextSize = unit.Sp(16)
	ui.labels.symbol.Font.Weight = font.ExtraBold
	ui.labels.symbol.Color = nrgba.Slate.Color()

	ui.labels.acronym = material.H5(g.header.Collection.Calibri().Theme, "IDLE")
	ui.labels.acronym.Alignment = text.Middle
	ui.labels.acronym.TextSize = unit.Sp(14)
	ui.labels.acronym.Color = nrgba.Slate.Color()

	ui.labels.fps = material.H5(g.header.Collection.Calibri().Theme, "0 FPS")
	ui.labels.fps.Alignment = text.Middle
	ui.labels.fps.TextSize = unit.Sp(14)

	ui.labels.purpleScore = material.H5(g.header.Collection.Calibri().Theme, "0")
	ui.labels.purpleScore.Color = team.Purple.NRGBA.Color()
	ui.labels.purpleScore.Alignment = text.Middle
	ui.labels.purpleScore.TextSize = unit.Sp(14)

	ui.labels.orangeScore = material.H5(g.header.Collection.Calibri().Theme, "0")
	ui.labels.orangeScore.Color = team.Orange.NRGBA.Color()
	ui.labels.orangeScore.Alignment = text.Middle
	ui.labels.orangeScore.TextSize = unit.Sp(14)

	ui.labels.selfScore = material.H5(g.header.Collection.Calibri().Theme, "0")
	ui.labels.selfScore.Color = team.Self.NRGBA.Color()
	ui.labels.selfScore.Alignment = text.Middle
	ui.labels.selfScore.TextSize = unit.Sp(14)

	ui.labels.clock = material.H5(g.header.Collection.Calibri().Theme, "00:00")
	ui.labels.clock.Alignment = text.Middle
	ui.labels.clock.TextSize = unit.Sp(14)

	ui.screens.purple = &screen.Widget{
		Border:      true,
		BorderColor: team.Purple.NRGBA,
		Image:       notify.PurpleScore,
	}

	ui.screens.orange = &screen.Widget{
		Border:      true,
		BorderColor: team.Orange.NRGBA,
		Image:       notify.OrangeScore,
	}

	ui.screens.aeos = &screen.Widget{
		Border:      true,
		BorderColor: team.Energy.NRGBA,
		Image:       notify.Energy,
	}

	ui.screens.time = &screen.Widget{
		Border:      true,
		BorderColor: team.Time.NRGBA,
		Image:       notify.Time,
	}

	ui.labels.regielekis, ui.labels.regielekiUnderlines = []material.LabelStyle{
		material.H5(g.header.Collection.Calibri().Theme, "E"),
		material.H5(g.header.Collection.Calibri().Theme, "E"),
		material.H5(g.header.Collection.Calibri().Theme, "E"),
	}, []material.LabelStyle{
		material.H5(g.header.Collection.Calibri().Theme, "_"),
		material.H5(g.header.Collection.Calibri().Theme, "_"),
		material.H5(g.header.Collection.Calibri().Theme, "_"),
	}

	for i := range ui.labels.regielekis {
		ui.labels.regielekis[i].Color = team.None.Color()
		ui.labels.regielekis[i].Alignment = text.Middle
		ui.labels.regielekis[i].TextSize = unit.Sp(14)

		ui.labels.regielekiUnderlines[i].Color = team.None.Color()
		ui.labels.regielekiUnderlines[i].Alignment = ui.labels.regielekis[i].Alignment
		ui.labels.regielekiUnderlines[i].TextSize = unit.Sp(18)
		ui.labels.regielekiUnderlines[i].Font.Weight = font.Bold
	}

	ui.labels.regiBottoms, ui.labels.regiBottomUnderlines = []material.LabelStyle{
		material.H5(g.header.Collection.Calibri().Theme, "R"),
		material.H5(g.header.Collection.Calibri().Theme, "R"),
		material.H5(g.header.Collection.Calibri().Theme, "R"),
	}, []material.LabelStyle{
		material.H5(g.header.Collection.Calibri().Theme, "_"),
		material.H5(g.header.Collection.Calibri().Theme, "_"),
		material.H5(g.header.Collection.Calibri().Theme, "_"),
	}

	for i := range ui.labels.regiBottoms {
		ui.labels.regiBottoms[i].Color = team.None.Color()
		ui.labels.regiBottoms[i].Alignment = text.Middle
		ui.labels.regiBottoms[i].TextSize = unit.Sp(14)

		ui.labels.regiBottomUnderlines[i].Color = ui.labels.regiBottoms[i].Color
		ui.labels.regiBottomUnderlines[i].Alignment = ui.labels.regiBottoms[i].Alignment
		ui.labels.regiBottomUnderlines[i].TextSize = unit.Sp(18)
		ui.labels.regiBottomUnderlines[i].Font.Weight = font.Bold
	}

	ui.labels.uptime = material.H5(g.header.Collection.Calibri().Theme, g.performance.uptime)
	ui.labels.uptime.Color = nrgba.DreamyPurple.Color()
	ui.labels.uptime.Alignment = text.Middle
	ui.labels.uptime.TextSize = unit.Sp(14)

	ui.labels.version = material.H5(g.header.Collection.Calibri().Theme, global.Version)
	ui.labels.version.Color = nrgba.Gray.Color()
	ui.labels.version.Alignment = text.Middle
	ui.labels.version.TextSize = unit.Sp(14)

	ui.spinners.run = spinner.Running()
	ui.spinners.stop = spinner.Stopped()

	ui.menu.settings = &button.Widget{
		Text:            "⚙",
		TextSize:        unit.Sp(18),
		TextInsetBottom: -2,
		Font:            g.header.Collection.NishikiTeki(),
		OnHoverHint:     func() { g.header.Tip("Modify capture settings") },

		Released:    nrgba.Transparent80,
		Pressed:     nrgba.SilverPurple,
		BorderWidth: unit.Sp(.1),
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			ui.buttons.projector.Click(ui.buttons.projector)
		},
	}

	ui.menu.client = &button.Widget{
		Text:        "📺",
		Font:        g.header.Collection.NishikiTeki(),
		OnHoverHint: ui.buttons.projector.HintEvent,
		Pressed:     nrgba.Discord.Alpha(100),
		TextSize:    unit.Sp(16),

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			ui.buttons.start.Click(ui.buttons.start)
			go g.client(func() {
				ui.buttons.stop.Click(ui.buttons.stop)
			})
		},
	}

	ui.menu.stats = &button.Widget{
		Text:        "¼",
		Font:        g.header.Collection.NishikiTeki(),
		OnHoverHint: func() { g.header.Tip("View capture statistics") },
		Pressed:     nrgba.Pinkity,
		TextSize:    unit.Sp(15),

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			stats.Data()

			s, ok := state.Dump()
			if !ok {
				notify.Warn(s)
			} else {
				notify.System(s)
			}
		},
	}

	ui.menu.results = &button.Widget{
		Text:        "+/-",
		TextSize:    unit.Sp(12),
		Font:        g.header.Collection.Cascadia(),
		OnHoverHint: func() { g.header.Tip("View win/loss history") },
		Pressed:     nrgba.Seafoam,

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			history.Dump()
		},
	}

	ui.menu.obs = &button.Widget{
		Text:        "obs",
		Font:        g.header.Collection.NishikiTeki(),
		OnHoverHint: func() { g.header.Tip("Open OBS client folder") },
		Pressed:     nrgba.Purple,
		TextSize:    unit.Sp(12),

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			drag := "Drag \"UniteHUD Client\" into any OBS scene."
			if config.Current.Profile == config.ProfileBroadcaster {
				drag = "Drag \"UniteHUD Broadcaster\" into any OBS scene."
			}

			g.ToastOK("Overlay", drag,
				OnToastOK(func() {
					err = open.Run(filepath.Join(global.WorkingDirectory(), "www"))
					if err != nil {
						notify.Error("🖥️  Failed to open www/ directory: %v", err)
						return
					}
				}),
			)
		},
	}

	ui.menu.clear = &button.Widget{
		Text:            "🧹",
		Font:            g.header.Collection.NishikiTeki(),
		OnHoverHint:     func() { g.header.Tip("Clear event history") },
		TextInsetBottom: -2,
		Pressed:         nrgba.Orange,
		TextSize:        unit.Sp(14),

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			notify.CLS()
			notify.System("🧹  Cleared")
		},
	}

	ui.menu.eco = &button.Widget{
		Text:        "⚶",
		Font:        g.header.Collection.NishikiTeki(),
		OnHoverHint: func() { g.header.Tip("Toggle resource saver") },
		Pressed:     nrgba.ForestGreen,
		TextSize:    unit.Sp(16),

		Click: func(this *button.Widget) {
			g.performance.eco = !g.performance.eco

			this.Activate()
			if g.performance.eco {
				this.Deactivate()
			}

			if g.performance.eco {
				notify.System("⚶  Resource saver has been enabled")
			} else {
				notify.System("⚶  Resource saver has been disabled")
			}
		},
	}

	ui.menu.logs = &button.Widget{
		Text:        "🗁",
		Font:        g.header.Collection.NishikiTeki(),
		OnHoverHint: func() { g.header.Tip("Open log directory") },
		Pressed:     nrgba.PastelBabyBlue,
		TextSize:    unit.Sp(16),

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			save.Logs()

			err := save.Open()
			if err != nil {
				notify.Error("🗁  Failed to open \"%s\" (%v)", save.Directory, err)
			}
		},
	}

	ui.menu.record = &button.Widget{
		Text:        "🎬",
		Font:        g.header.Collection.NishikiTeki(),
		OnHoverHint: func() { g.header.Tip("Record matched events") },
		Pressed:     nrgba.Pinkity.Alpha(100),
		TextSize:    15,

		Click: func(this *button.Widget) {
			title := "Record"
			description := "Record and save captured events on your computer?"
			yes := func() {
				defer save.Logs()

				config.Current.Record = true
				notify.System("🎬  Recording captured events in %s", save.Directory)
				this.Text = "■"
			}

			if config.Current.Record {
				title = "Stop"
				description = "Stop recording captured events?"
				yes = func() {
					defer save.Logs()

					notify.System("🎬  Saved captured events in %s", save.Directory)
					this.Text = "🎬"

					err := save.Open()
					if err != nil {
						notify.Error("🎬  Failed to open \"%s\" (%v)", save.Directory, err)
					}
				}
			}

			g.ToastYesNo(title, description, OnToastYes(yes), OnToastNo(this.Deactivate))
		},
	}

	ui.menu.file = &button.Widget{
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
				notify.Error("Failed to open \"%s\" (%v)", config.Current.File(), err)
				return
			}

			// Called once window is closed.
			err = config.Load(config.Current.Profile)
			if err != nil {
				notify.Error("Failed to reload \"%s\" (%v)", config.Current.File(), err)
				return
			}
		},
	}

	ui.menu.startstop = &button.Widget{
		Text:            "▶",
		Font:            g.header.Collection.NishikiTeki(),
		Pressed:         nrgba.PastelGreen,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Disabled:        false,
		OnHoverHint:     ui.buttons.start.OnHoverHint,
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			if this.Text == "▶" {
				ui.buttons.start.Click(ui.buttons.start)
				this.Text = "⏸"
				this.OnHoverHint = ui.buttons.stop.OnHoverHint
				// this.Pressed = nrgba.PastelRed
				this.Released = nrgba.PastelRed
			} else {
				ui.buttons.stop.Click(ui.buttons.stop)
				this.Text = "▶"
				this.OnHoverHint = ui.buttons.start.OnHoverHint
				this.Pressed = nrgba.PastelGreen
				this.Released = nrgba.Nothing
			}
		},
	}

	ui.menu.hideRight = &button.Widget{
		Text:            "⇇",
		Font:            g.header.Collection.NishikiTeki(),
		Pressed:         nrgba.PastelGreen,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		OnHoverHint:     func() { g.header.Tip("Show Main Menu preview area") },
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			if this.Text == "⇉" {
				this.Text = "⇇"
				ui.split.vertical.Ratio = 1
				this.OnHoverHint = func() { g.header.Tip("Show Main Menu preview area") }
			} else {
				this.Text = "⇉"
				ui.split.vertical.Ratio = .7
				this.OnHoverHint = func() { g.header.Tip("Hide Main Menu preview area") }
			}
		},
	}

	ui.menu.hideTop = &button.Widget{
		Text:            "⇈",
		Font:            g.header.Collection.NishikiTeki(),
		Pressed:         nrgba.PastelGreen,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		OnHoverHint:     func() { g.header.Tip("Show Main Menu configuration area") },
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			if this.Text == "⇈" {
				this.Text = "⇊"
				this.OnHoverHint = func() { g.header.Tip("Show Main Menu configuration area") }
			} else {
				this.Text = "⇈"
				this.OnHoverHint = func() { g.header.Tip("Hide Main Menu configuration area") }
			}
		},
	}

	return ui
}
