package ui

import (
	"image"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/fonts"
	"github.com/pidgy/unitehud/core/global"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/nrgba"
	"github.com/pidgy/unitehud/gui/visual"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/colorpicker"
	"github.com/pidgy/unitehud/gui/visual/decorate"
	"github.com/pidgy/unitehud/gui/visual/dropdown"
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/system/desktop"
	"github.com/pidgy/unitehud/system/desktop/clicked"
	"github.com/pidgy/unitehud/system/discord"
)

type section struct {
	h1                 bool
	title, description material.LabelStyle
	warning, widget    visual.Widgeter

	extras []visual.Widgeter
}

type settings struct {
	hwnd uintptr

	bar *title.Widget

	windows struct {
		parent  *GUI
		current *app.Window
	}

	state struct {
		open bool
	}

	dimensions struct {
		width,
		height int

		resize bool
	}

	list material.ListStyle

	sections struct {
		header,
		discord,
		notifications,
		factor,
		theme,
		themes *section
	}
}

func (g *GUI) settings(onclose func()) *settings {
	ui := g.settingsUI()

	go func() {
		defer onclose()

		ui.state.open = true
		defer func() {
			ui.state.open = false
		}()

		ui.windows.current.Perform(system.ActionRaise)

		ui.windows.parent.setInsetRight(ui.dimensions.width)
		defer ui.windows.parent.unsetInsetRight(ui.dimensions.width)

		sections := []*section{
			ui.sections.header,
			ui.sections.discord,
			ui.sections.notifications,
			ui.sections.factor,
			ui.sections.theme,
			ui.sections.themes,
		}

		var ops op.Ops

		for {
			switch event := ui.windows.current.NextEvent().(type) {
			case system.DestroyEvent:
				return
			case app.ViewEvent:
				ui.hwnd = event.HWND
				ui.windows.parent.attachWindowRight(ui.hwnd, ui.dimensions.width)
			case system.FrameEvent:
				if !ui.state.open {
					go ui.windows.current.Perform(system.ActionClose)
				}

				if ui.dimensions.resize {
					ui.dimensions.resize = false
					ui.windows.parent.attachWindowRight(ui.hwnd, ui.dimensions.width)
				}

				gtx := layout.NewContext(&ops, event)

				ui.bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					decorate.List(&ui.list)
					decorate.Scrollbar(&ui.list.ScrollbarStyle)

					return ui.list.Layout(gtx, len(sections), func(gtx layout.Context, index int) layout.Dimensions {
						return sections[index].section(gtx)
					})
				})

				ui.windows.current.Invalidate()
				event.Frame(gtx.Ops)
			default:
				notify.Missed(event, "Settings")
			}
		}
	}()

	return ui
}

