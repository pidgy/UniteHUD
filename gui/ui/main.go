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
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/skratchdot/open-golang/open"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/detect"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/core/server"
	"github.com/pidgy/unitehud/core/state"
	"github.com/pidgy/unitehud/core/stats"
	"github.com/pidgy/unitehud/core/stats/history"
	"github.com/pidgy/unitehud/core/team"
	"github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/ux/button"
	"github.com/pidgy/unitehud/gui/ux/decorate"
	"github.com/pidgy/unitehud/gui/ux/keys"
	"github.com/pidgy/unitehud/gui/ux/screen"
	"github.com/pidgy/unitehud/gui/ux/spinner"
	"github.com/pidgy/unitehud/gui/ux/split"
	"github.com/pidgy/unitehud/gui/ux/textblock"
	"github.com/pidgy/unitehud/media/audio"
	"github.com/pidgy/unitehud/media/video/device"
	"github.com/pidgy/unitehud/media/video/monitor"
	"github.com/pidgy/unitehud/media/video/window"
	"github.com/pidgy/unitehud/system/discord"
	"github.com/pidgy/unitehud/system/process"
	"github.com/pidgy/unitehud/system/save"
	"github.com/pidgy/unitehud/system/tray"
	"github.com/pidgy/unitehud/system/wapi"
)

// main defines main behavior and state.
type main struct {
	ops op.Ops

	navButtons struct {
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
		hideRight,
		alwaysOnTop,
		screenshot *button.Widget
	}

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
		threads, threadsGraph,
		holding,
		connectedClients,
		symbol,
		acronym,
		hz,
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
		previewImage *button.ImageWidget
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

	keybinds *keys.Bind
}

