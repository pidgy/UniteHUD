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
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/slider"
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/nrgba"
)

type section struct {
	title, description, warning material.LabelStyle
	widget                      visual.Widgeter
	dimensions                  layout.Dimensions
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

	defer bar.Remove(bar.Add(&button.Button{
		Text:            "🖫",
		Font:            bar.Collection.NishikiTeki(),
		Released:        nrgba.OfficeBlue,
		TextSize:        unit.Sp(16),
		TextInsetBottom: -1,
		Disabled:        false,
		OnHoverHint:     func() { bar.ToolTip("Save configuration") },
		Click: func(this *button.Button) {
			s.parent.ToastYesNo("Save", "Save configuration changes?",
				func() {
					defer this.Deactivate()

					err := config.Current.Save()
					if err != nil {
						notify.Error("Failed to save UniteHUD configuration (%v)", err)
					}

					notify.System("Configuration saved to " + config.Current.File())
				},
				func() {
					defer this.Deactivate()
				},
			)
		},
	}))

	frequency := &section{
		title:       material.Label(bar.Collection.Calibri().Theme, unit.Sp(15), "Match Interval"),
		description: material.Caption(bar.Collection.Calibri().Theme, "Increase the amount of match attempts per second"),
		warning:     material.Label(bar.Collection.NotoSans().Theme, unit.Sp(11), "⚠ Increased CPU Usage"),
		widget: &slider.Slider{
			Slider:     material.Slider(bar.Collection.Calibri().Theme, &widget.Float{Value: float32(config.Current.Advanced.IncreasedCaptureRate)}, 0, 99),
			Label:      material.Label(bar.Collection.Calibri().Theme, unit.Sp(15), ""),
			TextColors: []nrgba.NRGBA{nrgba.White, nrgba.PastelYellow, nrgba.PastelOrange, nrgba.PastelRed},
			OnValueChanged: func(f float32) {
				config.Current.Advanced.IncreasedCaptureRate = int64(f)
			},
		},
	}
	frequency.title.Color = nrgba.White.Color()
	frequency.description.Color = nrgba.White.Color()
	frequency.warning.Color = nrgba.PastelRed.Alpha(127).Color()
	frequency.warning.Font.Weight = 0

	var ops op.Ops

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
				colorBox(gtx, gtx.Constraints.Max, nrgba.Background)

				return layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					frequency.section(gtx),

					s.spacer(),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Dimensions{Size: gtx.Constraints.Max}
					}),
				)
			})

			s.window.Invalidate()
			e.Frame(gtx.Ops)
		default:
			notify.Debug("Event missed: %T (Settings Window)", e)
		}
	}
}

func (s *section) section(gtx layout.Context) layout.FlexChild {
	inset := layout.Inset{
		Left:   unit.Dp(10),
		Right:  unit.Dp(10),
		Bottom: unit.Dp(1),
	}

	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		colorBox(gtx, s.dimensions.Size, nrgba.DarkGray)

		s.dimensions = layout.Flex{
			Axis: layout.Vertical,
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

			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return inset.Layout(gtx, s.description.Layout)
			}),

			s.spacer(),

			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if s.warning.Color == nrgba.Nothing.Color() {
					return layout.Dimensions{Size: gtx.Constraints.Max}
				}
				return inset.Layout(gtx, s.warning.Layout)
			}),

			s.spacer(),
		)

		return s.dimensions
	})
}

func (s *section) spacer() layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Spacer{Width: unit.Dp(gtx.Constraints.Max.X), Height: 2}.Layout(gtx)
	})
}

func (s *settings) spacer() layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		colorBox(gtx, image.Pt(gtx.Constraints.Max.X, 2), nrgba.White)

		return layout.Spacer{Width: unit.Dp(gtx.Constraints.Max.X), Height: 2}.Layout(gtx)
	})
}
