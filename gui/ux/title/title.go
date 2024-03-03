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

	"github.com/pidgy/unitehud/avi/img"
	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/fonts"
	"github.com/pidgy/unitehud/core/nrgba"
	"github.com/pidgy/unitehud/gui/cursor"
	"github.com/pidgy/unitehud/gui/ux"
	"github.com/pidgy/unitehud/gui/ux/button"
	"github.com/pidgy/unitehud/gui/ux/decorate"
)

var (
	Height = 26

	buttonTextSize = unit.Sp(18)
	titleTextSize  = unit.Sp(16)
	tipTextSize    = unit.Sp(14)
)

type Widget struct {
	Title      string
	Collection fonts.Collection

	NoTip,
	NoDrag,
	Hide,
	HideButtons bool

	*decorations
}

type decorations struct {
	title struct {
		material.LabelStyle
		set   bool
		inset layout.Inset
	}

	tip struct {
		material.LabelStyle
		set   bool
		inset layout.Inset
	}

	icon struct {
		widget.Image
		set   bool
		inset layout.Inset
	}

	buttons struct {
		minimize,
		resize,
		close *button.Widget

		custom []*button.Widget
		open   bool
	}

	clicked struct {
		last time.Time
	}

	dragging struct {
		is,
		was bool

		first,
		last,
		diff f32.Point
	}
}

func New(title string, collection fonts.Collection, minimize, resize, close func()) *Widget {
	b := &Widget{
		Title:      title,
		Collection: collection,

		decorations: &decorations{
			title: struct {
				material.LabelStyle
				set   bool
				inset layout.Inset
			}{
				LabelStyle: material.Label(collection.Calibri().Theme, titleTextSize, title),
			},
			tip: struct {
				material.LabelStyle
				set   bool
				inset layout.Inset
			}{
				LabelStyle: material.Label(collection.Calibri().Theme, tipTextSize, ""),
			},
			icon: struct {
				widget.Image
				set   bool
				inset layout.Inset
			}{
				Image: widget.Image{
					Src:   paint.NewImageOp(img.Icon("icon48x48.png")),
					Fit:   widget.ScaleDown,
					Scale: .27,
				},
			},
			buttons: struct {
				minimize *button.Widget
				resize   *button.Widget
				close    *button.Widget
				custom   []*button.Widget
				open     bool
			}{
				minimize: &button.Widget{
					Text:         "-",
					TextSize:     buttonTextSize,
					Font:         collection.Cascadia(),
					Size:         button.IconSize,
					Pressed:      nrgba.NRGBA(config.Current.Theme.BackgroundAlt).Alpha(255),
					Released:     nrgba.PastelBlue,
					NoBorder:     true,
					SharpCorners: true,
					Disabled:     minimize == nil,
					Click: func(this *button.Widget) {
						defer this.Deactivate()
						if minimize != nil {
							minimize()
						}
					},
				},
				resize: &button.Widget{
					Text:         "□",
					TextSize:     buttonTextSize,
					Font:         collection.Cascadia(),
					Size:         button.IconSize,
					Pressed:      nrgba.NRGBA(config.Current.Theme.BackgroundAlt).Alpha(255),
					Released:     nrgba.PastelGreen,
					NoBorder:     true,
					SharpCorners: true,
					Disabled:     resize == nil,
					Click: func(this *button.Widget) {
						defer this.Deactivate()
						if resize != nil {
							resize()
						}
					},
				},
				close: &button.Widget{
					Text:         "×",
					TextSize:     buttonTextSize,
					Font:         collection.Cascadia(),
					Size:         button.IconSize,
					Pressed:      nrgba.NRGBA(config.Current.Theme.BackgroundAlt).Alpha(255),
					Released:     nrgba.PastelRed,
					NoBorder:     true,
					SharpCorners: true,

					TextInsetBottom: .5,
					Disabled:        close == nil,
					Click: func(this *button.Widget) {
						defer this.Deactivate()
						if close != nil {
							close()
						}
					},
				},
			},
		},
	}

	b.decorations.buttons.minimize.OnHoverHint = func() { b.Tip("Minimize") }
	b.decorations.buttons.resize.OnHoverHint = func() { b.Tip("Resize") }
	b.decorations.buttons.close.OnHoverHint = func() { b.Tip("Close") }

	b.decorations.buttons.custom = append(b.decorations.buttons.custom,
		&button.Widget{
			Text:            "≡",
			TextSize:        unit.Sp(25),
			Size:            button.IconSize,
			Font:            collection.NishikiTeki(),
			Released:        nrgba.NRGBA(config.Current.Theme.Background),
			Pressed:         nrgba.NRGBA(config.Current.Theme.BackgroundAlt).Alpha(255),
			NoBorder:        true,
			SharpCorners:    true,
			OnHoverHint:     func() { b.Tip("Additional options") },
			TextInsetBottom: 1,
			Click: func(this *button.Widget) {
				defer this.Deactivate()

				b.decorations.buttons.open = !b.decorations.buttons.open

				this.Text = "≡"
				this.TextSize = unit.Sp(25)
				this.TextInsetBottom = 1
				if b.decorations.buttons.open {
					this.Text = "🗦"
					this.TextSize = unit.Sp(20)
					this.TextInsetBottom = 0
				}
			},
		},
	)

	b.decorations.title.Font.Weight = font.Normal
	b.decorations.tip.Font.Weight = font.ExtraLight
	b.decorations.tip.Font.Style = font.Italic

	return b
}

