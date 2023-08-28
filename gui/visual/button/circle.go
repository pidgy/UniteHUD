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

	"github.com/pidgy/unitehud/nrgba"
)

type CircleButton struct {
	Name string

	BorderColor nrgba.NRGBA
	BorderWidth unit.Sp

	Size image.Point

	Active            bool
	Disabled          bool
	LastPressed       time.Time
	Pressed, Released nrgba.NRGBA

	Click       func(b *CircleButton)
	SingleClick bool // Toggle the Active field on Click events.

	hover bool
	alpha uint8
}

func (c *CircleButton) Deactivate() {
	c.Active = !c.Active
}

func (c *CircleButton) Error() {
	tmp := c.Pressed
	c.Pressed = nrgba.Red
	c.Disabled = true
	time.AfterFunc(time.Second*2, func() {
		c.Pressed = tmp
		c.Disabled = false
	})
}

func (c *CircleButton) Layout(gtx layout.Context) layout.Dimensions {
	if c.Size.Eq(image.Pt(0, 0)) {
		c.Size = Default
	}

	if c.alpha == 0 {
		c.alpha = c.Released.A
	}

	for _, e := range gtx.Events(c) {
		if e, ok := e.(pointer.Event); ok {
			switch e.Type {
			case pointer.Enter:
				c.Released = c.Released.Alpha(0x50)
				c.hover = true
			case pointer.Release:
				if c.hover && c.Click != nil {
					c.Click(c)
					if c.SingleClick {
						c.Active = !c.Active
					}
				} else {
					c.Active = !c.Active
				}
			case pointer.Leave:
				c.Released = c.Released.Alpha(c.alpha)
				c.hover = false
			case pointer.Press:
				c.Active = !c.Active
			case pointer.Move:
			}
		}
	}

	// Confine the area for pointer events.
	if !c.Disabled {
		area := clip.Rect(image.Rect(0, 0, c.Size.X, c.Size.Y)).Push(gtx.Ops)
		pointer.InputOp{
			Tag:   c,
			Types: pointer.Press | pointer.Release | pointer.Enter | pointer.Leave,
		}.Add(gtx.Ops)

		area.Pop()
	} else {
		c.hover = false
	}

	return c.draw(gtx)
}

func (c *CircleButton) uniform(gtx layout.Context) layout.Dimensions {
	defer clip.RRect{
		SE: 5, SW: 5, NE: 5, NW: 5,
		Rect: image.Rectangle{
			Max: image.Pt((c.Size.X), c.Size.Y)},
	}.Push(gtx.Ops).Pop()

	n := c.Pressed
	if !c.Active {
		n = c.Released
	}

	paint.ColorOp{Color: n.Color()}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: c.Size}
}

func (c *CircleButton) draw(gtx layout.Context) layout.Dimensions {
	if c.BorderWidth == unit.Sp(0) {
		c.BorderWidth = unit.Sp(2)
	}

	if c.hover {
		return widget.Border{
			Color:        c.BorderColor.Color(),
			Width:        unit.Dp(1),
			CornerRadius: unit.Dp(2),
		}.Layout(gtx, c.uniform)
	} else {
		return widget.Border{
			Color:        c.BorderColor.Color(),
			Width:        unit.Dp(c.BorderWidth),
			CornerRadius: unit.Dp(2),
		}.Layout(gtx, c.uniform)
	}
}
