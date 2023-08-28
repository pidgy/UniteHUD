package gui

import (
	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/fonts"
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/nrgba"
)

type settings struct {
	parent  *GUI
	visible bool

	width,
	height int
}

func (s *settings) close() bool {
	was := s.visible
	s.visible = false
	return was
}

func (s *settings) open() {
	if s.visible {
		return
	}
	s.visible = true
	defer func() { s.visible = false }()

	hwnd := uintptr(0)

	w := app.NewWindow(
		app.Title("UniteHUD Settings"),
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
		func() { w.Perform(system.ActionClose) },
	)

	list := &widget.List{
		Scrollbar: widget.Scrollbar{},
		List: layout.List{
			Axis:      layout.Vertical,
			Alignment: layout.Baseline,
		},
	}
	style := material.List(bar.Collection.Calibri().Theme, list)
	style.Track.Color = nrgba.Gray.Color()

	var ops op.Ops

	w.Perform(system.ActionRaise)

	s.parent.setInsetRight(s.width)
	defer s.parent.unsetInsetRight(s.width)

	for event := range w.Events() {
		switch e := event.(type) {
		case system.DestroyEvent:
			return
		case app.ViewEvent:
			hwnd = e.HWND
		case system.FrameEvent:
			if !s.visible {
				go w.Perform(system.ActionClose)
			}

			gtx := layout.NewContext(&ops, e)

			s.parent.attachWindowRight(hwnd, s.width)

			bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				colorBox(gtx, gtx.Constraints.Max, nrgba.Background)

				return layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					s.spacer(10, 10),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Dimensions{Size: gtx.Constraints.Max}
					}),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Dimensions{Size: gtx.Constraints.Max}
					}),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Dimensions{Size: gtx.Constraints.Max}
					}),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Dimensions{Size: gtx.Constraints.Max}
					}),
				)
			})

			w.Invalidate()
			e.Frame(gtx.Ops)
		}
	}
}

func (s *settings) spacer(x, y float32) layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		if x != 0 {
			gtx.Constraints.Max.X = int(x)
		}
		if y != 0 {
			gtx.Constraints.Max.Y = int(y)
		}
		colorBox(gtx, gtx.Constraints.Max, nrgba.White.Alpha(5))

		return layout.Spacer{Width: unit.Dp(x), Height: unit.Dp(y)}.Layout(gtx)
	})
}
