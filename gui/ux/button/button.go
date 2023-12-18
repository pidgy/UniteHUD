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

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/fonts"
	"github.com/pidgy/unitehud/core/nrgba"
	"github.com/pidgy/unitehud/gui/cursor"
)

var (
	DefaultSize = image.Pt(100, 35)
	IconSize    = image.Pt(31, 26)
)

type Widget struct {
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

	Click       func(this *Widget)
	SingleClick bool // Toggle the Active field on Click events.

	hover bool
	alpha uint8

	inset layout.Inset
	set   bool
}

func (b *Widget) Activate() {
	b.active = true
}

func (b *Widget) Deactivate() {
	b.active = false
}

func (b *Widget) Error() {
	tmp := b.Pressed
	b.Pressed = nrgba.Red
	b.Disabled = true
	time.AfterFunc(time.Second*2, func() {
		b.Pressed = tmp
		b.Disabled = false
	})
}

func (b *Widget) Layout(gtx layout.Context) layout.Dimensions {
	defer b.HoverHint()

	if b.Size.Eq(image.Pt(0, 0)) {
		b.Size = DefaultSize
	} else if b.Size.Eq(IconSize) {
		b.TextColor = nrgba.White
	} else {
		b.TextColor = nrgba.NRGBA(config.Current.Theme.Foreground)
	}

	if b.BorderWidth == 0 && !b.NoBorder {
		b.BorderWidth = unit.Sp(.5)
	}

	if b.alpha == 0 {
		b.alpha = b.Released.A
	}

	for _, e := range gtx.Events(b) {
		if e, ok := e.(pointer.Event); ok {
			switch e.Kind {
			case pointer.Cancel:
				cursor.Is(pointer.CursorDefault)

				b.hover = false
				b.Deactivate()
			case pointer.Enter:
				b.hover = true

				if b.Disabled {
					cursor.Is(pointer.CursorNotAllowed)
					continue
				}
				cursor.Is(pointer.CursorPointer)
			case pointer.Release:
				if b.Disabled {
					cursor.Is(pointer.CursorNotAllowed)
					continue
				}
				cursor.Is(pointer.CursorPointer)

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

				cursor.Is(pointer.CursorDefault)

				if b.Disabled {
					continue
				}

				b.Deactivate()
			case pointer.Press:
				b.hover = true

				if b.Disabled {
					cursor.Is(pointer.CursorNotAllowed)
					continue
				}

				b.Activate()
			case pointer.Move:
				b.hover = true

				if b.Disabled {
					cursor.Is(pointer.CursorNotAllowed)
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
		Kinds: pointer.Press | pointer.Release | pointer.Enter | pointer.Leave | pointer.Move | pointer.Cancel,
		Grab:  true,
	}.Add(gtx.Ops)
	area.Pop()

	return b.draw(gtx)
}

func (b *Widget) HoverHint() {
	if b.hover && b.OnHoverHint != nil {
		b.OnHoverHint()
	}
}

func (b *Widget) uniform(gtx layout.Context) layout.Dimensions {
	rect := clip.RRect{SE: 3, SW: 3, NE: 3, NW: 3, Rect: image.Rectangle{Max: image.Pt((b.Size.X), b.Size.Y)}}
	if b.SharpCorners {
		rect = clip.RRect{
			SE: 0,
			SW: 0,
			NE: 0,
			NW: 0,
			Rect: image.Rectangle{
				Max: image.Pt((b.Size.X), b.Size.Y),
			},
		}
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

func (b *Widget) draw(gtx layout.Context) layout.Dimensions {
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
			Color:        nrgba.NRGBA(config.Current.Theme.Borders).Alpha(50).Color(),
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
	} else if !b.active && b.Click == nil {
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

	b.inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, b.label.Layout)
	})

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
