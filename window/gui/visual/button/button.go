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
)

type Button struct {
	Active            bool
	Text              string
	Click             func()
	Pressed, Released color.NRGBA
	Disabled          bool
	Size              image.Point
	TextSize          unit.Value
	LastPressed       time.Time
}

var Max = image.Pt(100, 35)

func (b *Button) Layout(gtx layout.Context) layout.Dimensions {
	if b.Size.Eq(image.Pt(0, 0)) {
		b.Size = Max
	}

	// here we loop through all the events associated with this button.
	for _, e := range gtx.Events(b) {
		if e, ok := e.(pointer.Event); ok {
			switch e.Type {
			case pointer.Press:
				b.Active = !b.Active
			case pointer.Release:
				if b.Click != nil {
					b.Click()
				}
			}
		}
	}

	// Confine the area for pointer events.
	if !b.Disabled {
		area := clip.Rect(image.Rect(0, 0, b.Size.X, b.Size.Y)).Push(gtx.Ops)
		pointer.InputOp{
			Tag:   b,
			Types: pointer.Press | pointer.Release,
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
	widget.Border{
		Color:        color.NRGBA{A: 0xAF},
		Width:        unit.Px(3),
		CornerRadius: unit.Px(1),
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
		Left: unit.Dp(11),
		Top:  unit.Px((float32(b.Size.Y) / title.TextSize.V) * 3),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.N.Layout(gtx, title.Layout)
	})

}
