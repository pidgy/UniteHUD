package gui

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/fonts"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/nrgba"
)

func init() {
	tabs.tabs = []Tab{
		{Title: "⚙"},
		{Title: "¼"},
		{Title: "🗸"},
		{Title: "🖿"},
		{Title: "⚶"},
		{Title: "🖉"},
		{Title: "🖅"},
		{Title: "●"},
	}

}

const defaultDuration = 300 * time.Millisecond

var (
	tabs   Tabs
	slider Slider
)

// Slider implements sliding between old/new widget values.
type Slider struct {
	Duration time.Duration

	push int

	next *op.Ops

	nextCall op.CallOp
	lastCall op.CallOp

	t0     time.Time
	offset float32
}

type Tab struct {
	btn   widget.Clickable
	Title string
}

type Tabs struct {
	list     layout.List
	tabs     []Tab
	selected int
}

type (
	C = layout.Context
	D = layout.Dimensions
)

func (g *GUI) tabs() {
	defer g.next(is.Closing)

	dx, dy := float32(1280), float32(720)

	g.Window.Option(
		app.Title("tabs"),
		app.Size(unit.Dp(dx), unit.Dp(dy)),
		app.MaxSize(unit.Dp(dx+640), unit.Dp(dy+360)),
		app.MinSize(unit.Dp(dx), unit.Dp(dy)),
		app.Decorated(false),
	)

	th := fonts.NishikiTeki().Theme
	var ops op.Ops

	for g.is == is.TabMenu {
		e := <-g.Events()
		switch e := e.(type) {
		case system.DestroyEvent:
			return
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)

			g.Bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				colorBox(gtx, gtx.Constraints.Max, nrgba.White)
				return drawTabs(gtx, th)
			})

			g.frame(gtx, e)
		}
	}
}

func main2() {
	go func() {
		w := app.NewWindow()
		if err := loop(w); err != nil {
			log.Fatal(err)
		}
	}()
	app.Main()
}

func loop(w *app.Window) error {
	th := material.NewTheme(gofont.Collection())
	var ops op.Ops
	for {
		e := <-w.Events()
		switch e := e.(type) {
		case system.DestroyEvent:
			return e.Err
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)
			drawTabs(gtx, th)
			e.Frame(gtx.Ops)
		}
	}
}

func drawTabs(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return tabs.list.Layout(gtx, len(tabs.tabs), func(gtx C, tabIdx int) D {
				colorBox(gtx, gtx.Constraints.Max, nrgba.BackgroundAlt)

				t := &tabs.tabs[tabIdx]

				if t.btn.Clicked() {
					if tabs.selected < tabIdx {
						slider.PushLeft()
					} else if tabs.selected > tabIdx {
						slider.PushRight()
					}
					tabs.selected = tabIdx
				}

				var tabWidth int
				return layout.Stack{Alignment: layout.S}.Layout(gtx,
					layout.Stacked(func(gtx C) D {
						dims := material.Clickable(gtx, &t.btn, func(gtx C) D {
							return layout.UniformInset(unit.Dp(12)).Layout(gtx,
								func(gtx layout.Context) layout.Dimensions {
									h6 := material.H6(th, t.Title)
									h6.Color = nrgba.White.Color()
									return h6.Layout(gtx)
								},
							)
						})
						tabWidth = dims.Size.X
						return dims
					}),
					layout.Stacked(func(gtx C) D {
						if tabs.selected != tabIdx {
							return layout.Dimensions{}
						}
						tabHeight := gtx.Dp(unit.Dp(4))
						tabRect := image.Rect(0, 0, tabWidth, tabHeight)
						paint.FillShape(gtx.Ops, nrgba.White.Color(), clip.Rect(tabRect).Op())
						return layout.Dimensions{
							Size: image.Point{X: tabWidth, Y: tabHeight},
						}
					}),
				)
			})
		}),
		layout.Flexed(1, func(gtx C) D {
			return slider.Layout(gtx, func(gtx C) D {
				fill2(gtx, dynamicColor(tabs.selected), dynamicColor(tabs.selected+1))

				return layout.Center.Layout(gtx,
					material.H1(th, fmt.Sprintf("Tab content #%d", tabs.selected+1)).Layout,
				)
			})
		}),
	)
}

