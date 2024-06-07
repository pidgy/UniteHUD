package ui

import (
	"fmt"
	"image"
	"sort"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/avi/video"
	"github.com/pidgy/unitehud/core/fonts"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/gui/ux/area"
	"github.com/pidgy/unitehud/gui/ux/button"
	"github.com/pidgy/unitehud/gui/ux/decorate"
	"github.com/pidgy/unitehud/gui/ux/screen"
	"github.com/pidgy/unitehud/gui/ux/title"
)

type preview struct {
	hwnd uintptr

	bar *title.Widget

	windows struct {
		parent *GUI
		this   *app.Window
	}

	state struct {
		open bool
	}

	dimensions struct {
		width,
		height int

		resize bool
	}

	capture struct {
		areas     []*area.Capture
		liststyle material.ListStyle
		images    []*button.ImageWidget
	}

	labels struct {
		header,
		capture,
		match material.LabelStyle
	}
}

func (g *GUI) preview(a *areas, onclose func()) *preview {
	ui := g.previewUI()

	go func() {
		defer onclose()

		ui.state.open = true
		defer func() {
			ui.state.open = false
		}()

		ui.capture.images = []*button.ImageWidget{}
		ui.capture.areas = []*area.Capture{
			a.energy.Capture,
			// a.ko.Capture,
			a.objective.Capture,
			a.score.Capture,
			a.state.Capture,
			a.time.Capture,
			a.pressButtonToScore.Capture,
		}

		// Ordered by least widest.
		sort.Slice(ui.capture.areas, func(i, j int) bool {
			return ui.capture.areas[i].Base.Dy() < ui.capture.areas[j].Base.Dy()
		})

		for _, cap := range ui.capture.areas {
			img, err := video.CaptureRect(cap.Base)
			if err != nil {
				ui.windows.parent.ToastError(err)
				return
			}

			ui.capture.images = append(ui.capture.images, ui.makePreviewCaptureButton(cap, img))
		}

		go func() {
			for ; ui.state.open; time.Sleep(time.Second) {
				for i, cap := range ui.capture.areas {
					img, err := video.CaptureRect(cap.Base)
					if err != nil {
						ui.windows.parent.ToastError(err)
						return
					}

					ui.capture.images[i].SetImage(img)
				}
			}
		}()

		ui.windows.this.Perform(system.ActionRaise)

		ui.windows.parent.setInsetLeft(ui.dimensions.width)
		defer ui.windows.parent.unsetInsetLeft(ui.dimensions.width)

		var ops op.Ops

		for {
			switch event := ui.windows.this.NextEvent().(type) {
			case system.DestroyEvent:
				ui.state.open = false
				return
			case app.ViewEvent:
				ui.hwnd = event.HWND
				ui.windows.parent.attachWindowLeft(ui.hwnd, ui.dimensions.width)
			case system.FrameEvent:
				if !ui.state.open {
					go ui.windows.this.Perform(system.ActionClose)
				}

				if ui.dimensions.resize {
					ui.dimensions.resize = false
					ui.windows.parent.attachWindowLeft(ui.hwnd, ui.dimensions.width)
				}

				gtx := layout.NewContext(&ops, event)

				ui.bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Dimensions{Size: gtx.Constraints.Max}
					})

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
										return ui.labels.header.Layout(gtx)
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
							decorate.Border(gtx)
							return layout.Spacer{Width: unit.Dp(1), Height: unit.Dp(5)}.Layout(gtx)
						}),

						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Spacer{Width: unit.Dp(1), Height: unit.Dp(5)}.Layout(gtx)
						}),

						layout.Flexed(.8, func(gtx layout.Context) layout.Dimensions {
							decorate.Scrollbar(&ui.capture.liststyle.ScrollbarStyle)

							return ui.capture.liststyle.Layout(gtx, len(ui.capture.images), func(gtx layout.Context, index int) layout.Dimensions {
								cap := ui.capture.areas[index]
								img := ui.capture.images[index]

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
																ui.labels.capture.Text = cap.Option
																if cap.MatchedText != "" {
																	ui.labels.capture.Text = cap.MatchedText
																}

																if cap.MatchedColor != nrgba.Nothing.Color() {
																	ui.labels.capture.Color = cap.MatchedColor
																} else {
																	decorate.Label(&ui.labels.capture, ui.labels.capture.Text)
																}

																return ui.labels.capture.Layout(gtx)
															})
														})
													}),

													// Image.
													layout.Rigid(func(gtx layout.Context) layout.Dimensions {
														gtx.Constraints.Max.X /= 2
														gtx.Constraints.Max.Y /= len(ui.capture.images)
														return img.Layout(ui.bar.Collection.Calibri().Theme, gtx)
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
										decorate.Border(gtx)
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

				ui.windows.this.Invalidate()

				event.Frame(gtx.Ops)
			default:
				notify.Missed(event, "Preview")
			}
		}
	}()

	return ui
}