// main open's the Main Menu window.
func (g *GUI) main() {
	ui := g.mainUI()

	tray.SetStartStopEnabled()
	defer tray.SetStartStopDisabled()

	defer ui.spinners.run.Stop()
	defer ui.spinners.stop.Stop()

	defer g.nav.Remove(g.nav.Add(ui.navButtons.startstop))
	defer g.nav.Remove(g.nav.Add(ui.navButtons.settings))
	defer g.nav.Remove(g.nav.Add(ui.navButtons.client))
	defer g.nav.Remove(g.nav.Add(ui.navButtons.hideRight))
	defer g.nav.Remove(g.nav.Add(ui.navButtons.hideTop))
	defer g.nav.Remove(g.nav.Add(ui.navButtons.obs))
	defer g.nav.Remove(g.nav.Add(ui.navButtons.logs))
	defer g.nav.Remove(g.nav.Add(ui.navButtons.record))
	defer g.nav.Remove(g.nav.Add(ui.navButtons.alwaysOnTop))
	defer g.nav.Remove(g.nav.Add(ui.navButtons.screenshot))

	// defer g.header.Remove(g.header.Add(ui.menu.stats))
	// defer g.header.Remove(g.header.Add(ui.menu.results))
	// defer g.header.Remove(g.header.Add(ui.menu.clear))
	// defer g.header.Remove(g.header.Add(ui.menu.eco))
	// defer g.nav.Remove(g.nav.Add(ui.nav.file))

	g.window.Perform(system.ActionRaise)

	g.nav.Open()

	if config.Current.Crashed != "" {
		notify.Warn("[Crash] %s", config.Current.Crashed)

		err := save.Logs(notify.FeedStrings(), stats.Lines(), stats.Counts())
		if err != nil {
			notify.Warn("[UI] <ini:f:save> logs (%v)", err)
		}

		g.ToastYesNo(
			"Configuration Reset",
			"Recent crash detected. View log directory?",
			toastOnYes(
				func() {
					err := save.Open()
					if err != nil {
						notify.Error("[UI] <ini:f:open> save directory (%v)", err)
						return
					}
				},
			),
			nil,
		)
		err = config.Current.Reset()
		if err != nil {
			notify.Warn("[UI] Failed to reset configuration (%v)", err)
		}
	}

	tray.SetStartStopTitle("Start")

	for is.Currently(is.MainMenu) {
		if !g.open {
			time.Sleep(time.Millisecond * 10)
			continue
		}

		// if g.performance.eco && state.Idle() > time.Minute*30 && !ui.buttons.stop.Disabled {
		// 	desktop.Notification("Eco Mode").
		// 		Says("UniteHUD is still running and no matches detected for 30 minutes").
		// 		When(clicked.OpenUniteHUD).
		// 		Send()
		// }

		switch event := g.window.NextEvent().(type) {
		case app.ConfigEvent:
			g.dimensions.size = event.Config.Size
		case system.DestroyEvent:
			is.Next(is.Closing)
			return
		case system.FrameEvent:
			gtx := layout.NewContext(&ui.ops, event)

			g.dimensions.size = event.Size

			decorate.Background(gtx)
			decorate.Label(&ui.labels.cpu, "%s", process.Usage.CPU)
			decorate.Label(&ui.labels.cpuGraph, "%s", stats.CPUGraph())
			decorate.Label(&ui.labels.ram, "%s", process.Usage.RAM)
			decorate.Label(&ui.labels.ramGraph, "%s", stats.RAMGraph())
			decorate.Label(&ui.labels.threads, "%s", process.Usage.Threads)
			decorate.Label(&ui.labels.threadsGraph, "%s", stats.ThreadsGraph())
			decorate.Label(&ui.labels.holding, "%s", ui.labels.holding.Text)
			decorate.ForegroundAlt(&ui.labels.cpuGraph.Color)
			decorate.ForegroundAlt(&ui.labels.ramGraph.Color)
			decorate.ForegroundAlt(&ui.labels.threadsGraph.Color)

			g.nav.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return ui.split.vertical.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
						if ui.navButtons.hideTop.Text == "⇊" {
							return ui.textblocks.feed.Layout(gtx, notify.Feeds())
						}

						warnings, nonwarnings := []string{}, []string{}
						if config.Current.Advanced.DecreasedCaptureLevel > 0 {
							nonwarnings = append(warnings, fmt.Sprintf("Match Rate Factor: -%d", config.Current.Advanced.DecreasedCaptureLevel))
						}

						if len(warnings) > 0 {
							layout.Inset{
								Left: unit.Dp(3),
								Top:  unit.Dp(32),
							}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								ui.labels.warning.Text = fmt.Sprintf("⚠ CPU (%s)", strings.Join(warnings, ","))
								return ui.labels.warning.Layout(gtx)
							})
						}

						if len(nonwarnings) > 0 {
							ui.labels.warning.Text = fmt.Sprintf("✔ CPU %s", strings.Join(nonwarnings, ","))
							ui.labels.warning.Color = nrgba.PastelGreen.Color()

							layout.Inset{
								Left: unit.Dp(3),
								Top:  unit.Dp(32),
							}.Layout(gtx, ui.labels.warning.Layout)
						}

						layout.Inset{
							Left: unit.Dp(2),
							Top:  unit.Dp(.1),
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							if !discord.Connected() {
								ui.labels.discord.Color.A = 127
								ui.labels.discord.Text = "👾 Discord RPC Disabled"
							} else {
								ui.labels.discord.Color.A = 200
								ui.labels.discord.Text = fmt.Sprintf("👾 %s: %s", strings.ReplaceAll(discord.Current.Details, "UniteHUD - ", ""), discord.Current.State)
							}
							return ui.labels.discord.Layout(gtx)
						})

						layout.Inset{
							Left: unit.Dp(3),
							Top:  unit.Dp(17),
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							ui.labels.audio.Text = audio.Label()
							return ui.labels.audio.Layout(gtx)
						})

						switch {
						case device.IsActive():
							fps := device.FPS()
							ui.labels.window.Color = nrgba.Percent(fps / float64(config.Current.Video.Capture.Device.FPS)).Color()
							ui.labels.window.Text = fmt.Sprintf("📺 %s %.0fFPS", device.Name(config.Current.Video.Capture.Device.Index), fps)
						case window.IsActive():
							ui.labels.window.Text = fmt.Sprintf("📺 %s (%s) -> (%s)", config.Current.Video.Capture.Window.Name, window.Resolution(), notify.PreviewResolutionString())
						case monitor.IsActive():
							ui.labels.window.Text = fmt.Sprintf("📺 %s (%s) -> (%s)", config.Current.Video.Capture.Monitor.Name, monitor.Resolution(), notify.PreviewResolutionString())
						}

						layout.Inset{
							Left: unit.Dp(2),
							Top:  unit.Dp(50),
						}.Layout(gtx, ui.labels.window.Layout)

						layout.Inset{
							Top:  unit.Dp(28),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 560)),
						}.Layout(gtx, ui.labels.threads.Layout)

						layout.Inset{
							Top:  unit.Dp(1),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 600)),
						}.Layout(gtx, ui.labels.threadsGraph.Layout)

						layout.Inset{
							Top:  unit.Dp(28),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 410)),
						}.Layout(gtx, ui.labels.cpu.Layout)

						layout.Inset{
							Top:  unit.Dp(1),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 450)),
						}.Layout(gtx, ui.labels.cpuGraph.Layout)

						layout.Inset{
							Top:  unit.Dp(28),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 255)),
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

						if server.Clients() > 0 {
							ui.labels.connectedClients.Color = nrgba.Seafoam.Color()
						} else {
							ui.labels.connectedClients.Color = nrgba.PaleRed.Color()
						}
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

						ui.labels.hz.Color = nrgba.Status(g.hz.PS()).Color()
						ui.labels.hz.Text = fmt.Sprintf("%sHz", g.hz)
						layout.Inset{
							Top:  unit.Dp(18),
							Left: unit.Dp(float32(gtx.Constraints.Max.X - 135)),
						}.Layout(gtx, ui.labels.hz.Layout)

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

						decorate.Label(&ui.labels.clock, "%s", server.Clock())
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

						objs := server.ObjectivesSecured()
						for i, o := range objs {
							if o.Name == server.ObjectiveRegieleki {
								continue
							}
							if i+1 > len(ui.labels.regiBottoms) {
								continue
							}

							ui.labels.regiBottoms[i].Color = team.None.Color()
							ui.labels.regiBottoms[i].Text = "R"
							ui.labels.regiBottomUnderlines[i].Color = ui.labels.regiBottoms[i].Color

							ui.labels.regiBottoms[i].Text = strings.ToUpper(string(o.Name[4]))
							ui.labels.regiBottoms[i].Color = nrgba.Objective(o.Name).Color()
							ui.labels.regiBottomUnderlines[i].Color = team.Color(o.Team).Color()

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
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return ui.textblocks.feed.Layout(gtx, notify.Feeds())
						})

						return layout.Dimensions{Size: gtx.Constraints.Max}
					},
					)
				},
					func(gtx layout.Context) layout.Dimensions {
						if ui.navButtons.hideRight.Text == "⇇" {
							return layout.Dimensions{}
						}

						return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
							// Right-side criteria.
							{
								layout.Inset{
									Top: unit.Dp(float32(gtx.Constraints.Max.Y - int(float32(ui.buttons.start.Size.Y)*1.5))),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
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
									ui.buttons.previewImage.SetImage(notify.Preview())
									return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return ui.buttons.previewImage.Layout(g.nav.Cascadia().Theme, gtx)
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

			switch ui.keybinds.Event(gtx) {
			case keys.Ctrl("M"):
				g.minimize()
			case keys.Ctrl("C"):
				ui.navButtons.settings.Click(ui.navButtons.settings)
			case keys.Ctrl("F"):
				g.nav.Resize()
			case keys.Ctrl("P"):
				ui.navButtons.client.Click(ui.navButtons.client)
			case keys.Ctrl("S"):
				btn := ui.buttons.start
				if g.Running {
					btn = ui.buttons.stop
				}

				btn.Click(btn)
			case keys.Ctrl("W"):
				is.Next(is.Closing)
			case keys.Escape():
				if g.dimensions.fullscreen {
					g.nav.Resize()
				}
				if g.Running {
					// ui.buttons.stop.Click(ui.buttons.stop)
				}
			case keys.F11():
				g.nav.Resize()
			case keys.Command("Shift"), keys.Ctrl("V"):
				ui.navButtons.screenshot.Click(ui.navButtons.screenshot)
			}

			g.frame(gtx, event)

			ui.onFrame(1, ui.onFrame1, g)
			ui.onFrame(12, ui.onFrame2, g)
			ui.onFrame(24, ui.onFrame3, g)
		default:
			notify.Missed(event, "Main")
		}
	}
}

