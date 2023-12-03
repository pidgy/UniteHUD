package ui

import (
	"fmt"
	"image"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/core/fonts"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/nrgba"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/decorate"
	"github.com/pidgy/unitehud/gui/visual/title"
)

type (
	OnToastYes func()
	OnToastNo  func()
	OnToastOK  func()
)

const (
	toastTextSize = unit.Sp(15)
)

func (g *GUI) ToastCrash(reason string, closed, logs func()) {
	go func() {
		g.previous.toast.active = true
		defer func() { g.previous.toast.active = false }()

		width, height := float32(500), float32(125)

		w := app.NewWindow(
			app.Title("Crashed"),
			app.Size(unit.Dp(width), unit.Dp(height)),
			app.MaxSize(unit.Dp(width), unit.Dp(height)),
			app.MinSize(unit.Dp(width), unit.Dp(height)),
			app.Decorated(false),
		)

		bar := title.New("Crashed", fonts.NewCollection(), nil, nil, func() {
			w.Perform(system.ActionClose)
		})
		bar.NoTip = true
		bar.NoDrag = true

		c := material.Label(bar.Collection.Calibri().Theme, toastTextSize, reason)
		c.Color = nrgba.PastelRed.Color()
		c.Alignment = text.Middle

		btn := &button.Widget{
			Text:            "View Logs",
			TextSize:        unit.Sp(16),
			Font:            bar.Collection.Calibri(),
			Pressed:         nrgba.Transparent80,
			Released:        nrgba.DarkGray,
			Size:            image.Pt(96, 32),
			TextInsetBottom: -2,

			Click: func(this *button.Widget) {
				defer this.Deactivate()

				if logs != nil {
					logs()
				}

				w.Perform(system.ActionClose)
			},
		}

		var ops op.Ops

		for {
			switch event := w.NextEvent().(type) {
			case system.DestroyEvent:
				if closed != nil {
					closed()
				}
				return
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, event)

				bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{
							Axis:      layout.Vertical,
							Alignment: layout.Middle,
						}.Layout(gtx,
							layout.Flexed(.1, layout.Spacer{Height: 5}.Layout),

							layout.Flexed(.3, func(gtx layout.Context) layout.Dimensions {
								return c.Layout(gtx)
							}),

							layout.Flexed(.2, layout.Spacer{Height: 1}.Layout),

							layout.Flexed(.3, func(gtx layout.Context) layout.Dimensions {
								return layout.Center.Layout(gtx, btn.Layout)
							}),

							layout.Flexed(.1, layout.Spacer{Height: 5}.Layout),
						)
					})
				})

				w.Perform(system.ActionCenter)
				w.Perform(system.ActionRaise)
				w.Invalidate()

				event.Frame(gtx.Ops)

			default:
				notify.Missed(event, "ToastCrash")
			}
		}
	}()
}

func (g *GUI) ToastError(err error) {
	if g.previous.toast.err != nil && err.Error() == g.previous.toast.err.Error() && time.Since(g.previous.toast.time) < time.Second {
		return
	}

	g.previous.toast.err = err
	g.previous.toast.time = time.Now()

	e := err.Error()
	es := strings.Split(e, " ")
	es[0] = strings.Title(es[0])

	g.ToastOK("Error", strings.Join(es, " "), OnToastOK(nil))
}

func (g *GUI) ToastErrorf(format string, a ...interface{}) {
	g.ToastError(fmt.Errorf(format, a...))
}

