package gui

import (
	"image"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/fonts"
	"github.com/pidgy/unitehud/gui/visual"
	"github.com/pidgy/unitehud/gui/visual/colorpicker"
	"github.com/pidgy/unitehud/gui/visual/decorate"
	"github.com/pidgy/unitehud/gui/visual/slider"
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/nrgba"
)

type section struct {
	title, description material.LabelStyle
	warning, widget    visual.Widgeter
	dims               layout.Dimensions
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
		app.MaxSize(unit.Dp(s.width), unit.Dp(s.parent.max.Y)),
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

	frequency := &section{
		title:       material.Label(bar.Collection.Calibri().Theme, unit.Sp(15), "Match Interval"),
		description: material.Caption(bar.Collection.Calibri().Theme, "Increase the amount of match attempts per second"),
		widget: &slider.Widget{
			Slider:     material.Slider(bar.Collection.Calibri().Theme, &widget.Float{Value: float32(config.Current.Advanced.IncreasedCaptureRate)}, 0, 99),
			Label:      material.Label(bar.Collection.Calibri().Theme, unit.Sp(15), ""),
			TextColors: []nrgba.NRGBA{nrgba.White, nrgba.PastelYellow, nrgba.PastelOrange, nrgba.PastelRed},
			OnValueChanged: func(f float32) {
				config.Current.Advanced.IncreasedCaptureRate = int64(f)
			},
		},
	}
	frequencyWarningLabel := material.Label(bar.Collection.NotoSans().Theme, unit.Sp(11), "⚠ Increases CPU Usage")
	frequencyWarningLabel.Color = nrgba.PastelRed.Alpha(127).Color()
	frequencyWarningLabel.Font.Weight = 0
	frequency.warning = frequencyWarningLabel

	theme := &section{
		title:       material.Label(bar.Collection.Calibri().Theme, unit.Sp(15), "Theme"),
		description: material.Caption(bar.Collection.Calibri().Theme, "Change the color theme of UniteHUD"),
		widget: colorpicker.New(bar.Collection.Calibri(), []colorpicker.Option{
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
				Label: "Tool Tip Foreground",
				Value: &config.Current.Theme.ToolTipForeground,
			},
			{
				Label: "Borders",
				Value: &config.Current.Theme.Borders,
			},
			{
				Label: "Scrollbar",
				Value: &config.Current.Theme.Scrollbar,
			},
		}...),
	}

	theme.warning = theme.widget.(*colorpicker.Widget).DefaultButton

	var ops op.Ops

	s.window.Perform(system.ActionRaise)

	s.parent.setInsetRight(s.width)
	defer s.parent.unsetInsetRight(s.width)

	list := material.List(bar.Collection.Calibri().Theme, &widget.List{
		Scrollbar: widget.Scrollbar{},
		List: layout.List{
			Axis:      layout.Vertical,
			Alignment: layout.Start,
		},
	})

	widgets := []layout.Widget{
		frequency.section,
		s.spacer,
		theme.section,
		s.spacer,
	}

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
				decorate.ColorBox(gtx, gtx.Constraints.Max, nrgba.NRGBA(config.Current.Theme.BackgroundAlt))

				return list.Layout(gtx, len(widgets), func(gtx layout.Context, index int) layout.Dimensions {
					return widgets[index](gtx)
				})
			})

			s.window.Invalidate()
			e.Frame(gtx.Ops)
		default:
			notify.Debug("Event missed: %T (Settings Window)", e)
		}
	}
}

func (s *settings) fill() layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
}

func (s *section) section(gtx layout.Context) layout.Dimensions {
	inset := layout.UniformInset(2)

	decorate.ColorBox(gtx, s.dims.Size, nrgba.NRGBA(config.Current.Theme.BackgroundAlt))

	decorate.Label(&s.title, s.title.Text)
	decorate.Label(&s.description, s.title.Text)

	s.dims = layout.Inset{
		Top: unit.Dp(5),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: layout.Baseline,
		}.Layout(gtx,
			s.spacer(),

			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return inset.Layout(gtx, s.title.Layout)
			}),

			s.spacer(),

			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if s.widget == nil {
					return layout.Dimensions{Size: gtx.Constraints.Max}
				}
				return inset.Layout(gtx, s.widget.Layout)
			}),

			s.spacer(),

			s.spacer(),

			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				inset.Top += 5
				return inset.Layout(gtx, s.description.Layout)
			}),

			s.spacer(),

			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				inset.Top -= 5
				return inset.Layout(gtx, s.warning.Layout)
			}),

			s.spacer(),
		)
	})

	return s.dims
}

func (s *settings) spacer(gtx layout.Context) layout.Dimensions {
	return decorate.ColorBox(gtx, image.Pt(gtx.Constraints.Max.X, 5), nrgba.White.Alpha(80))
}

func (s *section) spacer() layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Spacer{Width: unit.Dp(gtx.Constraints.Max.X), Height: 1}.Layout(gtx)
	})
}
