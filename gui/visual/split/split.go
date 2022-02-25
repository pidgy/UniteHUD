package split

import (
	"image"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
)

type Split interface {
	Layout(gtx layout.Context, left, right layout.Widget) layout.Dimensions
}

type Horizontal split
type Vertical split

type split struct {
	// ratio keeps the current layout.
	// 0 is center, -1 completely to the left, 1 completely to the right.
	Ratio float32

	Adjustable bool

	// width for resizing the layout.
	bar unit.Value

	drag         bool
	dragID       pointer.ID
	dragX, dragY float32
}

var defaultBarWidth = unit.Dp(0)

func (h *Horizontal) Layout(gtx layout.Context, top, bottom layout.Widget) layout.Dimensions {
	bar := gtx.Px(h.bar)
	if bar <= 1 {
		bar = gtx.Px(defaultBarWidth)
	}

	proportion := (h.Ratio + 1) / 2
	topSize := int(proportion*float32(gtx.Constraints.Max.Y) - float32(bar))

	bottomOffset := topSize + bar
	bottomSize := gtx.Constraints.Max.Y - bottomOffset

	if h.Adjustable {
		// handle input
		for _, ev := range gtx.Events(h) {
			e, ok := ev.(pointer.Event)
			if !ok {
				continue
			}

			switch e.Type {
			case pointer.Enter:
				pointer.CursorNameOp{Name: pointer.CursorGrab}.Add(gtx.Ops)
			case pointer.Press:
				if h.drag {
					break
				}

				h.dragID = e.PointerID
				h.dragY = e.Position.Y

				pointer.CursorNameOp{Name: pointer.CursorGrab}.Add(gtx.Ops)
			case pointer.Drag:
				if h.dragID != e.PointerID {
					break
				}

				deltaY := e.Position.Y - h.dragY
				h.dragY = e.Position.Y

				deltaRatio := deltaY * 2 / float32(gtx.Constraints.Max.Y)
				h.Ratio += deltaRatio

			case pointer.Release:
				fallthrough
			case pointer.Cancel:
				pointer.CursorNameOp{Name: pointer.CursorGrab}.Add(gtx.Ops)
				h.drag = false
			}
		}

		// register for input
		barRect := image.Rect(0, topSize, gtx.Constraints.Max.X, bottomOffset)
		area := clip.Rect(barRect).Push(gtx.Ops)
		pointer.InputOp{Tag: h,
			Types: pointer.Press | pointer.Drag | pointer.Release | pointer.Enter,
			Grab:  h.drag,
		}.Add(gtx.Ops)
		area.Pop()
	}

	{
		gtx := gtx
		gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X, topSize))
		top(gtx)
	}

	/*
		{
			gtx := gtx
			barRect := image.Rect(0, topSize, gtx.Constraints.Max.X, bottomOffset)
			bg := clip.Rect(barRect).Push(gtx.Ops)
			paint.ColorOp{Color: color.NRGBA{R: 255, G: 255, B: 255, A: 255}}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			bg.Pop()
		}
	*/

	{
		off := op.Offset(f32.Pt(0, float32(bottomOffset))).Push(gtx.Ops)
		gtx := gtx
		gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X, bottomSize))
		bottom(gtx)
		off.Pop()
	}

	return layout.Dimensions{Size: gtx.Constraints.Max}
}

func (v *Vertical) Layout(gtx layout.Context, left, right layout.Widget) layout.Dimensions {
	bar := gtx.Px(v.bar)
	if bar <= 1 {
		bar = gtx.Px(defaultBarWidth)
	}

	proportion := (v.Ratio + 1) / 2
	leftsize := int(proportion*float32(gtx.Constraints.Max.X) - float32(bar))

	rightoffset := leftsize + bar
	rightsize := gtx.Constraints.Max.X - rightoffset

	if v.Adjustable {
		// handle input
		for _, ev := range gtx.Events(v) {
			e, ok := ev.(pointer.Event)
			if !ok {
				continue
			}

			switch e.Type {
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
			Types: pointer.Press | pointer.Drag | pointer.Release,
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
		off := op.Offset(f32.Pt(float32(rightoffset), 0)).Push(gtx.Ops)
		gtx := gtx
		gtx.Constraints = layout.Exact(image.Pt(rightsize, gtx.Constraints.Max.Y))
		right(gtx)
		off.Pop()
	}

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
