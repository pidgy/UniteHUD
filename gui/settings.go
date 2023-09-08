package gui

import (
	"image"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/fonts"
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
}

type settings struct {
	parent *GUI
	window *app.Window
	hwnd   uintptr
	closed bool
	resize bool

	width,
	height int
}

func (s *settings) close() bool {
	was := s.closed
	s.closed = true
	return !was
}

func (s *settings) open(onclose func()) {
	if !s.closed {
		return
	}
	s.closed = false

	defer onclose()

	s.window = app.NewWindow(
		app.Title("Settings"),
		app.Size(unit.Dp(s.width), unit.Dp(s.height)),
		app.MinSize(unit.Dp(s.width), unit.Dp(s.height)),
		app.MaxSize(unit.Dp(s.width), unit.Dp(s.parent.dimensions.max.Y)),
		app.Decorated(false),
	)

	bar := title.New(
		"Settings",
		fonts.NewCollection(),
		nil,
		nil,
		func() { s.window.Perform(system.ActionClose) },
	)
	bar.NoTip = true
	bar.NoDrag = true

	discord := &section{
		title:       material.Label(bar.Collection.NotoSans().Theme, 14, "🎮 Discord Activity"),
		description: material.Caption(bar.Collection.NotoSans().Theme, "Enable/Disable Discord activity updates"),
		widget: &button.Widget{
			Text:            "Enabled",
			Pressed:         nrgba.Transparent80,
			Released:        nrgba.PastelGreen,
			TextSize:        unit.Sp(14),
			TextInsetBottom: unit.Dp(-2),
			Size:            image.Pt(80, 20),
			Font:            bar.Collection.Calibri(),

			OnHoverHint: func() {},

			Click: func(this *button.Widget) {
				config.Current.Advanced.Discord.Disabled = !config.Current.Advanced.Discord.Disabled

				this.Released = nrgba.PastelGreen
				this.Text = "Enabled"

				if config.Current.Advanced.Discord.Disabled {
					this.Released = nrgba.PastelRed
					this.Text = "Disabled"
				}
			},
		},
	}
	discordWarning := material.Label(bar.Collection.NotoSans().Theme, unit.Sp(11),
		"🔌 Activity Privacy settings in Discord can prevent this feature from working")
	discordWarning.Color = nrgba.PastelRed.Alpha(127).Color()
	discordWarning.Font.Weight = 0
	discord.warning = discordWarning

	if config.Current.Advanced.Discord.Disabled {
		discord.widget.(*button.Widget).Released = nrgba.PastelRed
		discord.widget.(*button.Widget).Text = "Disabled"
	}

	notifications := &section{
		title:       material.Label(bar.Collection.NotoSans().Theme, 14, "🔔 Desktop Notifications"),
		description: material.Caption(bar.Collection.NotoSans().Theme, "Adjust desktop notifications for UniteHUD"),
	}
	notificationsWarning := material.Label(bar.Collection.NotoSans().Theme, unit.Sp(11),
		"📌 Some settings are automatically applied by the OS")
	notificationsWarning.Color = nrgba.PastelRed.Alpha(127).Color()
	notificationsWarning.Font.Weight = 0
	notifications.warning = notificationsWarning
	notifications.widget = &dropdown.Widget{
		Theme:    bar.Collection.NotoSans().Theme,
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
					Value: config.Current.Advanced.Notifications.Disabled.MatchStarting,
				},
				Callback: func(this *dropdown.Item) {
					this.Checked.Value = !this.Checked.Value
					config.Current.Advanced.Notifications.Disabled.MatchStarting = this.Checked.Value
				},
			},
			{
				Text: "Match Stopped",
				Checked: widget.Bool{
					Value: config.Current.Advanced.Notifications.Disabled.MatchStopped,
				},
				Callback: func(this *dropdown.Item) {
					config.Current.Advanced.Notifications.Disabled.MatchStopped = this.Checked.Value
				},
			},
			{
				Text: "Updates",
				Checked: widget.Bool{
					Value: config.Current.Advanced.Notifications.Disabled.Updates,
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

			for i := range this.Items {
				if this.Items[i] == item {
					continue
				}
				this.Items[i].Checked.Value = !item.Checked.Value
			}
		},
	}

	if config.Current.Advanced.Notifications.Disabled.All {
		this := notifications.widget.(*dropdown.Widget)
		for i := range this.Items {
			if this.Items[i].Text == "Disabled" {
				continue
			}
			this.Items[i].Checked.Value = true
		}
	}

	frequency := &section{
		title:       material.Label(bar.Collection.NotoSans().Theme, 14, "🕔 Match Interval"),
		description: material.Caption(bar.Collection.NotoSans().Theme, "Increase the amount of match attempts per second"),
		widget: &slider.Widget{
			Slider:     material.Slider(bar.Collection.NotoSans().Theme, &widget.Float{Value: float32(config.Current.Advanced.IncreasedCaptureRate)}, -99, 99),
			Label:      material.Label(bar.Collection.NotoSans().Theme, unit.Sp(15), ""),
			TextColors: []nrgba.NRGBA{nrgba.White, nrgba.PastelYellow, nrgba.PastelOrange, nrgba.PastelRed},
			OnValueChanged: func(f float32) {
				config.Current.Advanced.IncreasedCaptureRate = int64(f)
			},
		},
	}
	notificationsWarning = material.Label(bar.Collection.NotoSans().Theme, unit.Sp(11), "⚠ CPU Increase when ≥ 1")
	notificationsWarning.Color = nrgba.PastelRed.Alpha(127).Color()
	notificationsWarning.Font.Weight = 0
	frequency.warning = notificationsWarning

	theme := &section{
		title:       material.Label(bar.Collection.NotoSans().Theme, 14, "🎨 Theme"),
		description: material.Caption(bar.Collection.NotoSans().Theme, "Change the color theme of UniteHUD"),
		widget: colorpicker.New(
			bar.Collection.NotoSans(),
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
			Font:            bar.Collection.Calibri(),

			OnHoverHint: func() {},

			Click: func(this *button.Widget) {
				config.Current.SetDefaultTheme()
			},
		},
	}
	theme.warning.(*button.Widget).Click = func(this *button.Widget) {
		defer this.Deactivate()

		theme.widget.(*colorpicker.Widget).ApplyDefaults()
	}

	themes := &section{
		title:       material.Label(bar.Collection.NotoSans().Theme, 14, "📦 Preset Themes"),
		description: material.Caption(bar.Collection.NotoSans().Theme, "Select a theme preset to apply to UniteHUD"),
		warning:     material.Label(bar.Collection.NotoSans().Theme, 14, ""),
		widget: &dropdown.Widget{
			Theme:    bar.Collection.NotoSans().Theme,
			TextSize: 16,
			Items:    []*dropdown.Item{},
		},
	}
	for name := range config.Current.Themes {
		themes.widget.(*dropdown.Widget).Items = append(themes.widget.(*dropdown.Widget).Items,
			&dropdown.Item{
				Text: name,
				Callback: func(this *dropdown.Item) {
					println(this.Text)
				},
			},
		)
	}

	var ops op.Ops

	list := material.List(
		bar.Collection.Calibri().Theme,
		&widget.List{
			Scrollbar: widget.Scrollbar{},
			List: layout.List{
				Axis:      layout.Vertical,
				Alignment: layout.Start,
			},
		},
	)

	sections := []*section{
		discord,
		notifications,
		frequency,
		theme,
		themes,
	}

	s.window.Perform(system.ActionRaise)

	s.parent.setInsetRight(s.width)
	defer s.parent.unsetInsetRight(s.width)

	for event := range s.window.Events() {
		switch e := event.(type) {
		case system.DestroyEvent:
			return
		case app.ViewEvent:
			s.hwnd = e.HWND
			s.parent.attachWindowRight(s.hwnd, s.width)
		case system.FrameEvent:
			if s.closed {
				go s.window.Perform(system.ActionClose)
			}

			if s.resize {
				s.resize = false
				s.parent.attachWindowRight(s.hwnd, s.width)
			}

			gtx := layout.NewContext(&ops, e)

			bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
					decorate.Scrollbar(&list.ScrollbarStyle)

					return list.Layout(gtx, len(sections), func(gtx layout.Context, index int) layout.Dimensions {
						return sections[index].section(gtx)
					})
				})
			})

			s.window.Invalidate()
			e.Frame(gtx.Ops)
		default:
			notify.Missed(event, "Settings")
		}
	}
}