func (g *GUI) settingsUI() *settings {
	ui := &settings{}

	ui.dimensions.width = 350
	ui.dimensions.height = 700

	ui.windows.parent = g

	ui.bar = title.New(
		"Settings",
		fonts.NewCollection(),
		nil,
		nil,
		func() { ui.windows.current.Perform(system.ActionClose) },
	)
	ui.bar.NoTip = true
	ui.bar.NoDrag = true

	ui.windows.current = app.NewWindow(
		app.Title("Settings"),
		app.Size(unit.Dp(ui.dimensions.width), unit.Dp(ui.dimensions.height)),
		app.MinSize(unit.Dp(ui.dimensions.width), unit.Dp(ui.dimensions.height)),
		app.MaxSize(unit.Dp(ui.dimensions.width), unit.Dp(ui.windows.parent.dimensions.max.Y)),
		app.Decorated(false),
	)

	ui.list = material.List(
		ui.bar.Collection.Calibri().Theme,
		&widget.List{
			Scrollbar: widget.Scrollbar{},
			List: layout.List{
				Axis:      layout.Vertical,
				Alignment: layout.Start,
			},
		},
	)

	ui.sections.header = &section{
		h1:          true,
		description: material.H6(ui.bar.Collection.Calibri().Theme, "Advanced Settings"),
	}

	ui.sections.discord = &section{
		title:       material.Label(ui.bar.Collection.Calibri().Theme, 14, "🎮 Discord Activity"),
		description: material.Caption(ui.bar.Collection.Calibri().Theme, "Enable/Disable Discord activity updates"),
		widget: &button.Widget{
			Text:            "Enabled",
			Pressed:         nrgba.Transparent80,
			Released:        nrgba.Discord,
			TextSize:        unit.Sp(14),
			TextInsetBottom: unit.Dp(-2),
			Size:            image.Pt(80, 20),
			Font:            ui.bar.Collection.Calibri(),

			Click: func(this *button.Widget) {
				config.Current.Advanced.Discord.Disabled = !config.Current.Advanced.Discord.Disabled

				if config.Current.Advanced.Discord.Disabled {
					this.Released = nrgba.PastelRed
					this.Text = "Disabled"
					discord.Disconnect()
				} else {
					this.Released = nrgba.Discord
					this.Text = "Enabled"
				}
			},
		},
	}
	discordWarning := material.Label(
		ui.bar.Collection.Calibri().Theme,
		unit.Sp(12),
		"🔌 Activity Privacy settings in Discord can prevent this feature from working",
	)
	discordWarning.Color = nrgba.PastelRed.Alpha(127).Color()
	discordWarning.Font.Weight = 0
	ui.sections.discord.warning = discordWarning

	if config.Current.Advanced.Discord.Disabled {
		ui.sections.discord.widget.(*button.Widget).Released = nrgba.PastelRed
		ui.sections.discord.widget.(*button.Widget).Text = "Disabled"
	}

	ui.sections.notifications = &section{
		title:       material.Label(ui.bar.Collection.Calibri().Theme, 14, "🔔 Desktop Notifications"),
		description: material.Caption(ui.bar.Collection.Calibri().Theme, "Adjust desktop notifications for UniteHUD"),

		extras: []visual.Widgeter{
			&button.Widget{
				Text:            "🔔 Test",
				Pressed:         nrgba.Transparent80,
				Released:        nrgba.PastelOrange.Alpha(150),
				TextSize:        unit.Sp(12),
				TextInsetBottom: unit.Dp(-2),
				Size:            image.Pt(80, 20),
				Font:            ui.bar.Collection.Calibri(),

				Click: func(this *button.Widget) {
					was := config.Current.Advanced.Notifications.Disabled.All
					defer func() {
						config.Current.Advanced.Notifications.Disabled.All = was
					}()

					config.Current.Advanced.Notifications.Disabled.All = false

					desktop.Notification(global.Title).
						Says("Testing 1..2..3").
						When(clicked.VisitWebsite).
						Send()
				},
			},
		},
	}
	notificationsWarning := material.Label(
		ui.bar.Collection.Calibri().Theme,
		unit.Sp(12),
		"📌 Some settings are automatically applied by your Operating System",
	)
	notificationsWarning.Color = nrgba.PastelRed.Alpha(127).Color()
	notificationsWarning.Font.Weight = 0
	ui.sections.notifications.warning = notificationsWarning
	ui.sections.notifications.widget = &dropdown.Widget{
		Theme:    ui.bar.Collection.NotoSans().Theme,
		TextSize: 12,
		Items: []*dropdown.Item{
			{
				Text: "Disabled",
				Checked: widget.Bool{
					Value: config.Current.Advanced.Notifications.Disabled.All,
				},
				Callback: func(this *dropdown.Item) {
					config.Current.Advanced.Notifications.Disabled.All = this.Checked.Value
				},
			},
			{
				Text: "Muted",
				Checked: widget.Bool{
					Value: config.Current.Advanced.Notifications.Muted,
				},
				Callback: func(this *dropdown.Item) {
					config.Current.Advanced.Notifications.Muted = this.Checked.Value
				},
			},
			{
				Text: "Match Starting",
				Checked: widget.Bool{
					Value: !config.Current.Advanced.Notifications.Disabled.MatchStarting,
				},
				Callback: func(this *dropdown.Item) {
					this.Checked.Value = !this.Checked.Value
					config.Current.Advanced.Notifications.Disabled.MatchStarting = this.Checked.Value
				},
			},
			{
				Text: "Match Stopped",
				Checked: widget.Bool{
					Value: !config.Current.Advanced.Notifications.Disabled.MatchStopped,
				},
				Callback: func(this *dropdown.Item) {
					config.Current.Advanced.Notifications.Disabled.MatchStopped = this.Checked.Value
				},
			},
			{
				Text: "Updates",
				Checked: widget.Bool{
					Value: !config.Current.Advanced.Notifications.Disabled.Updates,
				},
				Callback: func(this *dropdown.Item) {
					config.Current.Advanced.Notifications.Disabled.Updates = this.Checked.Value
				},
			},
		},
		Callback: func(item *dropdown.Item, this *dropdown.Widget) bool {
			if item.Text != "Disabled" {
				return true
			}

			for _, i := range this.Items {
				switch i.Text {
				case "Disabled", "Muted":
					continue
				default:
					i.Checked.Value = !item.Checked.Value
				}
			}
			return true
		},
	}

	if config.Current.Advanced.Notifications.Disabled.All {
		this := ui.sections.notifications.widget.(*dropdown.Widget)
		for i := range this.Items {
			if this.Items[i].Text == "Disabled" {
				continue
			}
			this.Items[i].Checked.Value = true
		}
	}

	ui.sections.factor = &section{
		title:       material.Label(ui.bar.Collection.Calibri().Theme, 14, "🕔 Match Frequency Factor"),
		description: material.Caption(ui.bar.Collection.Calibri().Theme, "Decrease the amount of match attempts per second"),
		widget: &dropdown.Widget{
			Theme:    ui.bar.Collection.NotoSans().Theme,
			Radio:    true,
			TextSize: 12,
			Items: []*dropdown.Item{
				{
					Text: "Default",
					Checked: widget.Bool{
						Value: false,
					},
					Callback: func(this *dropdown.Item) {
						config.Current.Advanced.DecreasedCaptureLevel = 0
					},
				},
				{
					Text: "Moderate",
					Checked: widget.Bool{
						Value: false,
					},
					Callback: func(this *dropdown.Item) {
						config.Current.Advanced.DecreasedCaptureLevel = 1
					},
				},
				{
					Text: "Mild",
					Checked: widget.Bool{
						Value: false,
					},
					Callback: func(this *dropdown.Item) {
						config.Current.Advanced.DecreasedCaptureLevel = 2
					},
				},
				{
					Text: "Maximum",
					Checked: widget.Bool{
						Value: false,
					},
					Callback: func(this *dropdown.Item) {
						config.Current.Advanced.DecreasedCaptureLevel = 3
					},
				},
				{
					Text: "Unreliable",
					Checked: widget.Bool{
						Value: false,
					},
					Callback: func(this *dropdown.Item) {
						config.Current.Advanced.DecreasedCaptureLevel = 4
					},
				},
			},
		},
	}
	for i, item := range ui.sections.factor.widget.(*dropdown.Widget).Items {
		if i == int(config.Current.Advanced.DecreasedCaptureLevel) {
			item.Checked.Value = true
		}
	}
	frequencyWarning := material.Label(ui.bar.Collection.Calibri().Theme, unit.Sp(12), "✔ Decreasing the match factor will reduce CPU usage")
	frequencyWarning.Color = nrgba.PastelGreen.Alpha(127).Color()
	frequencyWarning.Font.Weight = 0
	ui.sections.factor.warning = frequencyWarning

	ui.sections.theme = &section{
		title:       material.Label(ui.bar.Collection.Calibri().Theme, 14, "🎨 Theme"),
		description: material.Label(ui.bar.Collection.Calibri().Theme, unit.Sp(12), "Adjust the overall color scheme of UniteHUD"),
		widget: colorpicker.New(
			ui.bar.Collection.Calibri(),
			colorpicker.Option{
				Label: "Background",
				Value: &config.Current.Theme.Background,
			},
			colorpicker.Option{
				Label: "Background Alt.",
				Value: &config.Current.Theme.BackgroundAlt,
			},
			colorpicker.Option{
				Label: "Foreground",
				Value: &config.Current.Theme.Foreground,
			},
			colorpicker.Option{
				Label: "Foreground Alt.",
				Value: &config.Current.Theme.ForegroundAlt,
			},
			colorpicker.Option{
				Label: "Title Bar Foreground",
				Value: &config.Current.Theme.TitleBarForeground,
			},
			colorpicker.Option{
				Label: "Title Bar Background",
				Value: &config.Current.Theme.TitleBarBackground,
			},
			colorpicker.Option{
				Label: "Splash",
				Value: &config.Current.Theme.Splash,
			},
			colorpicker.Option{
				Label: "Borders",
				Value: &config.Current.Theme.Borders,
			},
			colorpicker.Option{
				Label: "Scrollbar Background",
				Value: &config.Current.Theme.ScrollbarBackground,
			},
			colorpicker.Option{
				Label: "Scrollbar Foreground",
				Value: &config.Current.Theme.ScrollbarForeground,
			},
		),
		warning: &button.Widget{
			Text:            "Reset",
			Pressed:         nrgba.Transparent80,
			Released:        nrgba.DarkGray,
			TextSize:        unit.Sp(14),
			TextInsetBottom: unit.Dp(-2),
			Size:            image.Pt(80, 20),
			Font:            ui.bar.Collection.Calibri(),

			Click: func(this *button.Widget) {
				config.Current.SetDefaultTheme()
			},
		},
	}
	ui.sections.theme.warning.(*button.Widget).Click = func(this *button.Widget) {
		defer this.Deactivate()

		ui.sections.theme.widget.(*colorpicker.Widget).ApplyDefaults()
	}

	ui.sections.themes = &section{
		title:       material.Label(ui.bar.Collection.Calibri().Theme, 14, "📦 Preset Themes"),
		description: material.Caption(ui.bar.Collection.Calibri().Theme, "Select a theme preset to apply to UniteHUD"),
		warning:     material.Label(ui.bar.Collection.Calibri().Theme, 12, ""),
		widget: &dropdown.Widget{
			Theme:    ui.bar.Collection.NotoSans().Theme,
			TextSize: 12,
			Items:    []*dropdown.Item{},
		},
	}
	for name := range config.Current.Themes {
		ui.sections.themes.widget.(*dropdown.Widget).Items = append(ui.sections.themes.widget.(*dropdown.Widget).Items,
			&dropdown.Item{
				Text: name,
				Callback: func(this *dropdown.Item) {
					println(this.Text)
				},
			},
		)
	}

	return ui
}

