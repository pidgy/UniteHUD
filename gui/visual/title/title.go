package title

import (
	"image"
	"time"

	"gioui.org/font"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/cursor"
	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/img"
	"github.com/pidgy/unitehud/nrgba"
)

const (
	Default = "UniteHUD " + global.Version
)

var (
	height         = 24
	buttonSize     = image.Pt(height+5, height)
	buttonTextSize = unit.Sp(18)
	titleTextSize  = unit.Sp(12)
)

type Bar struct {
	Title   string
	theme   *material.Theme
	grabbed *bool

	Drag bool
	*decorations
}

type decorations struct {
	title      material.LabelStyle
	titleSet   bool
	titleInset layout.Inset

	icon      widget.Image
	iconSet   bool
	iconInset layout.Inset

	minimize, resize, close *button.Button

	clicked time.Time
}

func New(title string, theme *material.Theme, minimize, resize, close func()) *Bar {
	return &Bar{
		Title: "UniteHUD",

		theme:   theme,
		grabbed: new(bool),

		decorations: &decorations{
			title: material.Label(theme, titleTextSize, title),

			icon: widget.Image{
				Src:   paint.NewImageOp(img.Icon("icon48x48")),
				Fit:   widget.ScaleDown,
				Scale: .27,
			},

			minimize: &button.Button{
				Size:         buttonSize,
				Pressed:      nrgba.BackgroundAlt.Alpha(255),
				Released:     nrgba.PastelBlue,
				NoBorder:     true,
				SharpCorners: true,
				Text:         "-",
				TextSize:     buttonTextSize,
				Click: func(this *button.Button) {
					defer this.Deactivate()
					minimize()
				},
			},
			resize: &button.Button{
				Size:            buttonSize,
				Pressed:         nrgba.BackgroundAlt.Alpha(255),
				Released:        nrgba.PastelGreen,
				NoBorder:        true,
				SharpCorners:    true,
				Text:            "□",
				TextSize:        buttonTextSize,
				TextInsetBottom: 1,
				Click: func(this *button.Button) {
					defer this.Deactivate()
					resize()
				},
			},
			close: &button.Button{
				Size:            buttonSize,
				Pressed:         nrgba.BackgroundAlt.Alpha(255),
				Released:        nrgba.PastelRed,
				NoBorder:        true,
				SharpCorners:    true,
				Text:            "×",
				TextSize:        buttonTextSize,
				TextInsetBottom: .5,
				Click: func(this *button.Button) {
					defer this.Deactivate()
					close()
				},
			},
		},
	}
}

func (b *Bar) Layout(gtx layout.Context, content layout.Widget) layout.Dimensions {
	for _, ev := range gtx.Events(b.grabbed) {
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}

		println(e.Type.String())
		switch e.Type {
		case pointer.Enter:
		case pointer.Leave:
		case pointer.Cancel:
		case pointer.Press:
			cursor.Is(pointer.CursorPointer)
		case pointer.Release:
			cursor.Is(pointer.CursorDefault)

			if time.Since(b.decorations.clicked) < time.Second/2 {
				defer b.decorations.resize.Deactivate()
				b.decorations.resize.Click(b.resize)
			} else {
				b.decorations.clicked = time.Now()
			}
		case pointer.Move:
			b.Drag = true
		case pointer.Drag:
			b.Drag = true
			cursor.Is(pointer.CursorPointer)
		}
	}

	bar := image.Rect(0, 0, gtx.Constraints.Max.X, height)

	dims := layout.Flex{
		Spacing:   layout.SpaceAround,
		Alignment: layout.Baseline,
		Axis:      layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			paint.ColorOp{Color: nrgba.BackgroundAlt.Color()}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)

			layout.Flex{
				Spacing:   layout.SpaceAround,
				Alignment: layout.Baseline,
				Axis:      layout.Horizontal,
			}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !b.decorations.iconSet {
						b.decorations.iconSet = true

						idims := b.decorations.icon.Layout(gtx)

						y := unit.Dp((float64(bar.Max.Y) - float64(idims.Size.Y)) / 2)
						b.iconInset = layout.Inset{Left: 5, Top: y, Bottom: y}
					}

					idims := b.iconInset.Layout(gtx, b.icon.Layout)

					if !b.decorations.titleSet {
						b.decorations.titleSet = true

						dims := b.decorations.title.Layout(gtx)

						b.decorations.title.Text = b.Title
						b.decorations.title.Color = nrgba.White.Color()
						b.decorations.title.Font.Weight = font.Thin
						// x := unit.Dp((float64(bar.Max.X) - float64(dims.Size.X)) / 2)
						y := unit.Dp((float64(bar.Max.Y) - float64(dims.Size.Y)) / 2)
						b.titleInset = layout.Inset{Left: 5 + unit.Dp(idims.Size.X), Top: y, Bottom: y}
						return dims
					}

					dims := b.titleInset.Layout(gtx, b.title.Layout)

					return layout.Dimensions{Size: layout.Exact(image.Pt(dims.Size.X+idims.Size.X+10, bar.Max.Y)).Max}
				}),
				layout.Flexed(.1, layout.Spacer{Width: unit.Dp(gtx.Constraints.Max.X)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return b.decorations.minimize.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return b.decorations.resize.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return b.decorations.close.Layout(gtx)
				}),
			)

			return layout.Dimensions{Size: layout.Exact(bar.Size()).Max}
		}),
		layout.Rigid(layout.Spacer{Height: .5}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			bar := image.Rect(0, 0, gtx.Constraints.Max.X, 1)
			defer clip.Rect(bar).Push(gtx.Ops).Pop()
			paint.ColorOp{Color: nrgba.Gray.Color()}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			return layout.Dimensions{Size: layout.Exact(bar.Size()).Max}
		}),
		layout.Rigid(layout.Spacer{Height: .5}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return content(gtx)
		}))

	defer clip.Rect(bar.Sub(image.Pt(b.minimize.Size.X+b.resize.Size.X+b.close.Size.X, 0))).Push(gtx.Ops).Pop()
	pointer.InputOp{
		Tag:   b.grabbed,
		Types: pointer.Press | pointer.Drag | pointer.Release | pointer.Leave | pointer.Enter | pointer.Move,
		Grab:  true,
	}.Add(gtx.Ops)

	return dims
}
