package button

import (
	"image"
	"time"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/cursor"
	"github.com/pidgy/unitehud/fonts"
	"github.com/pidgy/unitehud/nrgba"
)

type Button struct {
	Text            string
	TextSize        unit.Sp
	TextInsetBottom unit.Dp
	TextColor       nrgba.NRGBA
	lastText        string

	label material.LabelStyle

	Hint        string
	OnHoverHint func()
	Font        *fonts.Style

	BorderWidth  unit.Sp
	NoBorder     bool
	SharpCorners bool

	Size image.Point

	active bool

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

func (b *Button) Activate() {
	b.active = true
}

func (b *Button) Deactivate() {
	b.active = false
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
	defer b.HoverHint()

	if b.Size.Eq(image.Pt(0, 0)) {
		b.Size = Max
	}

	if b.alpha == 0 {
		b.alpha = b.Released.A
	}

	if b.TextColor == nrgba.Transparent {
		b.TextColor = nrgba.White
	}

	for _, e := range gtx.Events(b) {
		if e, ok := e.(pointer.Event); ok {
			switch e.Type {
			case pointer.Cancel:
				b.hover = false
				b.Deactivate()
			case pointer.Enter:
				b.hover = true

				if b.Disabled {
					continue
				}
			case pointer.Release:
				if b.Disabled {
					continue
				}

				if b.hover && b.Click != nil {
					b.Click(b)
					if b.SingleClick {
						b.Activate()
					}
				} else {
					b.Deactivate()
				}
			case pointer.Leave:
				b.hover = false

				if b.Disabled {
					continue
				}

				b.Deactivate()
			case pointer.Press:
				b.hover = true

				if b.Disabled {
					continue
				}

				b.Activate()
			case pointer.Move:
				b.hover = true
			}
		}
	}

	switch {
	case b.hover && !b.Disabled:
		cursor.Is(pointer.CursorPointer)
	case b.hover && b.Disabled:
		cursor.Is(pointer.CursorNotAllowed)
	}

	// Confine the area for pointer events.
	area := clip.Rect(image.Rect(0, 0, b.Size.X, b.Size.Y)).Push(gtx.Ops)
	pointer.InputOp{
		Tag:   b,
		Types: pointer.Press | pointer.Release | pointer.Enter | pointer.Leave | pointer.Move | pointer.Cancel,
		Grab:  true,
	}.Add(gtx.Ops)
	area.Pop()

	return b.draw(gtx)
}

func (b *Button) HoverHint() {
	if b.hover && b.OnHoverHint != nil {
		b.OnHoverHint()
	}
}

func (b *Button) uniform(gtx layout.Context) layout.Dimensions {
	rect := clip.RRect{SE: 3, SW: 3, NE: 3, NW: 3, Rect: image.Rectangle{Max: image.Pt((b.Size.X), b.Size.Y)}}
	if b.SharpCorners {
		rect = clip.RRect{SE: 0, SW: 0, NE: 0, NW: 0, Rect: image.Rectangle{Max: image.Pt((b.Size.X), b.Size.Y)}}
	}

	defer rect.Push(gtx.Ops).Pop()

	col := b.Pressed
	if !b.active {
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

	if b.Disabled {
		widget.Border{
			Color:        nrgba.Disabled.Color(),
			Width:        unit.Dp(b.BorderWidth),
			CornerRadius: unit.Dp(2),
		}.Layout(gtx, b.uniform)
	} else {
		widget.Border{
			Color:        nrgba.White.Alpha(25).Color(),
			Width:        unit.Dp(b.BorderWidth),
			CornerRadius: unit.Dp(2),
		}.Layout(gtx, b.uniform)
	}

	if !b.set {
		b.label = material.Label(b.Font.Theme, b.TextSize, b.Text)
	}

	b.label.Text = b.Text
	b.label.TextSize = b.TextSize
	b.label.Color = b.TextColor.Color()
	b.label.MaxLines = 1
	b.label.Truncator = b.label.Text

	if b.active && b.Click != nil {
		b.label.Color.A = 0xFF
	}

	if !b.active && b.Click == nil {
		b.label.Color.A = 0x3F
	}

	max := layout.Exact(b.Size).Max

	gtx.Constraints.Max = max

	if b.Text != b.lastText {
		b.set = false
	}
	b.lastText = b.Text

	if !b.set {
		dims := b.label.Layout(gtx)
		x := unit.Dp((float64(max.X) - float64(dims.Size.X)) / 2)
		y := unit.Dp((float64(max.Y) - float64(dims.Size.Y)) / 2)
		b.inset = layout.Inset{Left: x, Top: y - b.TextInsetBottom}
		b.set = true

		//return dims
	}

	b.inset.Layout(gtx, b.label.Layout)

	// if b.hover {
	// 	gtx.Constraints.Min.Y -= 100
	// 	component.DesktopTooltip(b.Font.Theme, "This is a tooltip").Layout(gtx)
	// }

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