// mainUI creates the main menu's UI state manager.
func (g *GUI) mainUI() *main {
	ui := &main{
		keybinds: keys.New().
			Bind(keys.NoMod, keys.Escape(), keys.F11()).
			Bind(keys.CtrlMod, "C", "F", "M", "P", "S", "V", "W").
			Bind(keys.CommandMod, "Shift"),
	}

	var err error

	ui.split.vertical = split.NewVertical(1)

	ui.buttons.stop = &button.Widget{
		Text:            "Stop",
		Font:            g.nav.Calibri(),
		Hint:            "Stop capturing events (Ctrl+S)",
		OnHoverHint:     g.nav.Tip,
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

			g.Running = false
			g.Preview = true

			ui.navButtons.startstop.Text = "▶"
			ui.navButtons.startstop.Hint = ui.buttons.start.Hint
			ui.navButtons.startstop.Pressed = nrgba.PastelGreen
			ui.navButtons.startstop.Released = nrgba.Nothing

			detect.Pause()
			server.Clear()
			team.Clear()
			server.SetNotReady()

			err := save.Logs(notify.FeedStrings(), stats.Lines(), stats.Counts())
			if err != nil {
				notify.Warn("[UI] <ini:f:save> logs (%v)", err)
			}

			tray.SetStartStopTitle("Start")

			notify.Announce("[UI] Stopped %s", exe.Title)
		},
	}

	ui.buttons.start = &button.Widget{
		Text:            "Start",
		Font:            g.nav.Calibri(),
		Hint:            "Start capturing events (Ctrl+S)",
		OnHoverHint:     g.nav.Tip,
		Released:        nrgba.PastelGreen.Alpha(150),
		Pressed:         nrgba.Transparent80,
		BorderWidth:     unit.Sp(1.5),
		Size:            ui.buttons.stop.Size,
		TextInsetBottom: ui.buttons.stop.TextInsetBottom,
		Click: func(this *button.Widget) {
			config.Current.Reload()

			g.Preview = false

			ui.buttons.stop.Deactivate()
			ui.buttons.stop.Disabled = false
			ui.buttons.stop.Released = nrgba.PastelRed.Alpha(150)

			this.Deactivate()
			this.Disabled = true
			this.Released = nrgba.Disabled

			ui.navButtons.startstop.Text = "⏸"
			ui.navButtons.startstop.Hint = ui.buttons.stop.Hint
			ui.navButtons.startstop.Pressed = nrgba.Nothing
			ui.navButtons.startstop.Released = nrgba.PastelRed

			server.SetConfig(true)
			detect.Resume()
			notify.Clear()
			server.Clear()
			state.Clear()
			stats.Clear()
			team.Clear()
			server.SetReady()

			tray.SetStartStopTitle("Stop")

			g.Running = true

			notify.Announce("[UI] Started %s", exe.Title)
		},
	}

	ui.textblocks.feed, err = textblock.New(g.nav.Cascadia(), 75)
	if err != nil {
		ui.textblocks.feed = &textblock.Widget{}
		notify.Warn("[UI] <ini:f:load> font: (%v)", err)
	}

	ui.buttons.previewImage = &button.ImageWidget{
		Hint:        "Open the configuration settings window",
		OnHoverHint: g.nav.Tip,

		Widget: &screen.Widget{
			Border:      true,
			BorderColor: nrgba.Transparent,
		},
		Click: func(this *button.ImageWidget) {
			if !ui.buttons.stop.Disabled {
				ui.buttons.stop.Click(ui.buttons.stop)
			}

			if ui.navButtons.alwaysOnTop.Radio {
				ui.navButtons.alwaysOnTop.Click(ui.navButtons.alwaysOnTop)
			}

			is.Next(is.Configuring)
		},
	}

	// Labels.
	{
		ui.labels.audio = material.Caption(g.nav.NotoSans().Theme, audio.Label())
		ui.labels.audio.Color = nrgba.Slate.Color()
		ui.labels.audio.Alignment = text.Middle
		ui.labels.audio.Font.Weight = font.ExtraBold

		ui.labels.discord = material.Caption(g.nav.NotoSans().Theme, "👾 Discord Disabled")
		ui.labels.discord.Color = nrgba.Discord.Color()
		ui.labels.audio.Alignment = text.Middle
		ui.labels.discord.Font.Weight = font.ExtraBold

		ui.labels.warning = material.Caption(g.nav.NotoSans().Theme, "⚠ CPU")
		ui.labels.warning.Color = nrgba.Yellow.Alpha(200).Color()
		ui.labels.audio.Alignment = text.Middle
		ui.labels.warning.Font.Weight = font.ExtraBold

		ui.labels.window = material.Caption(g.nav.Calibri().Theme, "")
		ui.labels.window.Color = nrgba.PastelGreen.Color()
		ui.labels.window.Alignment = text.Middle
		ui.labels.window.Font.Weight = font.Medium
		ui.labels.window.TextSize = unit.Sp(14)

		ui.labels.cpu = material.H5(g.nav.Calibri().Theme, "")
		ui.labels.cpu.Alignment = text.Middle
		ui.labels.cpu.TextSize = unit.Sp(14)

		ui.labels.cpuGraph = material.H5(g.nav.Cascadia().Theme, "")
		ui.labels.cpuGraph.Color = nrgba.Gray.Color()
		ui.labels.cpuGraph.TextSize = unit.Sp(9)

		ui.labels.ram = material.H5(g.nav.Calibri().Theme, "")
		ui.labels.ram.Alignment = text.Middle
		ui.labels.ram.TextSize = unit.Sp(14)

		ui.labels.ramGraph = material.H5(g.nav.Cascadia().Theme, "")
		ui.labels.ramGraph.Color = nrgba.Gray.Color()
		ui.labels.ramGraph.TextSize = unit.Sp(9)

		ui.labels.threads = material.H5(g.nav.Calibri().Theme, "")
		ui.labels.threads.Alignment = text.Middle
		ui.labels.threads.TextSize = unit.Sp(14)

		ui.labels.threadsGraph = material.H5(g.nav.Cascadia().Theme, "")
		ui.labels.threadsGraph.Color = nrgba.Gray.Color()
		ui.labels.threadsGraph.TextSize = unit.Sp(9)

		ui.labels.holding = material.H5(g.nav.Calibri().Theme, "")
		ui.labels.holding.Color = nrgba.Gold.Color()
		ui.labels.holding.Alignment = text.Middle
		ui.labels.holding.TextSize = unit.Sp(14)

		ui.labels.connectedClients = material.H5(g.nav.Calibri().Theme, "")
		ui.labels.connectedClients.Alignment = text.Middle
		ui.labels.connectedClients.TextSize = unit.Sp(14)
		ui.labels.connectedClients.Text = "OBS"

		ui.labels.symbol = material.H5(g.nav.Calibri().Theme, "")
		ui.labels.symbol.Alignment = text.Middle
		ui.labels.symbol.TextSize = unit.Sp(16)
		ui.labels.symbol.Font.Weight = font.ExtraBold
		ui.labels.symbol.Color = nrgba.Slate.Color()

		ui.labels.acronym = material.H5(g.nav.Calibri().Theme, "IDLE")
		ui.labels.acronym.Alignment = text.Middle
		ui.labels.acronym.TextSize = unit.Sp(14)
		ui.labels.acronym.Color = nrgba.Slate.Color()

		ui.labels.hz = material.H5(g.nav.Calibri().Theme, "0 FPS")
		ui.labels.hz.Alignment = text.Middle
		ui.labels.hz.TextSize = unit.Sp(14)

		ui.labels.purpleScore = material.H5(g.nav.Calibri().Theme, "0")
		ui.labels.purpleScore.Color = team.Purple.NRGBA.Color()
		ui.labels.purpleScore.Alignment = text.Middle
		ui.labels.purpleScore.TextSize = unit.Sp(14)

		ui.labels.orangeScore = material.H5(g.nav.Calibri().Theme, "0")
		ui.labels.orangeScore.Color = team.Orange.NRGBA.Color()
		ui.labels.orangeScore.Alignment = text.Middle
		ui.labels.orangeScore.TextSize = unit.Sp(14)

		ui.labels.selfScore = material.H5(g.nav.Calibri().Theme, "0")
		ui.labels.selfScore.Color = team.Self.NRGBA.Color()
		ui.labels.selfScore.Alignment = text.Middle
		ui.labels.selfScore.TextSize = unit.Sp(14)

		ui.labels.clock = material.H5(g.nav.Calibri().Theme, "00:00")
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
			material.H5(g.nav.Calibri().Theme, "E"),
			material.H5(g.nav.Calibri().Theme, "E"),
			material.H5(g.nav.Calibri().Theme, "E"),
		}, []material.LabelStyle{
			material.H5(g.nav.Calibri().Theme, "_"),
			material.H5(g.nav.Calibri().Theme, "_"),
			material.H5(g.nav.Calibri().Theme, "_"),
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
			material.H5(g.nav.Calibri().Theme, "R"),
			material.H5(g.nav.Calibri().Theme, "R"),
			material.H5(g.nav.Calibri().Theme, "R"),
		}, []material.LabelStyle{
			material.H5(g.nav.Calibri().Theme, "_"),
			material.H5(g.nav.Calibri().Theme, "_"),
			material.H5(g.nav.Calibri().Theme, "_"),
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

		ui.labels.uptime = material.H5(g.nav.Calibri().Theme, g.performance.uptime)
		ui.labels.uptime.Color = nrgba.DreamyPurple.Color()
		ui.labels.uptime.Alignment = text.Middle
		ui.labels.uptime.TextSize = unit.Sp(14)

		ui.labels.version = material.H5(g.nav.Calibri().Theme, exe.Version)
		ui.labels.version.Color = nrgba.Gray.Color()
		ui.labels.version.Alignment = text.Middle
		ui.labels.version.TextSize = unit.Sp(14)

		ui.spinners.run = spinner.Running()
		ui.spinners.stop = spinner.Stopped()
	}

	ui.navButtons.settings = &button.Widget{
		Text:            "⚙",
		TextSize:        unit.Sp(18),
		TextInsetBottom: -2,
		Font:            g.nav.NishikiTeki(),
		// Hint: "Modify capture settings",
		Hint:        "Modify capture settings",
		OnHoverHint: g.nav.Tip,
		Released:    nrgba.Transparent80,
		Pressed:     nrgba.SilverPurple,
		BorderWidth: unit.Sp(.1),
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			ui.buttons.previewImage.Click(ui.buttons.previewImage)
		},
	}

	ui.navButtons.client = &button.Widget{
		Text:        "📺",
		Font:        g.nav.NishikiTeki(),
		Hint:        "Open a projector window (Ctrl+P)",
		OnHoverHint: g.nav.Tip,
		Pressed:     nrgba.Discord.Alpha(100),
		TextSize:    unit.Sp(16),

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			this.Disabled = true
			go g.projector(func() { this.Disabled = false })
		},
	}

	ui.navButtons.stats = &button.Widget{
		Text:        "¼",
		Font:        g.nav.NishikiTeki(),
		Hint:        "View capture statistics",
		OnHoverHint: g.nav.Tip,
		Pressed:     nrgba.Pinkity,
		TextSize:    unit.Sp(15),

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			stats.Data()

			s, ok := state.Dump()
			if !ok {
				notify.Warn("%s", s)
			} else {
				notify.System("%s", s)
			}
		},
	}

	ui.navButtons.results = &button.Widget{
		Text:        "+/-",
		TextSize:    unit.Sp(12),
		Font:        g.nav.Cascadia(),
		Hint:        "View win/loss history",
		OnHoverHint: g.nav.Tip,
		Pressed:     nrgba.Seafoam,

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			history.Dump()
		},
	}

	ui.navButtons.obs = &button.Widget{
		Text:        "obs",
		Font:        g.nav.NishikiTeki(),
		Hint:        "Open OBS client folder",
		OnHoverHint: g.nav.Tip,
		Pressed:     nrgba.Purple,
		TextSize:    unit.Sp(12),

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			g.ToastOK("Overlay", `Drag "UniteHUD Client.html" into OBS.`,
				toastOnOK(func() {
					err = open.Run(filepath.Join(exe.Directory(), "www"))
					if err != nil {
						notify.Error("[UI] <ini:f:open> www/ directory: %v", err)
						return
					}
				}),
				toastOnClose(nil),
			)
		},
	}

	ui.navButtons.clear = &button.Widget{
		Text:            "🧹",
		Font:            g.nav.NishikiTeki(),
		Hint:            "Clear event history",
		OnHoverHint:     g.nav.Tip,
		TextInsetBottom: -2,
		Pressed:         nrgba.Orange,
		TextSize:        unit.Sp(14),

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			notify.CLS()
			notify.System("[UI] Cleared")
		},
	}

	// ui.nav.eco = &button.Widget{
	// 	Text:        "🌳",
	// 	Font:        g.nav.NishikiTeki(),
	// 	Hint: "Toggle resource saver",
	// OnHoverHintStr:
	// 	g.nav.Tip,
	// 	Pressed:     nrgba.DarkSeafoam,
	// 	TextSize:    unit.Sp(16),

	// 	Click: func(this *button.Widget) {
	// 		g.performance.eco = !g.performance.eco

	// 		this.Activate()
	// 		if g.performance.eco {
	// 			this.Deactivate()
	// 		}

	// 		if g.performance.eco {
	// 			notify.System("[UI] Resource saver has been enabled")
	// 		} else {
	// 			notify.System("[UI] Resource saver has been disabled")
	// 		}
	// 	},
	// }

	ui.navButtons.logs = &button.Widget{
		Text:        "🗁",
		Font:        g.nav.NishikiTeki(),
		Hint:        "Open log directory",
		OnHoverHint: g.nav.Tip,
		Pressed:     nrgba.PastelBabyBlue,
		TextSize:    unit.Sp(16),

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			err = save.Logs(notify.FeedStrings(), stats.Lines(), stats.Counts())
			if err != nil {
				notify.Warn("[UI] <ini:f:save> logs (%v)", err)
			}

			err := save.Open()
			if err != nil {
				notify.Warn("[UI] <ini:f:open>: %s (%v)", save.Directory, err)
			}
		},
	}

	ui.navButtons.record = &button.Widget{
		Text:        "🎬",
		Font:        g.nav.NishikiTeki(),
		Hint:        "Record matched events",
		OnHoverHint: g.nav.Tip,
		Pressed:     nrgba.Pinkity.Alpha(100),
		TextSize:    14,

		Click: func(this *button.Widget) {
			title := "Record"
			description := "Record and save captured events on your computer?"
			yes := func() {
				config.Current.Record = true
				notify.System("[UI] Recording captured events in %s", save.Directory)
				this.Text = "■"
				this.TextSize = 15

				err := save.Logs(notify.FeedStrings(), stats.Lines(), stats.Counts())
				if err != nil {
					notify.Warn("[UI] <ini:f:save> logs (%v)", err)
				}
			}

			if config.Current.Record {
				title = "Stop"
				description = "Stop recording captured events?"
				yes = func() {
					notify.System("[UI] Saved captured events in %s", save.Directory)
					this.Text = "🎬"
					this.TextSize = 14

					err := save.Open()
					if err != nil {
						notify.Error("[UI] <ini:f:open> \"%s\" (%v)", save.Directory, err)
					}

					err = save.Logs(notify.FeedStrings(), stats.Lines(), stats.Counts())
					if err != nil {
						notify.Warn("[UI] <ini:f:save> logs (%v)", err)
					}

					config.Current.Record = false
				}
			}

			g.ToastYesNo(title, description, toastOnYes(yes), toastOnNo(this.Deactivate))
		},
	}

	ui.navButtons.file = &button.Widget{
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

			exe := "C:\\Windows\\system32\\notepad.exe"
			err := exec.Command(exe, config.Current.File()).Run()
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
		},
	}

	ui.navButtons.startstop = &button.Widget{
		Text:            "▶",
		Font:            g.nav.NishikiTeki(),
		Pressed:         nrgba.PastelGreen,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Disabled:        false,
		OnHoverHint:     g.nav.Tip,
		Hint:            ui.buttons.start.Hint,
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			if this.Text == "▶" {
				ui.buttons.start.Click(ui.buttons.start)
				this.Text = "⏸"
				this.Hint = ui.buttons.stop.Hint
				this.Released = nrgba.PastelRed
				tray.SetStartStopTitle("Stop")
			} else {
				ui.buttons.stop.Click(ui.buttons.stop)
				this.Text = "▶"
				this.Hint = ui.buttons.start.Hint
				this.Pressed = nrgba.PastelGreen
				this.Released = nrgba.Nothing
				tray.SetStartStopTitle("Start")
			}
		},
	}

	// config.Current.Advanced.Matching.Disabled.Previews = true

	ui.navButtons.hideRight = &button.Widget{
		Text:            "⇇",
		Font:            g.nav.NishikiTeki(),
		Pressed:         nrgba.Gray,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Hint:            "Show Main Menu preview area",
		OnHoverHint:     g.nav.Tip,
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			hidden := this.Text != "⇉"

			if !hidden {
				this.Text = "⇇"
				ui.split.vertical.Ratio = 1
				this.Hint = "Show Main Menu preview area"
				config.Current.Advanced.Matching.Disabled.Previews = true
			} else {
				this.Text = "⇉"
				ui.split.vertical.Ratio = .7
				this.Hint = "Hide Main Menu preview area"
				config.Current.Advanced.Matching.Disabled.Previews = false
			}

			detect.Images(hidden)
		},
	}

	ui.navButtons.hideTop = &button.Widget{
		Text:            "⇈",
		Font:            g.nav.NishikiTeki(),
		Pressed:         nrgba.Gray,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Hint:            "Show Main Menu configuration area",
		OnHoverHint:     g.nav.Tip,
		Click: func(this *button.Widget) {
			defer this.Deactivate()

			if this.Text == "⇈" {
				this.Text = "⇊"
				this.Hint = "Show Main Menu configuration area"
			} else {
				this.Text = "⇈"
				this.Hint = "Hide Main Menu configuration area"
			}
		},
	}

	ui.navButtons.alwaysOnTop = &button.Widget{
		Text:            "📌",
		Font:            g.nav.NishikiTeki(),
		Hint:            "Show UniteHUD Overlay HUD above all windows",
		OnHoverHint:     g.nav.Tip,
		Released:        nrgba.Transparent80,
		Pressed:         nrgba.Lilac,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			if this.Radio {
				this.Hint = "Show UniteHUD Overlay HUD above all windows"
				this.Text = "📌"
				this.Radio = false
				wapi.SetWindowNotAlwaysOnTop(g.HWND)
			} else {
				this.Hint = "Hide UniteHUD Overlay HUD under active windows"
				this.Text = "📌×"
				this.Radio = true
				wapi.SetWindowAlwaysOnTop(g.HWND)
			}
		},
	}

	ui.navButtons.screenshot = &button.Widget{
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

			w := wapi.Window(g.HWND)

			r, err := w.Dimensions()
			if err != nil {
				notify.Error("[UI] Failed to determine screenshot dimensions (%v)", err)
				g.ToastErrorf("Failed to capture screenshot (%v)", err)
				return
			}

			img, err := w.Capture(r, 1)
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

	return ui
}

