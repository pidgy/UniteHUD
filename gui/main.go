package gui

import (
	"fmt"
	"image"
	"os"
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

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/desktop"
	"github.com/pidgy/unitehud/desktop/clicked"
	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/decorate"
	"github.com/pidgy/unitehud/gui/visual/screen"
	"github.com/pidgy/unitehud/gui/visual/spinner"
	"github.com/pidgy/unitehud/gui/visual/split"
	"github.com/pidgy/unitehud/gui/visual/textblock"
	"github.com/pidgy/unitehud/history"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/nrgba"
	"github.com/pidgy/unitehud/save"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/state"
	"github.com/pidgy/unitehud/stats"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/video/device"
	"github.com/pidgy/unitehud/video/monitor"
	"github.com/pidgy/unitehud/video/window"
	"github.com/skratchdot/open-golang/open"
)

var once = true

func (g *GUI) main() {
	if once {
		once = false
		g.window.Option(
			app.Title("UniteHUD"),
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

	split := &split.Vertical{
		Ratio: .70,
	}

	spinRun := spinner.Running()
	defer spinRun.Stop()

	spinStop := spinner.Stopped()
	defer spinStop.Stop()

	stopButton := &button.Widget{
		Text:            "Stop",
		Font:            g.header.Collection.Calibri(),
		OnHoverHint:     func() { g.header.Tip("Stop capturing events") },
		Disabled:        true,
		Released:        nrgba.Disabled,
		BorderWidth:     unit.Sp(1.5),
		Size:            image.Pt(60, 25),
		TextInsetBottom: -2,
	}

	startButton := &button.Widget{
		Text:            "Start",
		Font:            g.header.Collection.Calibri(),
		OnHoverHint:     func() { g.header.Tip("Start capturing events") },
		Released:        nrgba.PastelGreen.Alpha(150),
		Pressed:         nrgba.Transparent80,
		BorderWidth:     unit.Sp(1.5),
		Size:            stopButton.Size,
		TextInsetBottom: stopButton.TextInsetBottom,
		Click: func(this *button.Widget) {
			g.Preview = false

			stopButton.Deactivate()
			stopButton.Disabled = false
			stopButton.Released = nrgba.PastelRed.Alpha(150)

			this.Deactivate()
			this.Disabled = true
			this.Released = nrgba.Disabled

			g.Actions <- Config
			g.Running = true
		},
	}

	stopButton.Click = func(this *button.Widget) {
		this.Deactivate()
		this.Disabled = true
		this.Released = nrgba.Disabled

		startButton.Deactivate()
		startButton.Disabled = false
		startButton.Released = nrgba.PastelGreen.Alpha(150)

		g.Actions <- Stop
		g.Running = false
		g.Preview = true
	}

	notifyFeedTextBlock, err := textblock.New(g.header.Collection.Cascadia())
	if err != nil {
		notifyFeedTextBlock = &textblock.Widget{}
		notify.Error("Failed to load font: (%v)", err)
	}

	defer g.header.Remove(g.header.Add(&button.Widget{
		Text:        "📺",
		Font:        g.header.Collection.NishikiTeki(),
		OnHoverHint: func() { g.header.Tip("Configure capture settings") },
		Released:    nrgba.PurpleBlue,
		TextSize:    unit.Sp(16),

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			if !stopButton.Disabled {
				stopButton.Click(stopButton)
			}

			g.next(is.Projecting)
		},
	}))

	defer g.header.Remove(g.header.Add(&button.Widget{
		Text:        "¼",
		Font:        g.header.Collection.NishikiTeki(),
		OnHoverHint: func() { g.header.Tip("View capture statistics") },
		Released:    nrgba.Pinkity,
		TextSize:    unit.Sp(14),

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
	}))

	defer g.header.Remove(g.header.Add(&button.Widget{
		Text:        "🗠",
		TextSize:    unit.Sp(16),
		Font:        g.header.Collection.NishikiTeki(),
		OnHoverHint: func() { g.header.Tip("View win/loss history") },
		Released:    nrgba.Seafoam,

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			history.Dump()
		},
	}))

	defer g.header.Remove(g.header.Add(&button.Widget{
		Text:        "obs",
		Font:        g.header.Collection.NishikiTeki(),
		OnHoverHint: func() { g.header.Tip("Open OBS client folder") },
		Released:    nrgba.Purple,
		TextSize:    unit.Sp(12),

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			drag := "Drag \"UniteHUD Client\" into any OBS scene."
			if config.Current.Profile == config.ProfileBroadcaster {
				drag = "Drag \"UniteHUD Broadcaster\" into any OBS scene."
			}

			g.ToastOK("Overlay", drag,
				OnToastOK(func() {
					exe, err := os.Executable()
					if err != nil {
						notify.Error("Failed to open www/ directory: %v", err)
						return
					}

					dir := filepath.Dir(exe)

					err = open.Run(dir + "/www")
					if err != nil {
						notify.Error("Failed to open www/ directory: %v", err)
						return
					}
				}),
			)
		},
	}))

	defer g.header.Remove(g.header.Add(&button.Widget{
		Text:        "🗘",
		Font:        g.header.Collection.NishikiTeki(),
		OnHoverHint: func() { g.header.Tip("Clear event history") },
		Released:    nrgba.Orange,
		TextSize:    unit.Sp(14),

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			notify.CLS()
			notify.System("Cleared")
		},
	}))

	defer g.header.Remove(g.header.Add(&button.Widget{
		Text:        "⚶",
		Font:        g.header.Collection.NishikiTeki(),
		OnHoverHint: func() { g.header.Tip("Toggle resource saver") },
		Released:    nrgba.ForestGreen,
		Pressed:     nrgba.PaleRed.Alpha(50),
		TextSize:    unit.Sp(16),

		Click: func(this *button.Widget) {
			g.performance.eco = !g.performance.eco

			this.Activate()
			if g.performance.eco {
				this.Deactivate()
			}

			if g.performance.eco {
				notify.System("Resource saver has been enabled")
			} else {
				notify.System("Resource saver has been disabled")
			}
		},
	}))

	defer g.header.Remove(g.header.Add(&button.Widget{
		Text:        "🗁",
		Font:        g.header.Collection.NishikiTeki(),
		OnHoverHint: func() { g.header.Tip("Open log directory") },
		Released:    nrgba.PastelBabyBlue,
		TextSize:    unit.Sp(16),

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			save.Logs()

			err := save.Open()
			if err != nil {
				notify.Error("Failed to open \"%s\" (%v)", save.Directory, err)
			}
		},
	}))

	defer g.header.Remove(g.header.Add(&button.Widget{
		Text:        "🎬",
		Font:        g.header.Collection.NishikiTeki(),
		OnHoverHint: func() { g.header.Tip("Record matched events") },
		Released:    nrgba.Pinkity.Alpha(100),
		TextSize:    16,

		Click: func(this *button.Widget) {
			title := "Record"
			description := "Record and save captured events on your computer?"
			yes := func() {
				defer save.Logs()

				config.Current.Record = true
				notify.System("Recording captured events in %s", save.Directory)
				this.Text = "■"
				this.TextSize = 15
			}

			if config.Current.Record {
				title = "Stop"
				description = "Stop recording captured events?"
				yes = func() {
					defer save.Logs()

					notify.System("Saved captured events in %s", save.Directory)
					this.Text = "🎬"
					this.TextSize = 16
					this.TextInsetBottom = 0

					err := save.Open()
					if err != nil {
						notify.Error("Failed to open \"%s\" (%v)", save.Directory, err)
					}
				}
			}

			g.ToastYesNo(title, description, OnToastYes(yes), OnToastNo(this.Deactivate))
		},
	}))

	projectorWindowButton := &button.ImageWidget{
		HintEvent: func() { g.header.Tip("Open projector window") },

		Widget: &screen.Widget{
			Border:      true,
			BorderColor: nrgba.Transparent,
		},
		Click: func(this *button.ImageWidget) {
			if !stopButton.Disabled {
				stopButton.Click(stopButton)
			}

			g.next(is.Projecting)
		},
	}

	warningLabel := material.Label(g.header.Collection.NotoSans().Theme, unit.Sp(11), "⚠ CPU")
	warningLabel.Color = nrgba.Yellow.Alpha(200).Color()
	warningLabel.Font.Weight = 0

	profileHeader := material.Caption(g.header.Collection.Calibri().Theme, "")
	profileHeader.Color = nrgba.DreamyPurple.Color()
	profileHeader.Alignment = text.Middle
	profileHeader.Font.Weight = font.ExtraBold
	profileHeader.TextSize = unit.Sp(14)

	windowHeader := material.Caption(g.header.Collection.Calibri().Theme, "")
	windowHeader.Color = nrgba.DarkSeafoam.Color()
	windowHeader.Alignment = text.Middle
	windowHeader.Font.Weight = font.ExtraBold
	windowHeader.TextSize = unit.Sp(14)

	cpuLabel := material.H5(g.header.Collection.Calibri().Theme, "")
	cpuLabel.Alignment = text.Middle
	cpuLabel.TextSize = unit.Sp(14)

	cpuGraph := material.H5(g.header.Collection.Cascadia().Theme, "")
	cpuGraph.Color = nrgba.Gray.Color()
	cpuGraph.TextSize = unit.Sp(9)

	ramLabel := material.H5(g.header.Collection.Calibri().Theme, "")
	ramLabel.Alignment = text.Middle
	ramLabel.TextSize = unit.Sp(14)

	ramGraph := material.H5(g.header.Collection.Cascadia().Theme, "")
	ramGraph.Color = nrgba.Gray.Color()
	ramGraph.TextSize = unit.Sp(9)

	holdingLabel := material.H5(g.header.Collection.Calibri().Theme, "")
	holdingLabel.Color = nrgba.Gold.Color()
	holdingLabel.Alignment = text.Middle
	holdingLabel.TextSize = unit.Sp(14)

	connectedClientsLabel := material.H5(g.header.Collection.Calibri().Theme, "")
	connectedClientsLabel.Alignment = text.Middle
	connectedClientsLabel.TextSize = unit.Sp(14)

	purpleScoreScreen := &screen.Widget{
		Border:      true,
		BorderColor: team.Purple.NRGBA,
		Image:       notify.PurpleScore,
	}

	orangeScoreScreen := &screen.Widget{
		Border:      true,
		BorderColor: team.Orange.NRGBA,
		Image:       notify.OrangeScore,
	}

	energyScoreScreen := &screen.Widget{
		Border:      true,
		BorderColor: team.Energy.NRGBA,
		Image:       notify.Energy,
	}

	timeScreen := &screen.Widget{
		Border:      true,
		BorderColor: team.Time.NRGBA,
		Image:       notify.Time,
	}

	dbgLabel := material.H5(g.header.Collection.Calibri().Theme, "DBG")
	dbgLabel.Alignment = text.Middle
	dbgLabel.TextSize = unit.Sp(14)
	dbgLabel.Color = nrgba.SeaBlue.Color()

	symbolLabel := material.H5(g.header.Collection.Calibri().Theme, "")
	symbolLabel.Alignment = text.Middle
	symbolLabel.TextSize = unit.Sp(16)
	symbolLabel.Font.Weight = font.ExtraBold
	symbolLabel.Color = nrgba.Slate.Color()

	acronymLabel := material.H5(g.header.Collection.Calibri().Theme, "IDLE")
	acronymLabel.Alignment = text.Middle
	acronymLabel.TextSize = unit.Sp(14)
	acronymLabel.Color = nrgba.Slate.Color()

	fpsLabel := material.H5(g.header.Collection.Calibri().Theme, "0 FPS")
	fpsLabel.Alignment = text.Middle
	fpsLabel.TextSize = unit.Sp(14)

	purpleScoreLabel := material.H5(g.header.Collection.Calibri().Theme, "0")
	purpleScoreLabel.Color = team.Purple.NRGBA.Color()
	purpleScoreLabel.Alignment = text.Middle
	purpleScoreLabel.TextSize = unit.Sp(14)

	orangeScoreLabel := material.H5(g.header.Collection.Calibri().Theme, "0")
	orangeScoreLabel.Color = team.Orange.NRGBA.Color()
	orangeScoreLabel.Alignment = text.Middle
	orangeScoreLabel.TextSize = unit.Sp(14)

	selfScoreLabel := material.H5(g.header.Collection.Calibri().Theme, "0")
	selfScoreLabel.Color = team.Self.NRGBA.Color()
	selfScoreLabel.Alignment = text.Middle
	selfScoreLabel.TextSize = unit.Sp(14)

	clockLabel := material.H5(g.header.Collection.Calibri().Theme, "00:00")
	clockLabel.Alignment = text.Middle
	clockLabel.TextSize = unit.Sp(14)

	regielekiLabels, regielekiUnderlineLabels := []material.LabelStyle{
		material.H5(g.header.Collection.Calibri().Theme, "E"),
		material.H5(g.header.Collection.Calibri().Theme, "E"),
		material.H5(g.header.Collection.Calibri().Theme, "E"),
	}, []material.LabelStyle{
		material.H5(g.header.Collection.Calibri().Theme, "_"),
		material.H5(g.header.Collection.Calibri().Theme, "_"),
		material.H5(g.header.Collection.Calibri().Theme, "_"),
	}

	for i := range regielekiLabels {
		regielekiLabels[i].Color = team.None.Color()
		regielekiLabels[i].Alignment = text.Middle
		regielekiLabels[i].TextSize = unit.Sp(14)

		regielekiUnderlineLabels[i].Color = team.None.Color()
		regielekiUnderlineLabels[i].Alignment = regielekiLabels[i].Alignment
		regielekiUnderlineLabels[i].TextSize = unit.Sp(18)
		regielekiUnderlineLabels[i].Font.Weight = font.Bold
	}

	regiBottomLabels, regiBottomUnderlineLabels := []material.LabelStyle{
		material.H5(g.header.Collection.Calibri().Theme, "R"),
		material.H5(g.header.Collection.Calibri().Theme, "R"),
		material.H5(g.header.Collection.Calibri().Theme, "R"),
	}, []material.LabelStyle{
		material.H5(g.header.Collection.Calibri().Theme, "_"),
		material.H5(g.header.Collection.Calibri().Theme, "_"),
		material.H5(g.header.Collection.Calibri().Theme, "_"),
	}

	for i := range regiBottomLabels {
		regiBottomLabels[i].Color = team.None.Color()
		regiBottomLabels[i].Alignment = text.Middle
		regiBottomLabels[i].TextSize = unit.Sp(14)

		regiBottomUnderlineLabels[i].Color = regiBottomLabels[i].Color
		regiBottomUnderlineLabels[i].Alignment = regiBottomLabels[i].Alignment
		regiBottomUnderlineLabels[i].TextSize = unit.Sp(18)
		regiBottomUnderlineLabels[i].Font.Weight = font.Bold
	}

	uptimeLabel := material.H5(g.header.Collection.Calibri().Theme, g.performance.uptime)
	uptimeLabel.Color = nrgba.DreamyPurple.Color()
	uptimeLabel.Alignment = text.Middle
	uptimeLabel.TextSize = unit.Sp(14)

	g.window.Perform(system.ActionRaise)
	if !g.firstOpen {
		g.firstOpen = true
		g.window.Perform(system.ActionCenter)
	}

	var ops op.Ops

	for g.is == is.MainMenu {
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

		if g.performance.eco && state.Idle() > time.Minute*30 && !stopButton.Disabled {
			desktop.Notification("Eco Mode").
				Says("No matches detected for 30 minutes, stopping to save resources").
				When(clicked.OpenUniteHUD).
				Send()

			stopButton.Click(stopButton)
		}

		switch event := (<-g.window.Events()).(type) {
		case app.ConfigEvent:
		case system.DestroyEvent:
			g.next(is.Closing)
			return
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, event)
			op.InvalidateOp{At: gtx.Now}.Add(gtx.Ops)

			g.dimensions.size = event.Size

			decorate.Background(gtx)
			decorate.Label(&cpuLabel, g.performance.cpu)
			decorate.Label(&cpuGraph, stats.CPUGraph())
			decorate.Label(&ramLabel, g.performance.ram)
			decorate.Label(&ramGraph, stats.RAMGraph())
			decorate.Label(&holdingLabel, holdingLabel.Text)
			decorate.ForegroundAlt(&cpuGraph.Color)
			decorate.ForegroundAlt(&ramGraph.Color)

			g.header.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return split.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
							{
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
									warningLabel.Text = fmt.Sprintf("⚠ CPU (%s)", strings.Join(warnings, ","))

									layout.Inset{
										Left: unit.Dp(4),
										Top:  unit.Dp(1),
									}.Layout(gtx, warningLabel.Layout)
								}

								if len(nonwarnings) > 0 {
									warningLabel.Text = fmt.Sprintf("⚠ CPU (%s)", strings.Join(nonwarnings, ","))
									warningLabel.Color = nrgba.PastelGreen.Color()
									layout.Inset{
										Left: unit.Dp(4),
										Top:  unit.Dp(1),
									}.Layout(gtx, warningLabel.Layout)
								}

								profileHeader.Text = fmt.Sprintf("%s // %s", strings.Title(config.Current.Profile), strings.Title(config.Current.Platform))
								layout.Inset{
									Left: unit.Dp(4),
									Top:  unit.Dp(35),
								}.Layout(gtx, profileHeader.Layout)

								switch {
								case device.IsActive():
									if windowHeader.Text == "" || windowHeader.Text == config.Current.Window {
										windowHeader.Text = device.Name(config.Current.VideoCaptureDevice)
									}
								case window.IsOpen():
									windowHeader.Text = config.Current.Window
								case monitor.IsDisplay():
									windowHeader.Text = config.Current.Window
								}
								if config.Current.LostWindow != "" {
									windowHeader.Text = config.Current.LostWindow
									windowHeader.Color = nrgba.PaleRed.Color()
								}
								layout.Inset{
									Left: unit.Dp(4),
									Top:  unit.Dp(50),
								}.Layout(gtx, windowHeader.Layout)
							}
							{
								layout.Inset{
									Top:  unit.Dp(28),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 408)),
								}.Layout(gtx, cpuLabel.Layout)

								layout.Inset{
									Top:  unit.Dp(1),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 450)),
								}.Layout(gtx, cpuGraph.Layout)

								layout.Inset{
									Top:  unit.Dp(28),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 248)),
								}.Layout(gtx, ramLabel.Layout)

								layout.Inset{
									Top:  unit.Dp(1),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 300)),
								}.Layout(gtx, ramGraph.Layout)

								holdingLabel.Text = fmt.Sprintf("%02d/%02d", team.Energy.Holding, team.Energy.HoldingMax)

								layout.Inset{
									Top:  unit.Dp(50),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 35)),
								}.Layout(gtx, holdingLabel.Layout)
							}
							{
								clients := server.Clients()
								if clients > 0 {
									connectedClientsLabel.Color = nrgba.Seafoam.Color()
								} else {
									connectedClientsLabel.Color = nrgba.PaleRed.Color()
								}

								connectedClientsLabel.Text = fmt.Sprintf("OBS %d", clients)
								layout.Inset{
									Top:  unit.Dp(34),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 135)),
								}.Layout(gtx, connectedClientsLabel.Layout)
							}
							{
								down := float32(1)
								left := 1

								if g.Running {
									symbolLabel.Text = spinRun.Next()
									symbolLabel.Color = nrgba.Green.Color()

									acronymLabel.Text = "RUN"
									acronymLabel.Color = nrgba.Green.Color()

									left = 1
									down = .5
								} else {
									acronymLabel.Color = nrgba.Slate.Color()
									symbolLabel.Color = nrgba.Slate.Color()
									acronymLabel.Text = "IDLE"
								}

								symbolLabel.Text = spinStop.Next()
								layout.Inset{
									Top:  unit.Dp(48 + down),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 145 - left)),
								}.Layout(gtx, symbolLabel.Layout)

								layout.Inset{
									Top:  unit.Dp(50),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 135)),
								}.Layout(gtx, acronymLabel.Layout)

								if global.DebugMode {
									layout.Inset{
										Top:  unit.Dp(18),
										Left: unit.Dp(float32(gtx.Constraints.Max.X - 135)),
									}.Layout(gtx, dbgLabel.Layout)
								}

								fpsLabel.Color = nrgba.Green.Color()

								// switch {
								// case g.fps.FPS() == g.fps.max:
								// 	fpsLabel.Color = nrgba.Green.Color()
								// case g.fps.frames < g.fps.max/3:
								// 	fpsLabel.Color = nrgba.Red.Color()
								// case g.fps.frames < g.fps.max/2:
								// 	fpsLabel.Color = nrgba.Orange.Color()
								// case g.fps.frames < g.fps.max:
								// 	fpsLabel.Color = nrgba.Yellow.Color()
								// }

								fpsLabel.Text = fmt.Sprintf("%s FPS", g.fps)
								layout.Inset{
									Top:  unit.Dp(2),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 135)),
								}.Layout(gtx, fpsLabel.Layout)
							}
							{
								o, p, s := server.Scores()

								purpleScoreLabel.Text = fmt.Sprintf("%d", p)
								layout.Inset{
									Top:  unit.Dp(2),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 35)),
								}.Layout(gtx, purpleScoreLabel.Layout)

								orangeScoreLabel.Text = fmt.Sprintf("%d", o)
								layout.Inset{
									Top:  unit.Dp(18),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 35)),
								}.Layout(gtx, orangeScoreLabel.Layout)

								selfScoreLabel.Text = fmt.Sprintf("%d", s)
								layout.Inset{
									Top:  unit.Dp(34),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 35)),
								}.Layout(gtx, selfScoreLabel.Layout)
							}
							{
								decorate.Label(&clockLabel, server.Clock())
								layout.Inset{
									Top:  unit.Dp(2),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 90)),
								}.Layout(gtx, clockLabel.Layout)
							}
							{
								for i, t := range server.Regielekis() {
									regielekiLabels[i].Color = team.None.Color()
									if t != team.None.Name {
										regielekiLabels[i].Color = nrgba.Regieleki.Color()
									}

									regielekiUnderlineLabels[i].Color = team.Color(t).Color()
								}

								for i := range regielekiLabels {
									layout.Inset{
										Top:  unit.Dp(18),
										Left: unit.Dp(float32(gtx.Constraints.Max.X-90) + float32(i*12)),
									}.Layout(gtx, regielekiLabels[i].Layout)

									layout.Inset{
										Top:  unit.Dp(15),
										Left: unit.Dp(float32(gtx.Constraints.Max.X-90) + float32(i*12)),
									}.Layout(gtx, regielekiUnderlineLabels[i].Layout)
								}
							}
							{
								b := server.Bottom()
								for i := range regiBottomLabels {
									regiBottomLabels[i].Color = team.None.Color()
									regiBottomLabels[i].Text = "R"
									regiBottomUnderlineLabels[i].Color = regiBottomLabels[i].Color

									if i < len(b) {
										t := b[i]
										regiBottomLabels[i].Text = strings.ToUpper(string(t.Name[4]))
										regiBottomLabels[i].Color = nrgba.Objective(t.Name).Color()
										regiBottomUnderlineLabels[i].Color = team.Color(t.Team).Color()
									}

									layout.Inset{
										Top:  unit.Dp(34),
										Left: unit.Dp(float32(gtx.Constraints.Max.X-90) + float32(i*12)),
									}.Layout(gtx, regiBottomLabels[i].Layout)

									layout.Inset{
										Top:  unit.Dp(31),
										Left: unit.Dp(float32(gtx.Constraints.Max.X-90) + float32(i*12)),
									}.Layout(gtx, regiBottomUnderlineLabels[i].Layout)
								}
							}
							{
								uptimeLabel.Text = g.performance.uptime

								layout.Inset{
									Top:  unit.Dp(50),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 90)),
								}.Layout(gtx, uptimeLabel.Layout)
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
						return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
							// Right-side criteria.
							{
								layout.Inset{
									Top: unit.Dp(float32(gtx.Constraints.Max.Y - int(float32(startButton.Size.Y)*1.5))),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Flex{Axis: layout.Horizontal}.Layout(
												gtx,
												layout.Flexed(.5, layout.Spacer{Width: unit.Dp(25)}.Layout),
												layout.Rigid(startButton.Layout),
												layout.Flexed(.5, layout.Spacer{Width: unit.Dp(25)}.Layout),
												layout.Rigid(stopButton.Layout),
												layout.Flexed(.5, layout.Spacer{Width: unit.Dp(25)}.Layout),
											)
										}),
									)
								})
							}

							// Event images.
							{

								layout.Inset{
									Top: unit.Dp(60),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									projectorWindowButton.SetImage(notify.Preview)
									return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return projectorWindowButton.Layout(g.header.Collection.Cascadia().Theme, gtx)
									})
								})

								layout.Inset{
									Top:  unit.Dp(147),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 150)),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.UniformInset(unit.Dp(5)).Layout(gtx,
										purpleScoreScreen.Layout,
									)
								})

								layout.Inset{
									Top:  unit.Dp(209),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 150)),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.UniformInset(unit.Dp(5)).Layout(gtx,
										orangeScoreScreen.Layout,
									)
								})

								layout.Inset{
									Top:  unit.Dp(271),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 68)),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.UniformInset(unit.Dp(5)).Layout(gtx,
										energyScoreScreen.Layout,
									)
								})

								layout.Inset{
									Top:  unit.Dp(333),
									Left: unit.Dp(float32(gtx.Constraints.Max.X - 68)),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.UniformInset(unit.Dp(5)).Layout(gtx,
										timeScreen.Layout,
									)
								})
							}

							return layout.Dimensions{Size: gtx.Constraints.Max}
						},
						)
					},
				)
			})

			g.frame(gtx, event)
		default:
			notify.Missed(event, "Main")
		}
	}
}
