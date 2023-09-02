package title

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/cursor"
	"github.com/pidgy/unitehud/fonts"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/decor"
	"github.com/pidgy/unitehud/img"
	"github.com/pidgy/unitehud/nrgba"
)

const (
	Default = "UniteHUD" // +" " +global.Version
)

var (
	Height = 26

	buttonSize     = image.Pt(Height+5, Height)
	buttonTextSize = unit.Sp(18)
	titleTextSize  = unit.Sp(16)
	tipTextSize    = unit.Sp(14)
)

type Bar struct {
	Title      string
	Collection fonts.Collection

	tip         string
	NoTip       bool
	Hide        bool
	HideButtons bool

	grabbed *bool

	*decorations
}

type decorations struct {
	title, tip material.LabelStyle
	titleSet   bool
	titleInset layout.Inset
	tipInset   layout.Inset

	icon      widget.Image
	iconSet   bool
	iconInset layout.Inset

	minimize, resize, close *button.Button
	customs                 []*button.Button
	customButtonsOpen       bool

	clicked time.Time

	drag struct {
		first, last, diff f32.Point
		dragging          bool
	}
}

func New(title string, collection fonts.Collection, minimize, resize, close func()) *Bar {
	b := &Bar{
		Title:      title,
		Collection: collection,

		grabbed: new(bool),

		decorations: &decorations{
			title: material.Label(collection.Calibri().Theme, titleTextSize, title),
			tip:   material.Label(collection.Calibri().Theme, tipTextSize, ""),
			icon: widget.Image{
				Src:   paint.NewImageOp(img.Icon("icon48x48")),
				Fit:   widget.ScaleDown,
				Scale: .27,
			},
			minimize: &button.Button{
				Text:         "-",
				TextSize:     buttonTextSize,
				Font:         collection.Cascadia(),
				Size:         buttonSize,
				Pressed:      nrgba.BackgroundAlt.Alpha(255),
				Released:     nrgba.PastelBlue,
				NoBorder:     true,
				SharpCorners: true,
				Disabled:     minimize == nil,
				Click: func(this *button.Button) {
					defer this.Deactivate()
					if minimize != nil {
						minimize()
					}
				},
			},
			resize: &button.Button{
				Text:         "□",
				TextSize:     buttonTextSize,
				Font:         collection.Cascadia(),
				Size:         buttonSize,
				Pressed:      nrgba.BackgroundAlt.Alpha(255),
				Released:     nrgba.PastelGreen,
				NoBorder:     true,
				SharpCorners: true,
				Disabled:     resize == nil,
				Click: func(this *button.Button) {
					defer this.Deactivate()

					if resize != nil {
						resize()
					}
				},
			},
			close: &button.Button{
				Text:         "×",
				TextSize:     buttonTextSize,
				Font:         collection.Cascadia(),
				Size:         buttonSize,
				Pressed:      nrgba.BackgroundAlt.Alpha(255),
				Released:     nrgba.PastelRed,
				NoBorder:     true,
				SharpCorners: true,

				TextInsetBottom: .5,
				Disabled:        close == nil,
				Click: func(this *button.Button) {
					defer this.Deactivate()
					if close != nil {
						close()
					}
				},
			},
		},
	}

	b.decorations.minimize.OnHoverHint = func() { b.ToolTip("Minimize") }
	b.decorations.resize.OnHoverHint = func() { b.ToolTip("Resize") }
	b.decorations.close.OnHoverHint = func() { b.ToolTip("Close") }

	b.customs = append(b.customs,
		&button.Button{
			Text:            "≡",
			TextSize:        unit.Sp(25),
			Size:            buttonSize,
			Font:            collection.NishikiTeki(),
			Released:        nrgba.Background,
			Pressed:         nrgba.BackgroundAlt.Alpha(255),
			NoBorder:        true,
			SharpCorners:    true,
			OnHoverHint:     func() { b.ToolTip("Additional options") },
			TextInsetBottom: 1,
			Click: func(this *button.Button) {
				defer this.Deactivate()

				b.customButtonsOpen = !b.customButtonsOpen

				this.Text = "≡"
				this.TextSize = unit.Sp(25)
				this.TextInsetBottom = 1
				if b.customButtonsOpen {
					this.Text = "🗦"
					this.TextSize = unit.Sp(20)
					this.TextInsetBottom = 0
				}
			},
		})

	return b
}

func (b *Bar) Add(btn *button.Button) *button.Button {
	if btn.Size.Eq(image.Pt(0, 0)) {
		btn.Size = buttonSize
	}
	if btn.TextSize == 0 {
		btn.TextSize = buttonTextSize
	}

	if btn.Pressed == nrgba.Transparent {
		btn.Pressed = nrgba.BackgroundAlt.Alpha(255)
	}

	btn.NoBorder = true
	btn.SharpCorners = true
	btn.TextInsetBottom++

	b.customs = append(b.customs, btn)

	return btn
}

func (b *Bar) Remove(btn *button.Button) {
	c := []*button.Button{}
	for _, b := range b.customs {
		if b.Text != btn.Text {
			c = append(c, b)
		}
	}
	b.customs = c

	if len(b.customs) == 1 && b.customButtonsOpen {
		b.customs[0].Click(b.customs[0])
	}
}

func (b *Bar) Dragging() (image.Point, bool) {
	if !b.drag.dragging {
		return image.Pt(0, 0), false
	}
	return b.drag.diff.Round(), true
}

