package split

import (
	"fmt"
	"image"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/gui/cursor"
)

type Widget interface {
	Layout(gtx layout.Context, left, right layout.Widget) layout.Dimensions
}

type Horizontal split
type Vertical split

type split struct {
	// ratio keeps the current layout.
	// 0 is center, -1 completely to the left, 1 completely to the right.
	Ratio   float32
	base    float32
	baseSet bool

	Adjustable bool

	// width for resizing the layout.
	bar unit.Dp

	drag         bool
	dragID       pointer.ID
	dragX, dragY float32
}

var (
	defaultBarSizeAdjustable = unit.Dp(5)
	defaultBarSize           = unit.Dp(0)
)

func NewHorizontal(ratio float32) *Horizontal {
	return &Horizontal{Ratio: ratio}
}

func NewVertical(ratio float32) *Vertical {
	return &Vertical{Ratio: ratio}
}

func (h *Horizontal) Layout(gtx layout.Context, top, bottom layout.Widget) layout.Dimensions {
	size := gtx.Dp(h.bar)
	if size <= 1 {
		size = gtx.Dp(defaultBarSize)
		if h.Adjustable {
			size = gtx.Dp(defaultBarSizeAdjustable)
		}
	}

	proportion := (h.Ratio + 1) / 2
	topSize := int(proportion*float32(gtx.Constraints.Max.Y) - float32(size))
	bottomOffset := topSize + size
	bottomSize := gtx.Constraints.Max.Y - bottomOffset

	if !h.baseSet {
		h.base = h.Ratio
		h.baseSet = true
	}

	if h.Adjustable {
		// handle input
		for _, ev := range gtx.Events(h) {
			e, ok := ev.(pointer.Event)
			if !ok {
				continue
			}

			switch e.Kind {
			case pointer.Enter:
				cursor.Is(pointer.CursorGrab)
			case pointer.Press:
				if h.drag {
					break
				}

				h.dragID = e.PointerID
				h.dragY = e.Position.Y

				cursor.Is(pointer.CursorGrab)
			case pointer.Drag:
				if h.dragID != e.PointerID {
					break
				}

				deltaY := e.Position.Y - h.dragY
				h.dragY = e.Position.Y

				deltaRatio := deltaY * 2 / float32(gtx.Constraints.Max.Y)
				h.Ratio += deltaRatio
				if h.Ratio > h.base {
					h.Ratio = h.base
				}

				fmt.Printf("%.5f\n", h.Ratio+deltaRatio)

			case pointer.Release:
				fallthrough
			case pointer.Cancel:
				cursor.Is(pointer.CursorGrab)

				h.drag = false
			}
		}

		// register for input
		barRect := image.Rect(0, topSize, gtx.Constraints.Max.X, bottomOffset)
		area := clip.Rect(barRect).Push(gtx.Ops)
		pointer.InputOp{
			Tag:   h,
			Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Enter,
			Grab:  h.drag,
		}.Add(gtx.Ops)
		area.Pop()
	}

	{
		gtx := gtx
		gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X, topSize))
		top(gtx)
	}

	{
		gtx := gtx
		barRect := image.Rect(0, topSize, gtx.Constraints.Max.X, bottomOffset)
		bg := clip.Rect(barRect).Push(gtx.Ops)
		paint.ColorOp{Color: nrgba.Splash.Color()}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		bg.Pop()
	}

	{
		off := op.Offset(image.Pt(0, bottomOffset)).Push(gtx.Ops)
		gtx := gtx
		gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X, bottomSize))
		bottom(gtx)
		off.Pop()
	}

	return layout.Dimensions{Size: gtx.Constraints.Max}
}

func (v *Vertical) Layout(gtx layout.Context, left, right layout.Widget) layout.Dimensions {
	barSize := gtx.Dp(v.bar)
	if barSize <= 1 {
		barSize = gtx.Dp(defaultBarSize)
		if v.Adjustable {
			barSize = gtx.Dp(defaultBarSizeAdjustable)
		}
	}

	proportion := (v.Ratio + 1) / 2
	leftsize := int(proportion*float32(gtx.Constraints.Max.X) - float32(barSize))
	rightoffset := leftsize + barSize
	rightsize := gtx.Constraints.Max.X - rightoffset

	if !v.baseSet {
		v.base = v.Ratio
		v.baseSet = true
	}

	if v.Adjustable {
		// handle input
		for _, ev := range gtx.Events(v) {
			e, ok := ev.(pointer.Event)
			if !ok {
				continue
			}

			switch e.Kind {
			case pointer.Press:
				if v.drag {
					break
				}

				v.dragID = e.PointerID
				v.dragX = e.Position.X

			case pointer.Drag:
				if v.dragID != e.PointerID {
					break
				}

				deltaX := e.Position.X - v.dragX
				v.dragX = e.Position.X

				deltaRatio := deltaX * 2 / float32(gtx.Constraints.Max.X)
				v.Ratio += deltaRatio

			case pointer.Release:
				fallthrough
			case pointer.Cancel:
				v.drag = false
			}
		}

		// register for input
		barRect := image.Rect(leftsize, 0, rightoffset, gtx.Constraints.Max.X)
		area := clip.Rect(barRect).Push(gtx.Ops)
		pointer.InputOp{Tag: v,
			Kinds: pointer.Press | pointer.Drag | pointer.Release,
			Grab:  v.drag,
		}.Add(gtx.Ops)
		area.Pop()
	}

	{
		gtx := gtx
		gtx.Constraints = layout.Exact(image.Pt(leftsize, gtx.Constraints.Max.Y))
		left(gtx)
	}

	{
		off := op.Offset(image.Pt((rightoffset), 0)).Push(gtx.Ops)
		gtx := gtx
		gtx.Constraints = layout.Exact(image.Pt(rightsize, gtx.Constraints.Max.Y))
		right(gtx)
		off.Pop()
	}

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
