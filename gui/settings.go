package gui

import (
	"image"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/desktop"
	"github.com/pidgy/unitehud/desktop/clicked"
	"github.com/pidgy/unitehud/discord"
	"github.com/pidgy/unitehud/fonts"
	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/gui/visual"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/colorpicker"
	"github.com/pidgy/unitehud/gui/visual/decorate"
	"github.com/pidgy/unitehud/gui/visual/dropdown"
	"github.com/pidgy/unitehud/gui/visual/slider"
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/nrgba"
)

type section struct {
	title, description material.LabelStyle
	warning, widget    visual.Widgeter
	isTitle            bool

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
		title,
		discord,
		notifications,
		frequency,
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
			ui.sections.title,
			ui.sections.discord,
			ui.sections.notifications,
			ui.sections.frequency,
			ui.sections.theme,
			ui.sections.themes,
		}

		var ops op.Ops

		for event := range ui.windows.current.Events() {
			switch e := event.(type) {
			case system.DestroyEvent:
				return
			case app.ViewEvent:
				ui.hwnd = e.HWND
				ui.windows.parent.attachWindowRight(ui.hwnd, ui.dimensions.width)
			case system.FrameEvent:
				if !ui.state.open {
					go ui.windows.current.Perform(system.ActionClose)
				}

				if ui.dimensions.resize {
					ui.dimensions.resize = false
					ui.windows.parent.attachWindowRight(ui.hwnd, ui.dimensions.width)
				}

				gtx := layout.NewContext(&ops, e)

				ui.bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
						decorate.List(&ui.list)
						decorate.Scrollbar(&ui.list.ScrollbarStyle)

						return ui.list.Layout(gtx, len(sections), func(gtx layout.Context, index int) layout.Dimensions {
							return sections[index].section(gtx)
						})
					})
				})

				ui.windows.current.Invalidate()
				e.Frame(gtx.Ops)
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

	ui.sections.title = &section{
		description: material.Label(ui.bar.Collection.NotoSans().Theme, 16, "⚙  Advanced Settings"),
		isTitle:     true,
	}
	ui.sections.title.description.Alignment = text.Middle

	ui.sections.discord = &section{
		title:       material.Label(ui.bar.Collection.NotoSans().Theme, 14, "🎮  Discord Activity"),
		description: material.Caption(ui.bar.Collection.NotoSans().Theme, "Enable/Disable Discord activity updates"),
		widget: &button.Widget{
			Text:            "Enabled",
			Pressed:         nrgba.Transparent80,
			Released:        nrgba.PastelGreen,
			TextSize:        unit.Sp(14),
			TextInsetBottom: unit.Dp(-2),
			Size:            image.Pt(80, 20),
			Font:            ui.bar.Collection.Calibri(),

			OnHoverHint: func() {},

			Click: func(this *button.Widget) {
				config.Current.Advanced.Discord.Disabled = !config.Current.Advanced.Discord.Disabled

				if config.Current.Advanced.Discord.Disabled {
					this.Released = nrgba.PastelRed
					this.Text = "Disabled"
					discord.Disconnect()
				} else {
					this.Released = nrgba.PastelGreen
					this.Text = "Enabled"
				}
			},
		},
	}
	discordWarning := material.Label(ui.bar.Collection.NotoSans().Theme, unit.Sp(11),
		"🔌 Activity Privacy settings in Discord can prevent this feature from working")
	discordWarning.Color = nrgba.PastelRed.Alpha(127).Color()
	discordWarning.Font.Weight = 0
	ui.sections.discord.warning = discordWarning

	if config.Current.Advanced.Discord.Disabled {
		ui.sections.discord.widget.(*button.Widget).Released = nrgba.PastelRed
		ui.sections.discord.widget.(*button.Widget).Text = "Disabled"
	}

	ui.sections.notifications = &section{
		title:       material.Label(ui.bar.Collection.NotoSans().Theme, 14, "🔔  Desktop Notifications"),
		description: material.Caption(ui.bar.Collection.NotoSans().Theme, "Adjust desktop notifications for UniteHUD"),

		extras: []visual.Widgeter{
			&button.Widget{
				Text:            "🔔 Test",
				Pressed:         nrgba.Transparent80,
				Released:        nrgba.PastelBlue,
				TextSize:        unit.Sp(12),
				TextInsetBottom: unit.Dp(0),
				Size:            image.Pt(80, 20),
				Font:            ui.bar.Collection.NotoSans(),

				OnHoverHint: func() {},

				Click: func(this *button.Widget) {
					desktop.Notification(global.Title).
						Says("Testing 1..2..3").
						When(clicked.VisitWebsite).
						Send()
				},
			},
		},
	}
	notificationsWarning := material.Label(ui.bar.Collection.NotoSans().Theme, unit.Sp(11),
		"📌 Some settings are automatically applied by the OS")
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
		Callback: func(item *dropdown.Item, this *dropdown.Widget) {
			if item.Text != "Disabled" {
				return
			}

			for _, i := range this.Items {
				switch i.Text {
				case "Disabled", "Muted":
					continue
				default:
					i.Checked.Value = !item.Checked.Value
				}
			}
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

	ui.sections.frequency = &section{
		title:       material.Label(ui.bar.Collection.NotoSans().Theme, 14, "🕔  Match Interval"),
		description: material.Caption(ui.bar.Collection.NotoSans().Theme, "Increase the amount of match attempts per second"),
		widget: &slider.Widget{
			Slider:     material.Slider(ui.bar.Collection.NotoSans().Theme, &widget.Float{Value: float32(config.Current.Advanced.IncreasedCaptureRate)}, -99, 99),
			Label:      material.Label(ui.bar.Collection.NotoSans().Theme, unit.Sp(15), ""),
			TextColors: []nrgba.NRGBA{nrgba.White, nrgba.PastelYellow, nrgba.PastelOrange, nrgba.PastelRed},
			OnValueChanged: func(f float32) {
				config.Current.Advanced.IncreasedCaptureRate = int64(f)
			},
		},
	}
	frequencyWarning := material.Label(ui.bar.Collection.NotoSans().Theme, unit.Sp(11), "⚠ CPU Increase when ≥ 1")
	frequencyWarning.Color = nrgba.PastelRed.Alpha(127).Color()
	frequencyWarning.Font.Weight = 0
	ui.sections.frequency.warning = frequencyWarning

	ui.sections.theme = &section{
		title:       material.Label(ui.bar.Collection.NotoSans().Theme, 14, "🎨  Theme"),
		description: material.Caption(ui.bar.Collection.NotoSans().Theme, "Change the color theme of UniteHUD"),
		widget: colorpicker.New(
			ui.bar.Collection.NotoSans(),
			[]colorpicker.Options{
				{
					Label: "Background",
					Value: &config.Current.Theme.Background,
				},
				{
					Label: "Background Alt.",
					Value: &config.Current.Theme.BackgroundAlt,
				},
				{
					Label: "Foreground",
					Value: &config.Current.Theme.Foreground,
				},
				{
					Label: "Foreground Alt.",
					Value: &config.Current.Theme.ForegroundAlt,
				},
				{
					Label: "Title Bar Foreground",
					Value: &config.Current.Theme.TitleBarForeground,
				},
				{
					Label: "Title Bar Background",
					Value: &config.Current.Theme.TitleBarBackground,
				},
				{
					Label: "Splash",
					Value: &config.Current.Theme.Splash,
				},
				{
					Label: "Borders",
					Value: &config.Current.Theme.Borders,
				},
				{
					Label: "Scrollbar Background",
					Value: &config.Current.Theme.ScrollbarBackground,
				},
				{
					Label: "Scrollbar Foreground",
					Value: &config.Current.Theme.ScrollbarForeground,
				},
			}...,
		),
		warning: &button.Widget{
			Text:            "Defaults",
			Pressed:         nrgba.Transparent80,
			Released:        nrgba.DarkGray,
			TextSize:        unit.Sp(14),
			TextInsetBottom: unit.Dp(-2),
			Size:            image.Pt(80, 20),
			Font:            ui.bar.Collection.Calibri(),

			OnHoverHint: func() {},

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
		title:       material.Label(ui.bar.Collection.NotoSans().Theme, 14, "📦  Preset Themes"),
		description: material.Caption(ui.bar.Collection.NotoSans().Theme, "Select a theme preset to apply to UniteHUD"),
		warning:     material.Label(ui.bar.Collection.NotoSans().Theme, 14, ""),
		widget: &dropdown.Widget{
			Theme:    ui.bar.Collection.NotoSans().Theme,
			TextSize: 16,
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
		Bottom: 5,
	}

	alignment := layout.Baseline

	title := s.warning == nil && s.widget == nil && len(s.extras) == 0
	if title {
		alignment = layout.Middle
		inset = layout.Inset{}
	}

	s.title.Font.Weight = font.Black

	decorate.Label(&s.title, s.title.Text)
	decorate.Label(&s.description, s.description.Text)

	children := []layout.FlexChild{
		s.spacer(),

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if s.title.Text == "" {
				return layout.Dimensions{Size: layout.Exact(image.Pt(0, 0)).Max}
			}
			return inset.Layout(gtx, s.title.Layout)
		}),

		s.spacer(),

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if s.widget == nil {
				return layout.Dimensions{Size: layout.Exact(image.Pt(0, 0)).Max}
			}
			return inset.Layout(gtx, s.widget.Layout)
		}),

		s.spacer(),

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if s.description.Text == "" {
				return layout.Dimensions{Size: layout.Exact(image.Pt(0, 0)).Max}
			}
			return inset.Layout(gtx, s.description.Layout)
		}),

		s.spacer(),

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if s.warning == nil {
				return layout.Dimensions{Size: layout.Exact(image.Pt(0, 0)).Max}
			}
			return inset.Layout(gtx, s.warning.Layout)
		}),
	}

	return decorate.Fill(gtx, nrgba.NRGBA(config.Current.Theme.Background), func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Top: unit.Dp(5),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return decorate.Fill(gtx, nrgba.NRGBA(config.Current.Theme.BackgroundAlt), func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis:      layout.Vertical,
					Alignment: alignment,
				}.Layout(gtx, append(children, s.footer(inset)...)...)
			})
		})
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

	return append(c, s.spacer(), s.spacer())
}

func (s *section) spacer() layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Spacer{Width: unit.Dp(gtx.Constraints.Max.X), Height: 2}.Layout(gtx)
	})
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