func fill2(gtx layout.Context, col1, col2 color.NRGBA) {
	dr := image.Rectangle{Max: gtx.Constraints.Min}
	paint.FillShape(gtx.Ops,
		nrgba.BackgroundAlt.Color(),
		clip.Rect(dr).Op(),
	)

	col2.R = byte(float32(col2.R))
	col2.G = byte(float32(col2.G))
	col2.B = byte(float32(col2.B))
	paint.LinearGradientOp{
		Stop1:  f32.Pt(float32(dr.Min.X), 0),
		Stop2:  f32.Pt(float32(dr.Max.X), 0),
		Color1: nrgba.BackgroundAlt.Color(),
		Color2: nrgba.BackgroundAlt.Color(),
	}.Add(gtx.Ops)
	defer clip.Rect(dr).Push(gtx.Ops).Pop()
	paint.PaintOp{}.Add(gtx.Ops)
}

func dynamicColor(i int) color.NRGBA {
	return nrgba.BackgroundAlt.Color()
}

// PushLeft pushes the existing widget to the left.
func (s *Slider) PushLeft() { s.push = 1 }

// PushRight pushes the existing widget to the right.
func (s *Slider) PushRight() { s.push = -1 }

// Layout lays out widget that can be pushed.
func (s *Slider) Layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	if s.push != 0 {
		s.next = nil
		s.lastCall = s.nextCall
		s.offset = float32(s.push)
		s.t0 = gtx.Now
		s.push = 0
	}

	var delta time.Duration
	if !s.t0.IsZero() {
		now := gtx.Now
		delta = now.Sub(s.t0)
		s.t0 = now
	}

	if s.offset != 0 {
		duration := s.Duration
		if duration == 0 {
			duration = defaultDuration
		}
		movement := float32(delta.Seconds()) / float32(duration.Seconds())
		if s.offset < 0 {
			s.offset += movement
			if s.offset >= 0 {
				s.offset = 0
			}
		} else {
			s.offset -= movement
			if s.offset <= 0 {
				s.offset = 0
			}
		}

		op.InvalidateOp{}.Add(gtx.Ops)
	}

	var dims layout.Dimensions
	{
		if s.next == nil {
			s.next = new(op.Ops)
		}
		gtx := gtx
		gtx.Ops = s.next
		gtx.Ops.Reset()
		m := op.Record(gtx.Ops)
		dims = w(gtx)
		s.nextCall = m.Stop()
	}

	if s.offset == 0 {
		s.nextCall.Add(gtx.Ops)
		return dims
	}

	offset := smooth(s.offset)

	if s.offset > 0 {
		defer op.Offset(image.Point{
			X: int(float32(dims.Size.X) * (offset - 1)),
		}).Push(gtx.Ops).Pop()
		s.lastCall.Add(gtx.Ops)

		defer op.Offset(image.Point{
			X: dims.Size.X,
		}).Push(gtx.Ops).Pop()
		s.nextCall.Add(gtx.Ops)
	} else {
		defer op.Offset(image.Point{
			X: int(float32(dims.Size.X) * (offset + 1)),
		}).Push(gtx.Ops).Pop()
		s.lastCall.Add(gtx.Ops)

		defer op.Offset(image.Point{
			X: -dims.Size.X,
		}).Push(gtx.Ops).Pop()
		s.nextCall.Add(gtx.Ops)
	}
	return dims
}

// smooth handles -1 to 1 with ease-in-out cubic easing func.
func smooth(t float32) float32 {
	if t < 0 {
		return -easeInOutCubic(-t)
	}
	return easeInOutCubic(t)
}

// easeInOutCubic maps a linear value to a ease-in-out-cubic easing function.
func easeInOutCubic(t float32) float32 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	return (t-1)*(2*t-2)*(2*t-2) + 1
}
