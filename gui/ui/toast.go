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
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/avi/img/splash"
	"github.com/pidgy/unitehud/core/fonts"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/gui/ux/button"
	"github.com/pidgy/unitehud/gui/ux/decorate"
	"github.com/pidgy/unitehud/gui/ux/title"
)

type (
	OnToastYes func()
	OnToastNo  func()
	OnToastOK  func()
)

const (
	toastTextSize = unit.Sp(15)
)

type (
	toast struct {
		g *GUI

		bar    *title.Widget
		window *app.Window
		label  material.LabelStyle
		ops    op.Ops
	}

	closeable struct {
		*toast
		waitq chan bool
	}

	waiter interface {
		close()
		wait() waiter
	}
)

func (c *closeable) close() {
	c.toast.window.Perform(system.ActionClose)
}

func (c *closeable) wait() waiter {
	<-c.waitq
	return c
}

func (c *closeable) ready() {
	go func() { c.waitq <- true }()
}

func (g *GUI) toast(header, msg string, width, height float32) *toast {
	notify.Debug("[Toast] %s: %s", header, msg)

	if g.previous.toast.active {
		return nil
	}
	g.previous.toast.active = true

	t := &toast{
		g: g,

		window: app.NewWindow(
			app.Title(header),
			app.Size(unit.Dp(width), unit.Dp(height)),
			app.MaxSize(unit.Dp(width), unit.Dp(height)),
			app.MinSize(unit.Dp(width), unit.Dp(height)),
			app.Decorated(false),
		),
	}
	t.bar = title.New(header, fonts.NewCollection(), nil, nil, func() {
		t.window.Perform(system.ActionClose)
	})
	t.bar.NoTip = true
	t.bar.NoDrag = true

	t.label = material.Label(t.bar.Collection.Calibri().Theme, toastTextSize, titleFirstWord(msg))
	t.label.Alignment = text.Middle
	return t
}

func (t *toast) close() {
	t.g.previous.toast.active = false
}

func (g *GUI) ToastError(err error) {
	if g.previous.toast.err != nil && err.Error() == g.previous.toast.err.Error() && time.Since(g.previous.toast.time) < time.Second {
		return
	}

	g.previous.toast.err = err
	g.previous.toast.time = time.Now()

	g.ToastOK("Error", err.Error(), OnToastOK(nil))
}

func (g *GUI) ToastErrorf(format string, a ...interface{}) {
	g.ToastError(fmt.Errorf(format, a...))
}