func (s *settings) fill() layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
}

func (s *section) section(gtx layout.Context) layout.Dimensions {
	inset := layout.Inset{
		Top:    0,
		Left:   12,
		Right:  12,
		Bottom: 0,
	}
	s.title.Font.Weight = font.ExtraLight

	decorate.Label(&s.title, s.title.Text)
	decorate.Label(&s.description, s.description.Text)

	return layout.Inset{
		Top: unit.Dp(5),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: layout.Baseline,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return s.title.Layout(gtx)
				})
			}),

			s.spacer(),

			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if s.widget == nil {
					return layout.Dimensions{Size: gtx.Constraints.Max}
				}
				return inset.Layout(gtx, s.widget.Layout)
			}),

			s.spacer(),

			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				inset.Top += 5
				return inset.Layout(gtx, s.description.Layout)
			}),

			s.spacer(),

			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				inset.Bottom += 5
				return inset.Layout(gtx, s.warning.Layout)
			}),

			s.spacer(),
			s.spacer(),
		)
	})
}

func (s *settings) spacer(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Bottom: 10}.Layout(gtx, decorate.Border)
}

func (s *section) spacer() layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Spacer{Width: unit.Dp(gtx.Constraints.Max.X), Height: 2}.Layout(gtx)
	})
}