func (b *Widget) Add(btn *button.Widget) *button.Widget {
	if btn.Size.Eq(image.Pt(0, 0)) {
		btn.Size = button.IconSize
	}
	if btn.TextSize == 0 {
		btn.TextSize = buttonTextSize
	}

	if btn.Pressed == nrgba.Transparent {
		btn.Pressed = nrgba.NRGBA(config.Current.Theme.BackgroundAlt).Alpha(255)
	}

	btn.NoBorder = true
	btn.SharpCorners = true
	btn.TextInsetBottom++

	b.decorations.buttons.custom = append(b.decorations.buttons.custom, btn)

	return btn
}

func (b *Widget) Dragging() (image.Point, bool) {
	if !b.dragging.is {
		return image.Pt(0, 0), false
	}
	return b.dragging.diff.Round(), true
}

func (b *Widget) Layout(gtx layout.Context, content layout.Widget) layout.Dimensions {
	cursor.Draw(gtx)

	for _, ev := range gtx.Events(b) {
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}

		if b.dragging.is {
			if b.NoDrag {
				cursor.Is(pointer.CursorNotAllowed)
			} else {
				cursor.Is(pointer.CursorPointer)
			}
			b.dragging.was = true
		} else if b.dragging.was {
			b.dragging.was = false
			cursor.Is(pointer.CursorDefault)
		}

		switch e.Kind {
		case pointer.Enter, pointer.Move:
		case pointer.Press:
			b.dragging.first = e.Position
		case pointer.Release:
			b.dragging.is = false

			b.dragging.last = e.Position
			b.dragging.diff = b.dragging.last.Sub(b.dragging.first)

			if time.Since(b.decorations.clicked.last) < time.Millisecond*250 {
				defer b.decorations.buttons.resize.Deactivate()
				b.decorations.buttons.resize.Click(b.decorations.buttons.resize)
				break
			}
			b.decorations.clicked.last = time.Now()
		case pointer.Drag:
			if b.dragging.last.Round().Eq(e.Position.Round()) {
				break
			}
			b.dragging.last = e.Position

			b.dragging.diff = b.dragging.last.Sub(b.dragging.first)

			b.dragging.is = true
		}
	}

	bar := image.Rect(0, 0, gtx.Constraints.Max.X, Height)

	dims := layout.Flex{
		Spacing:   layout.SpaceAround,
		Alignment: layout.Baseline,
		Axis:      layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if b.Hide {
				// return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, Height/3)}
				return layout.Dimensions{}
			}

			decorate.BackgroundTitleBar(gtx, image.Pt(gtx.Constraints.Max.X, Height))

			children := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !b.decorations.icon.set {
						b.decorations.icon.set = true

						idims := b.decorations.icon.Layout(gtx)

						y := unit.Dp((float64(bar.Max.Y) - float64(idims.Size.Y)) / 2)
						b.icon.inset = layout.Inset{Left: 5, Top: y - 1, Bottom: y}
					}

					iconDims := b.icon.inset.Layout(gtx, b.icon.Layout)

					decorate.LabelColor(&b.decorations.title.LabelStyle, config.Current.Theme.TitleBarForeground)
					decorate.LabelColor(&b.decorations.tip.LabelStyle, config.Current.Theme.ForegroundAlt)

					if !b.decorations.title.set {
						b.decorations.title.set = true

						titleDims := b.decorations.title.Layout(gtx)

						y := unit.Dp((float64(bar.Max.Y) - float64(titleDims.Size.Y)))
						b.decorations.title.inset = layout.Inset{Left: 5 + unit.Dp(iconDims.Size.X), Top: y, Bottom: y}

						tipDims := b.decorations.tip.Layout(gtx)
						b.decorations.tip.inset = layout.Inset{
							Left:   unit.Dp(12 + iconDims.Size.X + titleDims.Size.X),
							Top:    2,
							Bottom: 0,
						}

						return layout.Dimensions{Size: titleDims.Size.Add(tipDims.Size)}
					}

					dims := b.decorations.title.inset.Layout(gtx, b.decorations.title.Layout)
					if !b.NoTip && b.decorations.tip.Text != "" {
						b.decorations.tip.Text = "🗧" + b.decorations.tip.Text
						dims = layout.Dimensions{Size: dims.Size.Add(b.decorations.tip.inset.Layout(gtx, b.decorations.tip.Layout).Size)}
					}

					return layout.Dimensions{Size: layout.Exact(image.Pt(dims.Size.X+iconDims.Size.X+10, bar.Max.Y)).Max}
				}),

				layout.Flexed(.1, layout.Spacer{Width: unit.Dp(gtx.Constraints.Max.X)}.Layout),
			}

			if len(b.decorations.buttons.custom) > 1 {
				if b.decorations.buttons.open {
					for i := range b.decorations.buttons.custom[1:] {
						btn := b.decorations.buttons.custom[len(b.decorations.buttons.custom)-1-i]

						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return btn.Layout(gtx)
						}))
					}
				}

				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return b.decorations.buttons.custom[0].Layout(gtx)
				}))
			}

			if !b.HideButtons {
				children = append(children,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return b.decorations.buttons.minimize.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return b.decorations.buttons.resize.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return b.decorations.buttons.close.Layout(gtx)
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
				return layout.Dimensions{}
				// return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, Height/3)}
			}
			return decorate.BorderIdle(gtx)
		}),

		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			b.decorations.tip.Text = ""
			return content(gtx)
		}),
	)

	b.decorations.buttons.minimize.HoverHint()
	b.decorations.buttons.resize.HoverHint()
	b.decorations.buttons.close.HoverHint()

	customSizes := button.IconSize.Mul(3) // Min, Max, Close.
	customSizes.Y = 0
	for _, btn := range b.decorations.buttons.custom {
		customSizes.X += btn.Size.X
		btn.HoverHint()

		if !b.decorations.buttons.open {
			break
		}
	}

	defer clip.Rect(bar.Sub(customSizes)).Push(gtx.Ops).Pop()
	pointer.InputOp{
		Tag:   b,
		Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Leave | pointer.Enter | pointer.Move,
		Grab:  true,
	}.Add(gtx.Ops)

	return dims
}

func (b *Widget) OnClose(fn func(*button.Widget)) ux.Thener {
	tmp := b.buttons.close.Click
	b.buttons.close.Click = fn
	return ux.Then{T: func() { b.buttons.close.Click = tmp }}
}

func (b *Widget) Open() {
	this := b.decorations.buttons.custom[0]
	this.Click(this)
}

func (b *Widget) Remove(btn *button.Widget) {
	c := []*button.Widget{}
	for _, b := range b.decorations.buttons.custom {
		if b.Text != btn.Text {
			c = append(c, b)
		}
	}
	b.decorations.buttons.custom = c

	if len(b.decorations.buttons.custom) == 1 && b.decorations.buttons.open {
		b.decorations.buttons.custom[0].Click(b.decorations.buttons.custom[0])
	}
}

func (b *Widget) Resize() {
	b.buttons.resize.Click(b.buttons.resize)
}

func (b *Widget) Tip(t string) {
	if t == "" {
		return
	}
	b.decorations.tip.Text = t
}
