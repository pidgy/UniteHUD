package button

import (
	"image"
	"image/color"
	"time"

	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/rgba"
)

type Button struct {
	Text                          string
	TextSize                      unit.Value
	TextOffsetTop, TextOffsetLeft float32
	BorderWidth                   unit.Value

	Size image.Point

	Active            bool
	Disabled          bool
	LastPressed       time.Time
	Pressed, Released color.NRGBA

	Click       func()
	SingleClick bool // Toggle the Active field on Click events.

	hover bool
	alpha uint8
}

var Max = image.Pt(100, 35)

func (b *Button) Error() {
	tmp := b.Pressed
	b.Pressed = color.NRGBA(rgba.Red)
	b.Disabled = true
	time.AfterFunc(time.Second*2, func() {
		b.Pressed = tmp
		b.Disabled = false
	})
}

func (b *Button) Layout(gtx layout.Context) layout.Dimensions {
	if b.Size.Eq(image.Pt(0, 0)) {
		b.Size = Max
	}

	if b.alpha == 0 {
		b.alpha = b.Released.A
	}

	for _, e := range gtx.Events(b) {
		if e, ok := e.(pointer.Event); ok {
			switch e.Type {
			case pointer.Enter:
				b.Released = color.NRGBA(rgba.Alpha(color.RGBA(b.Released), 0x50))
				b.hover = true
			case pointer.Release:
				if b.hover && b.Click != nil {
					b.Click()
					if b.SingleClick {
						b.Active = !b.Active
					}
				} else {
					b.Active = !b.Active
				}
			case pointer.Leave:
				b.Released = color.NRGBA(rgba.Alpha(color.RGBA(b.Released), b.alpha))
				b.hover = false
			case pointer.Press:
				b.Active = !b.Active
			case pointer.Move:
				println("Move")
			}
		}
	}

	// Confine the area for pointer events.
	if !b.Disabled {
		area := clip.Rect(image.Rect(0, 0, b.Size.X, b.Size.Y)).Push(gtx.Ops)
		pointer.InputOp{
			Tag:   b,
			Types: pointer.Press | pointer.Release | pointer.Enter | pointer.Leave,
		}.Add(gtx.Ops)
		area.Pop()
	}

	return b.draw(gtx)
}

func (b *Button) uniform(gtx layout.Context) layout.Dimensions {
	defer clip.RRect{SE: 3, SW: 3, NE: 3, NW: 3, Rect: f32.Rectangle{Max: f32.Pt(float32(b.Size.X), float32(b.Size.Y))}}.Push(gtx.Ops).Pop()

	col := b.Pressed
	if !b.Active {
		col = b.Released
	}

	paint.ColorOp{Color: col}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: b.Size}
}

func (b *Button) draw(gtx layout.Context) layout.Dimensions {
	if b.BorderWidth.V == unit.Px(0).V {
		b.BorderWidth = unit.Px(2)
	}

	widget.Border{
		Color:        color.NRGBA{A: 0xAF},
		Width:        b.BorderWidth,
		CornerRadius: unit.Px(2),
	}.Layout(gtx, b.uniform)

	title := material.Body1(material.NewTheme(gofont.Collection()), b.Text)
	title.Color = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	if b.Active && b.Click != nil {
		title.Color.A = 0xFF
	}
	if !b.Active && b.Click == nil {
		title.Color.A = 0x3F
	}

	if b.TextSize.V == 0 {
		title.TextSize = unit.Px(16)
	} else {
		title.TextSize = b.TextSize
	}

	return layout.Inset{
		Left: unit.Dp(11 + b.TextOffsetLeft),
		Top:  unit.Px(((float32(b.Size.Y) / title.TextSize.V) * 3) + b.TextOffsetTop),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.N.Layout(gtx, title.Layout)
	})
}