func (g *GUI) previewUI() *preview {
	ui := &preview{}

	ui.dimensions.width = 350
	ui.dimensions.height = 700
	ui.dimensions.resize = true

	ui.windows.parent = g

	ui.windows.this = app.NewWindow(
		app.Title("Preview"),
		app.Size(unit.Dp(ui.dimensions.width), unit.Dp(ui.dimensions.height)),
		app.MinSize(unit.Dp(ui.dimensions.width), unit.Dp(ui.dimensions.height)),
		app.MaxSize(unit.Dp(ui.dimensions.width), unit.Dp(ui.windows.parent.dimensions.max.Y)),
		app.Decorated(false),
	)

	ui.bar = title.New(
		"Preview",
		fonts.NewCollection(),
		nil,
		nil,
		func() { ui.windows.this.Perform(system.ActionClose) },
	)
	ui.bar.NoTip = true
	ui.bar.NoDrag = true

	ui.labels.header = material.Body1(ui.bar.Collection.Calibri().Theme, "📌 Press ⛶ to start detecting events, select images below to save to your desktop.")
	ui.labels.header.Color = nrgba.Slate.Color()
	ui.labels.header.Font.Weight = 200
	ui.labels.header.TextSize = 14
	ui.labels.header.Alignment = text.Start

	ui.labels.capture = material.Label(ui.bar.Collection.Calibri().Theme, unit.Sp(14), "")
	ui.labels.capture.Color = nrgba.Highlight.Color()
	ui.labels.capture.Font.Weight = 100
	ui.labels.capture.Alignment = text.Start

	ui.labels.match = material.Label(ui.bar.Collection.Calibri().Theme, unit.Sp(14), "")
	ui.labels.match.Color = nrgba.Highlight.Color()
	ui.labels.match.Font.Weight = 100

	ui.capture.liststyle = material.List(ui.bar.Collection.Calibri().Theme, &widget.List{
		Scrollbar: widget.Scrollbar{},
		List: layout.List{
			Axis:      layout.Vertical,
			Alignment: layout.Baseline,
		},
	})

	ui.capture.images = []*button.ImageWidget{}

	return ui
}

func (p *preview) close() {
	if p != nil {
		p.windows.this.Perform(system.ActionClose)
	}
}

func (p *preview) makePreviewCaptureButton(cap *area.Capture, img image.Image) *button.ImageWidget {
	return &button.ImageWidget{
		Widget: &screen.Widget{
			Image:       img,
			Border:      true,
			BorderColor: nrgba.Transparent,
			AutoScale:   true,
		},
		Click: func(i *button.ImageWidget) {
			err := cap.Open()
			if err != nil {
				p.windows.parent.ToastError(fmt.Errorf("Failed to open capture preview (%v)", err))
			}
		},
	}
}

func (p *preview) open() bool {
	return p != nil
}

func (p *preview) resize() {
	if p != nil {
		p.dimensions.resize = true
	}
}

func dimensions(gtx layout.Context) layout.Dimensions {
	return layout.Dimensions{Size: gtx.Constraints.Max}
}