func (b *Bar) Layout(gtx layout.Context, content layout.Widget) layout.Dimensions {
	cursor.Draw(gtx)

	for _, ev := range gtx.Events(b.grabbed) {
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}

		//cursor.Is(pointer.CursorDefault)
		if b.drag.dragging {
			cursor.Is(pointer.CursorPointer)
		}

		switch e.Type {
		case pointer.Enter, pointer.Move:
		case pointer.Press:
			b.drag.first = e.Position
		case pointer.Release:
			b.drag.dragging = false

			b.drag.last = e.Position
			b.drag.diff = b.drag.last.Sub(b.drag.first)

			if time.Since(b.decorations.clicked) < time.Millisecond*250 {
				defer b.decorations.resize.Deactivate()
				b.decorations.resize.Click(b.resize)
				break
			}
			b.decorations.clicked = time.Now()
		case pointer.Drag:
			if b.drag.last.Round().Eq(e.Position.Round()) {
				break
			}
			b.drag.last = e.Position

			b.drag.diff = b.drag.last.Sub(b.drag.first)

			b.drag.dragging = true
		}
	}

	bar := image.Rect(0, 0, gtx.Constraints.Max.X, Height)

	decor.ColorBox(gtx, gtx.Constraints.Max, nrgba.Background)

	dims := layout.Flex{
		Spacing:   layout.SpaceAround,
		Alignment: layout.Baseline,
		Axis:      layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if b.Hide {
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, Height/3)}
			}

			decor.ColorBox(gtx, image.Pt(gtx.Constraints.Max.X, Height), nrgba.Background)

			children := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !b.decorations.iconSet {
						b.decorations.iconSet = true

						idims := b.decorations.icon.Layout(gtx)

						y := unit.Dp((float64(bar.Max.Y) - float64(idims.Size.Y)) / 2)
						b.iconInset = layout.Inset{Left: 5, Top: y - 1, Bottom: y}
					}

					idims := b.iconInset.Layout(gtx, b.icon.Layout)

					if !b.decorations.titleSet {
						b.decorations.titleSet = true

						titleDims := b.decorations.title.Layout(gtx)

						b.decorations.title.Text = b.Title
						b.decorations.title.Color = nrgba.White.Color()
						b.decorations.title.Font.Weight = font.SemiBold
						y := unit.Dp((float64(bar.Max.Y) - float64(titleDims.Size.Y)))
						b.decorations.titleInset = layout.Inset{Left: 5 + unit.Dp(idims.Size.X), Top: y, Bottom: y}

						tipDims := b.decorations.tip.Layout(gtx)
						y = unit.Dp((float64(bar.Max.Y) - float64(tipDims.Size.Y)))
						b.decorations.tipInset = layout.Inset{Left: 5 + unit.Dp(idims.Size.X), Top: y - 1, Bottom: y + 1}
						b.tipInset.Left += unit.Dp(5 + titleDims.Size.X)
						b.decorations.tip.Color = nrgba.White.Alpha(100).Color()
						b.decorations.tip.Font.Weight = font.ExtraLight
						b.decorations.tip.Font.Style = font.Italic

						return layout.Dimensions{Size: titleDims.Size.Add(tipDims.Size)}
					}

					dims := b.decorations.titleInset.Layout(gtx, b.decorations.title.Layout)

					if !b.NoTip && b.tip != "" {
						b.decorations.tip.Text = b.tip
						dims = layout.Dimensions{Size: dims.Size.Add(b.tipInset.Layout(gtx, b.decorations.tip.Layout).Size)}
					}

					return layout.Dimensions{Size: layout.Exact(image.Pt(dims.Size.X+idims.Size.X+10, bar.Max.Y)).Max}
				}),

				layout.Flexed(.1, layout.Spacer{Width: unit.Dp(gtx.Constraints.Max.X)}.Layout),
			}

			if len(b.customs) > 1 {
				if b.customButtonsOpen {
					for i := range b.customs[1:] {
						btn := b.customs[len(b.customs)-1-i]

						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return btn.Layout(gtx)
						}))
					}
				}

				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return b.customs[0].Layout(gtx)
				}))
			}

			if !b.HideButtons {
				children = append(children,
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
			}

			b.decorations.title.Text = b.Title

			layout.Flex{
				Spacing:   layout.SpaceAround,
				Alignment: layout.Baseline,
				Axis:      layout.Horizontal,
			}.Layout(gtx, children...)

			return layout.Dimensions{Size: layout.Exact(bar.Size()).Max}
		}),

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if b.Hide {
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, Height/3)}
			}

			size := image.Rect(0, 0, gtx.Constraints.Max.X, 1).Max
			decor.ColorBox(gtx, size, nrgba.Gray)
			return layout.Dimensions{Size: size}
		}),

		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			b.tip = ""
			return content(gtx)
		}),
	)

	b.decorations.minimize.HoverHint()
	b.decorations.resize.HoverHint()
	b.decorations.close.HoverHint()

	customSizes := 0
	for _, btn := range b.customs {
		customSizes += btn.Size.X
		btn.HoverHint()

		if !b.customButtonsOpen {
			break
		}
	}

	defer clip.Rect(bar.Sub(image.Pt(b.minimize.Size.X+b.resize.Size.X+b.close.Size.X+customSizes, 0))).Push(gtx.Ops).Pop()
	pointer.InputOp{
		Tag:   b.grabbed,
		Types: pointer.Press | pointer.Drag | pointer.Release | pointer.Leave | pointer.Enter | pointer.Move,
		Grab:  true,
	}.Add(gtx.Ops)

	return dims
}

func (b *Bar) ToolTip(t string) {
	if t == "" {
		return
	}
	b.tip = t
}