func (g *GUI) ToastOK(header, msg string, ok OnToastOK) {
	t := g.toast(header, msg, float32(400), float32(125))
	if t == nil {
		return
	}

	go func() {
		notify.Debug("[UI] Opening Ok (active: %t)", g.previous.toast.active)
		defer notify.Debug("[UI] Closing Ok (active: %t)", g.previous.toast.active)
		defer t.close()

		okButton := &button.Widget{
			Text:            "OK",
			TextSize:        unit.Sp(16),
			Font:            t.bar.Collection.Calibri(),
			Pressed:         nrgba.Transparent80,
			Released:        nrgba.DarkGray,
			Size:            image.Pt(96, 32),
			TextInsetBottom: -2,

			Click: func(this *button.Widget) {
				defer this.Deactivate()

				if ok != nil {
					ok()
				}

				t.window.Perform(system.ActionClose)
			},
		}

		for e := t.window.NextEvent(); ; e = t.window.NextEvent() {
			if _, ok := e.(system.DestroyEvent); ok {
				t.window.Perform(system.ActionClose)
				return
			}

			event, ok := e.(system.FrameEvent)
			if !ok {
				notify.Missed(event, "ToastOk")
				continue
			}

			gtx := layout.NewContext(&t.ops, event)

			t.bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{
						Axis:      layout.Vertical,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Rigid(layout.Spacer{Height: 10}.Layout),

						layout.Flexed(.5, func(gtx layout.Context) layout.Dimensions {
							decorate.Label(&t.label, t.label.Text)
							return layout.Center.Layout(gtx, t.label.Layout)
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

			t.window.Perform(system.ActionCenter)
			t.window.Perform(system.ActionRaise)
			t.window.Invalidate()
			event.Frame(gtx.Ops)
		}
	}()
}

func (g *GUI) ToastSplash(header, msg string, img image.Image) waiter {
	c := &closeable{
		toast: g.toast(header, msg, float32(640), float32(360)),
		waitq: make(chan bool),
	}
	if c.toast == nil {
		return c
	}
	defer c.ready()

	go func() {
		notify.Debug("[UI] Opening Splash (active: %t)", g.previous.toast.active)
		defer notify.Debug("[UI] Closing Splash (active: %t)", g.previous.toast.active)
		defer c.toast.close()

		c.toast.bar.Hide = true
		c.label.TextSize = toastTextSize * 1.5
		c.label.Color = nrgba.White.Color()

		op := paint.NewImageOp(img)

		for e := c.toast.window.NextEvent(); ; e = c.toast.window.NextEvent() {
			if _, ok := e.(system.DestroyEvent); ok {
				c.toast.window.Perform(system.ActionClose)
				return
			}

			event, ok := e.(system.FrameEvent)
			if !ok {
				notify.Missed(event, "ToastSplash")
				continue
			}

			gtx := layout.NewContext(&c.toast.ops, event)

			layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx layout.Context) layout.Dimensions {
					return widget.Image{
						Src:   op,
						Scale: float32(splash.Loading().Bounds().Dx()) / float32(gtx.Constraints.Max.X),
						Fit:   widget.Cover,
					}.Layout(gtx)
				}),
			)

			layout.S.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(25)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return c.label.Layout(gtx)
					})
				})
			})

			c.toast.window.Perform(system.ActionCenter)
			c.toast.window.Perform(system.ActionRaise)
			c.toast.window.Invalidate()
			event.Frame(gtx.Ops)
		}
	}()

	return c
}

func (g *GUI) ToastYesNo(header, msg string, y OnToastYes, n OnToastNo) {
	t := g.toast(header, msg, float32(400), float32(125))
	if t == nil {
		return
	}

	go func() {
		notify.Debug("[UI] Opening Yes/No (active: %t)", g.previous.toast.active)
		defer notify.Debug("[UI] Closing Yes/No (active: %t)", g.previous.toast.active)
		defer t.close()

		yButton := &button.Widget{
			Text:            "Yes",
			TextSize:        unit.Sp(16),
			Font:            t.bar.Collection.Calibri(),
			Pressed:         nrgba.Transparent80,
			Released:        nrgba.DarkGray,
			Size:            image.Pt(96, 32),
			TextInsetBottom: -2,
			Click: func(this *button.Widget) {
				if y != nil {
					y()
				}
				t.window.Perform(system.ActionClose)
			},
		}

		nButton := &button.Widget{
			Text:            "No",
			TextSize:        unit.Sp(16),
			Font:            t.bar.Collection.Calibri(),
			Pressed:         nrgba.Transparent80,
			Released:        nrgba.DarkGray,
			Size:            image.Pt(96, 32),
			TextInsetBottom: -2,
			Click: func(this *button.Widget) {
				if n != nil {
					n()
				}
				t.window.Perform(system.ActionClose)
			},
		}

		for e := t.window.NextEvent(); ; e = t.window.NextEvent() {
			if _, ok := e.(system.DestroyEvent); ok {
				t.window.Perform(system.ActionClose)
				return
			}

			event, ok := e.(system.FrameEvent)
			if !ok {
				notify.Missed(event, "ToastYesNo")
				continue
			}

			gtx := layout.NewContext(&t.ops, event)

			t.bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{
						Axis:      layout.Vertical,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Rigid(layout.Spacer{Height: 10}.Layout),

						layout.Flexed(.5, func(gtx layout.Context) layout.Dimensions {
							decorate.Label(&t.label, t.label.Text)
							return layout.Center.Layout(gtx, t.label.Layout)
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

			t.window.Perform(system.ActionCenter)
			t.window.Perform(system.ActionRaise)
			t.window.Invalidate()
			event.Frame(gtx.Ops)
		}
	}()
}

func titleFirstWord(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}
