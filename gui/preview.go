package gui

import (
	"fmt"
	"image"
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
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/nrgba"
	"github.com/pidgy/unitehud/video"
)

var previewCapturesOpen = false

func (g *GUI) previewCaptures(a *areas) {
	if previewCapturesOpen {
		return
	}
	previewCapturesOpen = true
	defer func() { previewCapturesOpen = false }()

	captures := []*area.Capture{
		a.energy.Capture,
		a.ko.Capture,
		a.objective.Capture,
		a.score.Capture,
		a.screen,
		a.state.Capture,
		a.time.Capture,
	}

	// Ordered by widest.
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

	header := fmt.Sprintf("%s %s", g.Title, "Capture Preview")

	w := app.NewWindow(
		app.Title(header),
		app.Size(unit.Dp(g.min.X), unit.Dp(g.min.Y)),
		app.MinSize(unit.Dp(g.min.X), unit.Dp(g.min.Y)),
		app.MaxSize(unit.Dp(g.max.X), unit.Dp(g.max.Y)),
		app.Decorated(false),
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

	max := false
	bar := title.New(header, nil, func() {
		max = !max
		if !max {
			w.Perform(system.ActionUnmaximize)
		} else {
			w.Perform(system.ActionMaximize)
		}
	}, func() {
		w.Perform(system.ActionClose)
	})

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

			bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				colorBox(gtx, gtx.Constraints.Max, nrgba.Background)

				return style.Layout(gtx, len(images), func(gtx layout.Context, index int) layout.Dimensions {
					cap := captures[index]
					img := images[index]

					label := material.Label(g.normal, unit.Sp(18), cap.Option)
					label.Color = nrgba.Highlight.Color()
					label.Font.Weight = 200

					if cap.Matched != nil {
						label.Text = cap.Matched.Text
						label.Color = cap.Matched.Color()
					}

					return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return label.Layout(gtx)
									})
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return img.Layout(g.normal, gtx)
							}),
						)
					})
				})
			})
			w.Invalidate()

			e.Frame(gtx.Ops)
		}
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
