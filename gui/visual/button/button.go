package button

import (
	"image"
	"time"

	"gioui.org/font/gofont"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/cursor"
	"github.com/pidgy/unitehud/nrgba"
)

type Button struct {
	Text            string
	TextSize        unit.Sp
	TextInsetBottom unit.Dp
	BorderWidth     unit.Sp
	NoBorder        bool
	SharpCorners    bool

	Size image.Point

	Active            bool
	Disabled          bool
	LastPressed       time.Time
	Pressed, Released nrgba.NRGBA

	Click       func(b *Button)
	SingleClick bool // Toggle the Active field on Click events.

	hover bool
	alpha uint8

	inset layout.Inset
	set   bool
}

var Max = image.Pt(100, 35)

func (b *Button) Deactivate() {
	b.Active = !b.Active
}

func (b *Button) Error() {
	tmp := b.Pressed
	b.Pressed = nrgba.Red
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

	not := func() bool {
		if b.Disabled {
			cursor.Is(pointer.CursorNotAllowed)
			b.hover = false
		}
		return b.Disabled
	}

	for _, e := range gtx.Events(b) {
		if e, ok := e.(pointer.Event); ok {
			switch e.Type {
			case pointer.Enter:
				if not() {
					continue
				}
				cursor.Is(pointer.CursorPointer)

				b.Released = b.Released.Alpha(0x50)
				b.hover = true
			case pointer.Release:
				if not() {
					continue
				}

				cursor.Is(pointer.CursorPointer)

				if b.hover && b.Click != nil {
					b.Click(b)
					if b.SingleClick {
						b.Active = !b.Active
					}
				} else {
					b.Active = !b.Active
				}
			case pointer.Leave:
				cursor.Is(pointer.CursorDefault)

				b.Released = b.Released.Alpha(b.alpha)
				b.hover = false
			case pointer.Press:
				if not() {
					continue
				}

				cursor.Is(pointer.CursorPointer)

				b.Active = !b.Active
			case pointer.Move:
				if not() {
					continue
				}

				cursor.Is(pointer.CursorPointer)
			}
		}
	}

	// Confine the area for pointer events.
	area := clip.Rect(image.Rect(0, 0, b.Size.X, b.Size.Y)).Push(gtx.Ops)
	pointer.InputOp{
		Tag:   b,
		Types: pointer.Press | pointer.Release | pointer.Enter | pointer.Leave | pointer.Move,
	}.Add(gtx.Ops)
	area.Pop()

	return b.draw(gtx)
}

func (b *Button) uniform(gtx layout.Context) layout.Dimensions {
	rect := clip.RRect{SE: 3, SW: 3, NE: 3, NW: 3, Rect: image.Rectangle{Max: image.Pt((b.Size.X), b.Size.Y)}}
	if b.SharpCorners {
		rect = clip.RRect{SE: 0, SW: 0, NE: 0, NW: 0, Rect: image.Rectangle{Max: image.Pt((b.Size.X), b.Size.Y)}}
	}

	defer rect.Push(gtx.Ops).Pop()

	col := b.Pressed
	if !b.Active {
		col = b.Released
	}
	if b.hover {
		col = b.Pressed
	}

	paint.ColorOp{Color: col.Color()}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	return layout.Dimensions{Size: b.Size}
}

func (b *Button) draw(gtx layout.Context) layout.Dimensions {
	if b.TextSize == 0 {
		b.TextSize = unit.Sp(16)
	}

	if b.NoBorder {
		b.BorderWidth = 0
	}

	if b.hover {
		widget.Border{
			Color:        nrgba.White.Color(),
			Width:        unit.Dp(b.BorderWidth),
			CornerRadius: unit.Dp(2),
		}.Layout(gtx, b.uniform)
	} else {
		widget.Border{
			Color:        nrgba.Disabled.Color(),
			Width:        unit.Dp(b.BorderWidth),
			CornerRadius: unit.Dp(2),
		}.Layout(gtx, b.uniform)
	}

	t := material.Label(material.NewTheme(gofont.Collection()), b.TextSize, b.Text)
	t.Color = nrgba.White.Color()
	t.MaxLines = 1
	t.Truncator = t.Text

	if b.Active && b.Click != nil {
		t.Color.A = 0xFF
	}

	if !b.Active && b.Click == nil {
		t.Color.A = 0x3F
	}

	max := layout.Exact(b.Size).Max

	gtx.Constraints.Max = max

	if !b.set {
		dims := t.Layout(gtx)
		x := unit.Dp((float64(max.X) - float64(dims.Size.X)) / 2)
		y := unit.Dp((float64(max.Y) - float64(dims.Size.Y)) / 2)
		b.inset = layout.Inset{Left: x, Right: x, Top: y - b.TextInsetBottom, Bottom: y + b.TextInsetBottom}
		b.set = true
		return dims
	}

	b.inset.Layout(gtx, t.Layout)
	/*
		layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceEvenly, WeightSum: 3}.Layout(gtx,
			layout.Rigid(layout.Spacer{Width: 1}.Layout),
			layout.Rigid(t.Layout),
			layout.Rigid(layout.Spacer{Width: 1}.Layout),
		)
	*/
	return layout.Dimensions{Size: gtx.Constraints.Max}
}