func (s *section) section(gtx layout.Context) layout.Dimensions {
	inset := layout.Inset{
		Top:    5,
		Left:   12.5,
		Right:  2,
		Bottom: 10,
	}

	alignment := layout.Baseline

	s.title.Font.Weight = font.Black

	decorate.Label(&s.title, s.title.Text)
	decorate.LabelAlpha(&s.title, 150)

	decorate.Label(&s.description, s.description.Text)

	children := []layout.FlexChild{
		// Title: "Advanced Settings".
		// Subtitle: "Discord Activity", "Theme Presets".
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if s.title.Text == "" {
				return layout.Dimensions{Size: layout.Exact(image.Pt(0, 0)).Max}
			}

			layout.Inset{Bottom: 5}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return decorate.Line(
					gtx,
					clip.Rect(image.Rect(8, 0, gtx.Constraints.Max.X-8, 1)),
					nrgba.NRGBA(config.Current.Theme.Borders),
				)
			})

			return layout.Inset{
				Top:    inset.Top + 10,
				Left:   inset.Left - 2,
				Right:  inset.Right - 2,
				Bottom: inset.Bottom,
			}.Layout(gtx, s.title.Layout)
		}),

		// Widget: Button, Slider, etc.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if s.widget == nil {
				return layout.Dimensions{Size: layout.Exact(image.Pt(0, 0)).Max}
			}

			return inset.Layout(gtx, s.widget.Layout)
		}),

		// Label: "Configure blah".
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if s.description.Text == "" {
				return layout.Dimensions{Size: layout.Exact(image.Pt(0, 0)).Max}
			}

			if s.h1 {
				return layout.Inset{
					Top:    inset.Bottom,
					Left:   inset.Left,
					Bottom: inset.Top,
				}.Layout(gtx, s.description.Layout)
			}

			return inset.Layout(gtx, s.description.Layout)
		}),

		// Label: "This setting will blah".
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if s.warning == nil {
				return layout.Dimensions{Size: layout.Exact(image.Pt(0, 0)).Max}
			}
			return inset.Layout(gtx, s.warning.Layout)
		}),
	}

	return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: alignment,
		}.Layout(gtx, append(children, s.footer(inset)...)...)
	})
}

func (s *section) footer(inset layout.Inset) []layout.FlexChild {
	c := []layout.FlexChild{}

	for _, widget := range s.extras {
		c = append(c,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return inset.Layout(gtx, widget.Layout)
			}),
		)
	}

	return c
}

func (s *settings) close() {
	if s != nil {
		go s.windows.current.Perform(system.ActionClose)
	}
}

func (s *settings) fill() layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
}

func (s *settings) open() bool {
	return s != nil
}

func (s *settings) resize() {
	if s != nil {
		s.dimensions.resize = true
	}
}

func (s *settings) spacer(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Bottom: 10}.Layout(gtx, decorate.Border)
}