func (g *GUI) ToastOK(header, msg string, ok OnToastOK) {
	if g.previous.toast.active {
		return
	}

	go func() {
		g.previous.toast.active = true
		defer func() { g.previous.toast.active = false }()

		width, height := float32(400), float32(125)

		w := app.NewWindow(
			app.Title(header),
			app.Size(unit.Dp(width), unit.Dp(height)),
			app.MaxSize(unit.Dp(width), unit.Dp(height)),
			app.MinSize(unit.Dp(width), unit.Dp(height)),
			app.Decorated(false),
		)

		bar := title.New(header, fonts.NewCollection(), nil, nil, func() {
			w.Perform(system.ActionClose)
		})
		bar.NoTip = true
		bar.NoDrag = true

		label := material.Label(bar.Collection.Calibri().Theme, toastTextSize, msg)

		okButton := &button.Widget{
			Text:            "OK",
			TextSize:        unit.Sp(16),
			Font:            bar.Collection.Calibri(),
			Pressed:         nrgba.Transparent80,
			Released:        nrgba.DarkGray,
			Size:            image.Pt(96, 32),
			TextInsetBottom: -2,

			Click: func(this *button.Widget) {
				defer this.Deactivate()

				if ok != nil {
					ok()
				}

				w.Perform(system.ActionClose)
			},
		}

		var ops op.Ops

		for {
			event, ok := w.NextEvent().(system.FrameEvent)
			if !ok {
				notify.Missed(event, "ToastOk")
				continue
			}

			gtx := layout.NewContext(&ops, event)

			bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{
						Axis:      layout.Vertical,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Rigid(layout.Spacer{Height: 10}.Layout),

						layout.Flexed(.5, func(gtx layout.Context) layout.Dimensions {
							decorate.Label(&label, label.Text)
							return layout.Center.Layout(gtx, label.Layout)
						}),

						layout.Flexed(.5, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{
								Axis: layout.Horizontal,
							}.Layout(gtx,
								layout.Rigid(layout.Spacer{Width: 5}.Layout),

								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return layout.Center.Layout(gtx, okButton.Layout)
								}),

								layout.Rigid(layout.Spacer{Width: 5}.Layout),
							)
						}),

						layout.Rigid(layout.Spacer{Height: 2}.Layout),
					)
				})
			})

			w.Perform(system.ActionCenter)
			w.Perform(system.ActionRaise)
			w.Invalidate()
			event.Frame(gtx.Ops)
		}
	}()
}

func (g *GUI) ToastYesNo(header, msg string, y OnToastYes, n OnToastNo) {
	if g.previous.toast.active {
		return
	}

	go func() {
		g.previous.toast.active = true
		defer func() { g.previous.toast.active = false }()

		width, height := unit.Dp(400), unit.Dp(125)

		w := app.NewWindow(
			app.Title(header),
			app.Size(width, height),
			app.MaxSize(width, height),
			app.MinSize(width, height),
			app.Decorated(false),
		)

		bar := title.New(header, fonts.NewCollection(), nil, nil, func() {
			w.Perform(system.ActionClose)
		})
		bar.NoTip = true
		bar.NoDrag = true

		label := material.Label(bar.Collection.Calibri().Theme, toastTextSize, msg)

		yButton := &button.Widget{
			Text:            "Yes",
			TextSize:        unit.Sp(16),
			Font:            bar.Collection.Calibri(),
			Pressed:         nrgba.Transparent80,
			Released:        nrgba.DarkGray,
			Size:            image.Pt(96, 32),
			TextInsetBottom: -2,
			Click: func(this *button.Widget) {
				if y != nil {
					y()
				}
				w.Perform(system.ActionClose)
			},
		}

		nButton := &button.Widget{
			Text:            "No",
			TextSize:        unit.Sp(16),
			Font:            bar.Collection.Calibri(),
			Pressed:         nrgba.Transparent80,
			Released:        nrgba.DarkGray,
			Size:            image.Pt(96, 32),
			TextInsetBottom: -2,
			Click: func(this *button.Widget) {
				if n != nil {
					n()
				}
				w.Perform(system.ActionClose)
			},
		}

		var ops op.Ops

		for {
			event, ok := w.NextEvent().(system.FrameEvent)
			if !ok {
				notify.Missed(event, "ToastYesNo")
				continue
			}

			gtx := layout.NewContext(&ops, event)

			bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{
						Axis:      layout.Vertical,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Rigid(layout.Spacer{Height: 10}.Layout),

						layout.Flexed(.5, func(gtx layout.Context) layout.Dimensions {
							decorate.Label(&label, label.Text)
							return layout.Center.Layout(gtx, label.Layout)
						}),

						layout.Flexed(.5, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
								layout.Rigid(layout.Spacer{Width: 5}.Layout),

								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return layout.Center.Layout(gtx, yButton.Layout)
								}),

								layout.Rigid(layout.Spacer{Width: 1}.Layout),

								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return layout.Center.Layout(gtx, nButton.Layout)
								}),

								layout.Rigid(layout.Spacer{Width: 5}.Layout),
							)
						}),

						layout.Rigid(layout.Spacer{Height: 2}.Layout),
					)
				})
			})

			w.Perform(system.ActionCenter)
			w.Perform(system.ActionRaise)
			w.Invalidate()
			event.Frame(gtx.Ops)
		}
	}()
}
