package button

import (
	"image"
	"image/color"
	"time"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/pidgy/unitehud/rgba"
)

type CircleButton struct {
	Name string

	BorderColor color.NRGBA
	BorderWidth unit.Value

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

func (c *CircleButton) Deactivate() {
	c.Active = !c.Active
}

func (c *CircleButton) Error() {
	tmp := c.Pressed
	c.Pressed = color.NRGBA(rgba.Red)
	c.Disabled = true
	time.AfterFunc(time.Second*2, func() {
		c.Pressed = tmp
		c.Disabled = false
	})
}

func (c *CircleButton) Layout(gtx layout.Context) layout.Dimensions {
	if c.Size.Eq(image.Pt(0, 0)) {
		c.Size = Max
	}

	if c.alpha == 0 {
		c.alpha = c.Released.A
	}

	for _, e := range gtx.Events(c) {
		if e, ok := e.(pointer.Event); ok {
			switch e.Type {
			case pointer.Enter:
				c.Released = color.NRGBA(rgba.Alpha(color.RGBA(c.Released), 0x50))
				c.hover = true
			case pointer.Release:
				if c.hover && c.Click != nil {
					c.Click()
					if c.SingleClick {
						c.Active = !c.Active
					}
				} else {
					c.Active = !c.Active
				}
			case pointer.Leave:
				c.Released = color.NRGBA(rgba.Alpha(color.RGBA(c.Released), c.alpha))
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
		Rect: f32.Rectangle{
			Max: f32.Pt(float32(c.Size.X), float32(c.Size.Y))},
	}.Push(gtx.Ops).Pop()

	col := c.Pressed
	if !c.Active {
		col = c.Released
	}

	paint.ColorOp{Color: col}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: c.Size}
}

func (c *CircleButton) draw(gtx layout.Context) layout.Dimensions {
	if c.BorderWidth.V == unit.Px(0).V {
		c.BorderWidth = unit.Px(2)
	}

	if c.hover {
		return widget.Border{
			Color:        c.BorderColor,
			Width:        unit.Dp(1),
			CornerRadius: unit.Px(2),
		}.Layout(gtx, c.uniform)
	} else {
		return widget.Border{
			Color:        c.BorderColor,
			Width:        c.BorderWidth,
			CornerRadius: unit.Px(2),
		}.Layout(gtx, c.uniform)
	}
}