var (
	// framer tracks one-shot frame callbacks by frame index.
	framer = map[int]struct {
		count int
		done  bool
	}{}
)

// onFrame handles the event callback.
func (ui *main) onFrame(frame int, fn func(*GUI), g *GUI) {
	f := framer[frame]
	if f.done || f.count == 4096 { // Frame limitation.
		return
	}

	if f.count++; frame == f.count {
		defer fn(g)
		f.done = true
	}

	framer[frame] = f
}

// onFrame1 handles the event callback.
func (ui *main) onFrame1(g *GUI) {
	max := g.dimensions.max()
	g.dimensions.size = image.Pt(1100, 700)

	g.window.Option(
		app.Size(
			unit.Dp(g.dimensions.size.X),
			unit.Dp(g.dimensions.size.Y),
		),
		app.MinSize(
			unit.Dp(g.dimensions.size.X),
			unit.Dp(g.dimensions.size.Y),
		),
		app.MaxSize(
			unit.Dp(max.X),
			unit.Dp(max.Y),
		),
	)

	// g.window.Perform(system.ActionCenter)
	// g.window.Perform(system.ActionUnmaximize)

	go func() {
		for ; ; time.Sleep(time.Second / 3) {
			if tray.StartStopEvent() {
				ui.navButtons.startstop.Click(ui.navButtons.startstop)
			}
		}
	}()

	// go wapi.SetWindowLongPtrA.Call(g.HWND, wapi.GetWindowLongFlags.Style, uintptr(wapi.WindowStyles.Overlapped))
}

