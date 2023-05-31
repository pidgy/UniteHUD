package gui

import (
	"image"
	"runtime"
	"sort"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/gui/visual/area"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/screen"
	"github.com/pidgy/unitehud/nrgba"
	"github.com/pidgy/unitehud/video"
)

var previewCapturesOpen = false

func (g *GUI) previewCaptures(captures []*area.Capture) {
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		if previewCapturesOpen {
			return
		}
		previewCapturesOpen = true
		defer func() { previewCapturesOpen = false }()

		// Order by widest.
		sort.Slice(captures, func(i, j int) bool {
			return captures[i].Base.Dy() > captures[j].Base.Dy()
		})

		images := []*button.Image{}

		for _, cap := range captures {
			img, err := video.CaptureRect(cap.Base)
			if err != nil {
				g.ToastError(err)
				return
			}

			images = append(images, g.makePreviewCaptureButton(cap, img))
		}

		w := app.NewWindow(
			app.Title("UniteHUD Capture Preview"),
			app.Size(unit.Dp(852), unit.Dp(480)),
			app.MinSize(unit.Dp(852), unit.Dp(480)),
		)

		go func() {
			for ; previewCapturesOpen; time.Sleep(time.Second) {
				for i, cap := range captures {
					img, err := video.CaptureRect(cap.Base)
					if err != nil {
						g.ToastError(err)
						return
					}

					images[i].SetImage(img)
				}
			}
		}()

		list := &widget.List{
			Scrollbar: widget.Scrollbar{},
			List: layout.List{
				Axis:      layout.Vertical,
				Alignment: layout.Baseline,
			},
		}
		style := material.List(g.normal, list)
		style.Track.Color = nrgba.Gray.Color()

		var ops op.Ops

		for event := range w.Events() {
			switch e := event.(type) {
			case system.DestroyEvent:
				return
			case system.StageEvent:
				w.Perform(system.ActionCenter)
				w.Perform(system.ActionRaise)
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)

				ops.Reset()

				colorBox(gtx, gtx.Constraints.Max, nrgba.DarkGray)

				layout.Flex{
					Spacing:   layout.SpaceEnd,
					Alignment: layout.Baseline,
					Axis:      layout.Vertical,
				}.Layout(gtx,
					// Top vertical spacer.
					g.drawPreviewFlexChildSpacer(gtx),

					// Top vertical.
					layout.Flexed(0.5, func(gtx layout.Context) layout.Dimensions {
						return g.drawHorizontalPreview(gtx, images, captures, 0, 1, 2)
					}),

					// Center/Bottom vertical spacer.
					g.drawPreviewFlexChildSpacer(gtx),

					// Bottom vertical.
					layout.Flexed(0.5, func(gtx layout.Context) layout.Dimensions {
						return g.drawHorizontalPreview(gtx, images, captures, 3, 4, 5, 6)
					}),
				)

				/*
					for i := range images {
						if captures[i].Base.Dx() > 1000 {
							continue
						}

						base := captures[i].Base.Sub(captures[i].Base.Max.Div(2))

						s := clip.Rect(base).Push(gtx.Ops)

						widget.Image{
							Src:      paint.NewImageOp(images[i]),
							Position: layout.Center,
							Fit:      widget.ScaleDown,
						}.Layout(gtx)

						paint.PaintOp{}.Add(gtx.Ops)
						s.Pop()
					}
				*/
				w.Invalidate()

				e.Frame(gtx.Ops)
			}
		}
	}()
}

func (g *GUI) drawHorizontalPreview(gtx layout.Context, images []*button.Image, captures []*area.Capture, indicies ...int) layout.Dimensions {
	children := append([]layout.FlexChild{}, g.drawPreviewFlexChildSpacer(gtx))
	for _, i := range indicies {
		if i >= len(images) || i >= len(captures) {
			break
		}

		children = append(children,
			g.drawPreviewFlexChild(gtx, images[i], captures[i]),
			g.drawPreviewFlexChildSpacer(gtx),
		)
	}
	children = append(children, g.drawPreviewFlexChildSpacer(gtx))

	layout.Flex{
		Spacing:   layout.SpaceEnd,
		Alignment: layout.Baseline,
		Axis:      layout.Horizontal,
	}.Layout(gtx, children...)

	return dimensions(gtx)
}

func (g *GUI) drawPreviewFlexChild(gtx layout.Context, img *button.Image, cap *area.Capture) layout.FlexChild {
	return layout.Flexed(0.3, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Flexed(.1, g.drawPreviewLabel(gtx, cap)),
			layout.Flexed(.9, g.drawPreviewImage(gtx, img, cap)),
		)
	})
}

func (g *GUI) drawPreviewFlexChildSpacer(gtx layout.Context) layout.FlexChild {
	return layout.Rigid(layout.Spacer{Width: unit.Dp(5), Height: unit.Dp(5)}.Layout)
}

func (g *GUI) drawPreviewImage(gtx layout.Context, img *button.Image, cap *area.Capture) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return img.Layout(g.normal, gtx)
		})
	}
}

func (g *GUI) drawPreviewHeader(gtx layout.Context, txt string) layout.Dimensions {
	label := material.H5(g.normal, txt)
	label.Color = nrgba.White.Color()
	return layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return label.Layout(gtx) })
}

func (g *GUI) drawPreviewLabel(gtx layout.Context, cap *area.Capture) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		label := material.Label(g.normal, unit.Sp(18), cap.Option)
		label.Color = nrgba.Highlight.Color()
		label.Font.Weight = 200

		if cap.Matched != nil {
			label.Text = cap.Matched.Text
			label.Color = cap.Matched.Color()
		}

		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return label.Layout(gtx)
			})
		})
	}
}

func (g *GUI) makePreviewCaptureButton(cap *area.Capture, img image.Image) *button.Image {
	return &button.Image{
		Screen: &screen.Screen{
			Image:       img,
			Border:      true,
			BorderColor: nrgba.Transparent,
			AutoScale:   true,
		},
		Click: func(i *button.Image) {
			err := cap.Open()
			if err != nil {
				g.ToastError(err)
			}
		},
	}
}

func dimensions(gtx layout.Context) layout.Dimensions {
	return layout.Dimensions{Size: gtx.Constraints.Max}
}
