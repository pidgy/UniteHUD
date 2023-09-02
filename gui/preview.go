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

	"github.com/pidgy/unitehud/fonts"
	"github.com/pidgy/unitehud/gui/visual/area"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/decor"
	"github.com/pidgy/unitehud/gui/visual/screen"
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/nrgba"
	"github.com/pidgy/unitehud/video"
)

type preview struct {
	parent *GUI
	window *app.Window
	hwnd   uintptr

	closed bool
	resize bool

	width,
	height int
}

func (p *preview) close() bool {
	was := p.closed
	p.closed = true
	return !was
}

func (p *preview) open(a *areas, onclose func()) {
	if !p.closed {
		return
	}
	p.closed = false

	defer onclose()

	p.window = app.NewWindow(
		app.Title("Preview"),
		app.Size(unit.Dp(p.width), unit.Dp(p.height)),
		app.MinSize(unit.Dp(p.width), unit.Dp(p.height)),
		app.MaxSize(unit.Dp(p.width), unit.Dp(p.parent.max.Y)),
		app.Decorated(false),
	)

	images := []*button.Image{}

	bar := title.New(
		"Preview",
		fonts.NewCollection(),
		nil,
		nil,
		func() { p.window.Perform(system.ActionClose) },
	)
	bar.NoTip = true

	captures := []*area.Capture{
		a.energy.Capture,
		a.ko.Capture,
		a.objective.Capture,
		a.score.Capture,
		a.state.Capture,
		a.time.Capture,
	}

	// Ordered by least widest.
	sort.Slice(captures, func(i, j int) bool {
		return captures[i].Base.Dy() < captures[j].Base.Dy()
	})

	for _, cap := range captures {
		img, err := video.CaptureRect(cap.Base)
		if err != nil {
			p.parent.ToastError(err)
			return
		}

		images = append(images, p.makePreviewCaptureButton(cap, img))
	}

	go func() {
		for ; p.closed; time.Sleep(time.Second) {
			for i, cap := range captures {
				img, err := video.CaptureRect(cap.Base)
				if err != nil {
					p.parent.ToastError(err)
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
	style := material.List(bar.Collection.Calibri().Theme, list)
	style.Track.Color = nrgba.Gray.Color()

	var ops op.Ops

	headerLabel := material.Body1(bar.Collection.Calibri().Theme, "Preview the areas visible to UniteHUD, select an image to save and preview on your desktop")
	headerLabel.Color = nrgba.Highlight.Color()
	headerLabel.Font.Weight = 200

	captureLabel := material.Label(bar.Collection.Calibri().Theme, unit.Sp(18), "")
	captureLabel.Color = nrgba.Highlight.Color()
	captureLabel.Font.Weight = 200

	matchLabel := material.Label(bar.Collection.Calibri().Theme, unit.Sp(18), "")
	matchLabel.Color = nrgba.Highlight.Color()
	matchLabel.Font.Weight = 200

	p.window.Perform(system.ActionRaise)

	p.parent.setInsetLeft(p.width)
	defer p.parent.unsetInsetLeft(p.width)

	for event := range p.window.Events() {
		switch e := event.(type) {
		case system.DestroyEvent:
			return
		case app.ViewEvent:
			p.hwnd = e.HWND
			p.parent.attachWindowLeft(p.hwnd, p.width)
		case system.FrameEvent:
			if p.closed {
				go p.window.Perform(system.ActionClose)
			}

			if p.resize {
				p.resize = false
				p.parent.attachWindowLeft(p.hwnd, p.width)
			}

			gtx := layout.NewContext(&ops, e)

			bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				decor.ColorBox(gtx, gtx.Constraints.Max, nrgba.Background)

				return layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					// Title.
					layout.Flexed(.2, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{
							Axis: layout.Horizontal,
						}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Spacer{Width: unit.Dp(10), Height: unit.Dp(1)}.Layout(gtx)
							}),

							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return headerLabel.Layout(gtx)
								})
							}),

							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Spacer{Width: unit.Dp(10), Height: unit.Dp(1)}.Layout(gtx)
							}),
						)
					}),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Spacer{Width: unit.Dp(1), Height: unit.Dp(15)}.Layout(gtx)
					}),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						decor.ColorBox(gtx, image.Pt(gtx.Constraints.Max.X, 1), nrgba.White.Alpha(5))
						return layout.Spacer{Width: unit.Dp(1), Height: unit.Dp(5)}.Layout(gtx)
					}),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Spacer{Width: unit.Dp(1), Height: unit.Dp(5)}.Layout(gtx)
					}),

					layout.Flexed(.8, func(gtx layout.Context) layout.Dimensions {
						return style.Layout(gtx, len(images), func(gtx layout.Context, index int) layout.Dimensions {
							cap := captures[index]
							img := images[index]

							return layout.Flex{
								Axis:      layout.Vertical,
								Spacing:   layout.SpaceEvenly,
								Alignment: layout.Middle,
							}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{
										Axis:      layout.Horizontal,
										Spacing:   layout.SpaceEvenly,
										Alignment: layout.Middle,
									}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Spacer{Width: unit.Dp(1), Height: unit.Dp(1)}.Layout(gtx)
										}),

										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Flex{
												Axis: layout.Vertical,
											}.Layout(gtx,
												// Title.
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
														return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
															captureLabel.Color = nrgba.White.Color()
															if cap.MatchedColor != nrgba.Nothing.Color() {
																captureLabel.Color = cap.MatchedColor
															}

															captureLabel.Text = cap.Option
															if cap.MatchedText != "" {
																captureLabel.Text = cap.MatchedText
															}

															return captureLabel.Layout(gtx)
														})
													})
												}),

												// Image.
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													gtx.Constraints.Max.X /= 2
													gtx.Constraints.Max.Y /= len(images)
													return img.Layout(bar.Collection.Calibri().Theme, gtx)
												}),
											)
										}),

										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Spacer{Width: unit.Dp(1), Height: unit.Dp(1)}.Layout(gtx)
										}),
									)
								}),

								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Spacer{Width: unit.Dp(1), Height: unit.Dp(15)}.Layout(gtx)
								}),

								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									decor.ColorBox(gtx, image.Pt(gtx.Constraints.Max.X, 1), nrgba.White.Alpha(5))
									return layout.Spacer{Width: unit.Dp(1), Height: unit.Dp(5)}.Layout(gtx)
								}),

								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Spacer{Width: unit.Dp(1), Height: unit.Dp(5)}.Layout(gtx)
								}),
							)
						})
					}),
				)
			})

			p.window.Invalidate()
			e.Frame(gtx.Ops)
		default:
			notify.Debug("Event missed: %T (Preview Window)", e)
		}
	}
}

func (p *preview) makePreviewCaptureButton(cap *area.Capture, img image.Image) *button.Image {
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
				p.parent.ToastError(fmt.Errorf("Failed to open capture preview (%v)", err))
			}
		},
	}
}

func dimensions(gtx layout.Context) layout.Dimensions {
	return layout.Dimensions{Size: gtx.Constraints.Max}
}