// onFrame2 handles the event callback.
func (ui *main) onFrame2(g *GUI) {
	if config.IsNew() {
		g.ToastNewsletter(
			exe.Title,
			bulletin{
				Title: fmt.Sprintf("Welcome to %s!", exe.TitleAndVersion),
				Topics: []struct {
					Subtitle string
					Points   []string
				}{
					{
						Subtitle: "How to Get Started",
						Points: []string{
							"Configure video capture settings by selecting ⚙ from the title bar",
							"Add the overlay HUD to OBS by selecting obs from the title bar",
						},
					},
					{
						Subtitle: "Community",
						Points: []string{
							"Have questions? Join the UniteHUD Discord!",
							"Follow @UniteHUD on X to stay up to date with major highlights!",
							"Track the open source project on github.com",
						},
					},
					{
						Subtitle: "Customize!",
						Points: []string{
							"Adjust the color scheme of UniteHUD from the Advanced Settings menu",
							"Modify the overlay HUD and customize event animations for your stream",
						},
					},
				},
			},
			toastOnClose(nil),
		)
	}
}

// onFrame3 handles the event callback.
func (ui *main) onFrame3(g *GUI) {
	if !discord.Connected() && config.Current.Remember.Discord == config.DiscordRememberStandby {
		was := config.Current.Advanced.Discord.Disabled

		g.ToastYesNoRemember(
			exe.Title,
			"<ini:t:connect_discord>",
			"<ini:t:connect_discord_remember>",
			toastOnYes(func() {
				config.Current.Advanced.Discord.Disabled = false
			}),
			toastOnNo(func() {
				config.Current.Advanced.Discord.Disabled = true
			}),
			toastOnClose(func() {
				discord.Asked = true

				if was == config.Current.Advanced.Discord.Disabled {
					config.Current.Advanced.Discord.Disabled = true
				}

				err := config.Current.Save()
				if err != nil {
					notify.Error("[UI] <ini:f:save> UniteHUD configuration (%v)", err)
					return
				}
			}),
			toastOnRemember(func(b bool) {
				if !b {
					return
				}

				config.Current.Remember.Discord = config.DiscordRememberDisabled
				if config.Current.Advanced.Discord.Disabled {
					config.Current.Remember.Discord = config.DiscordRememberEnabled
				}
			}),
		)
	}
}
