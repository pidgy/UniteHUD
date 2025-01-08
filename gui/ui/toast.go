package ui

import (
	"fmt"
	"image"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/font"
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
	OnToastYes      func()
	OnToastNo       func()
	OnToastOK       func()
	OnToastRemember func(b bool)
)

const (
	toastTextSize = unit.Sp(15)
)

type (
	Bulletin struct {
		Title string

		Topics []struct {
			Subtitle string
			Points   []string
		}
	}

	toast struct {
		g *GUI

		bar    *title.Widget
		window *app.Window
		label  material.LabelStyle
		ops    op.Ops

		forced bool
	}

	closeable struct {
		*toast
		waitq chan bool
	}

	waiter interface {
		close(*GUI)
		wait() waiter
	}
)

var (
	active = []*toast{}
)

func (t *toast) close(g *GUI) {
	t.window.Perform(system.ActionClose)

	g.previous.toast.err = nil
	g.previous.toast.active = false

	for i := range active {
		if t == active[i] {
			active = append(active[:i], active[i+1:]...)
			return
		}
	}
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

	return g.toastForce(header, msg, width, height)
}

func (g *GUI) toastForce(header, msg string, width, height float32) *toast {
	t := &toast{
		g: g,

		window: app.NewWindow(
			app.Title(header),
			app.Decorated(false),
			app.Size(unit.Dp(width), unit.Dp(height)),
			app.MinSize(unit.Dp(width), unit.Dp(height)),
			app.MaxSize(unit.Dp(width), unit.Dp(height))),

		forced: true,
	}
	t.bar = title.New(header, fonts.NewCollection(), nil, nil, func() {
		t.window.Perform(system.ActionClose)
	})
	t.bar.NoTip = true
	t.bar.NoDrag = true

	t.label = material.Label(t.bar.Collection.Calibri().Theme, toastTextSize, titleFirstWord(msg))
	t.label.Alignment = text.Middle

	active = append(active, t)

	return t
}

func (g *GUI) ToastError(err error) {
	if g.previous.toast.err != nil && err.Error() == g.previous.toast.err.Error() {
		return
	}

	g.previous.toast.err = err
	g.previous.toast.time = time.Now()

	t := g.toastForce("Error", err.Error(), float32(400), float32(125))
	if t == nil {
		return
	}

	g.toastOK2(t, OnToastOK(nil))
}

func (g *GUI) ToastErrorf(format string, a ...interface{}) {
	g.ToastError(fmt.Errorf(format, a...))
}

func (g *GUI) ToastNewsletter(header string, bulletin Bulletin, ok OnToastOK) {
	t := g.toastForce(header, bulletin.Title, float32(600), float32(450))
	if t == nil {
		return
	}

	t.label.TextSize = toastTextSize * 1.5

	go func() {
		notify.Debug("[UI] Toast: Opening Newsletter (active: %t)", g.previous.toast.active)
		defer notify.Debug("[UI] Toast: Closing Newsletter (active: %t)", g.previous.toast.active)
		defer t.close(g)

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

		topic := func(i int) (topics []layout.FlexChild) {
			topics = append(topics, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				subtitle := material.Label(t.bar.Collection.Calibri().Theme, toastTextSize, bulletin.Topics[i].Subtitle)
				decorate.Label(&subtitle, subtitle.Text)
				subtitle.TextSize = toastTextSize * 1.25
				subtitle.Font.Weight = font.ExtraBold
				return layout.Center.Layout(gtx, subtitle.Layout)
			}))

			topics = append(topics, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				point := func(j int) layout.FlexChild {
					return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						p := material.Label(t.bar.Collection.Calibri().Theme, toastTextSize, fmt.Sprintf("• %s", bulletin.Topics[i].Points[j]))
						decorate.Label(&p, p.Text)
						p.Font.Style = font.Italic
						p.Alignment = text.Middle
						return layout.E.Layout(gtx, p.Layout)
					})
				}

				points := []layout.FlexChild{}
				for j := range bulletin.Topics[i].Points {
					points = append(points, point(j))
				}

				return layout.Flex{
					Axis:      layout.Vertical,
					Alignment: layout.Start,
				}.Layout(gtx, points...)
			}))

			return append(topics, layout.Rigid(layout.Spacer{Height: 25}.Layout))
		}

		c := []layout.FlexChild{}
		for i := range bulletin.Topics {
			c = append(c, topic(i)...)
		}

		list := material.List(
			t.bar.Collection.Calibri().Theme,
			&widget.List{
				Scrollbar: widget.Scrollbar{},
				List: layout.List{
					Axis:        layout.Vertical,
					ScrollToEnd: false,
					Alignment:   layout.Baseline,
				},
			},
		)

		for e := t.window.NextEvent(); ; e = t.window.NextEvent() {
			if _, ok := e.(system.DestroyEvent); ok {
				t.window.Perform(system.ActionClose)
				return
			}

			event, ok := e.(system.FrameEvent)
			if !ok {
				notify.Missed(event, "ToastNewsletter")
				continue
			}

			gtx := layout.NewContext(&t.ops, event)

			t.bar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{
						Axis:      layout.Vertical,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Flexed(.2, func(gtx layout.Context) layout.Dimensions {
							decorate.Label(&t.label, t.label.Text)

							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return decorate.Underline(gtx, t.label.Layout)
							})
						}),

						layout.Flexed(.5, func(gtx layout.Context) layout.Dimensions {
							decorate.Scrollbar(&list.ScrollbarStyle)
							decorate.List(&list)

							return decorate.UnderlineBorder(gtx, func(gtx layout.Context) layout.Dimensions {
								return list.Layout(gtx, 1, func(gtx layout.Context, index int) layout.Dimensions {
									return layout.Flex{
										Axis:      layout.Vertical,
										Alignment: layout.Middle,
									}.Layout(gtx, c...)
								})
							})
						}),

						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return decorate.Underline(gtx, layout.Spacer{Height: 10}.Layout)
						}),

						layout.Flexed(.2, func(gtx layout.Context) layout.Dimensions {
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

func (g *GUI) ToastOK(header, msg string, ok OnToastOK) {
	t := g.toast(header, msg, float32(400), float32(125))
	if t == nil {
		return
	}

	g.toastOK2(t, ok)
}

func (g *GUI) toastOK2(t *toast, ok OnToastOK) {
	first := true

	go func() {
		notify.Debug("[UI] Toast: Opening Ok (active: %t)", g.previous.toast.active)
		defer notify.Debug("[UI] Toast: Closing Ok (active: %t)", g.previous.toast.active)
		defer t.close(g)

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
				okButton.Click(okButton)
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

			if first || t.forced {
				t.window.Perform(system.ActionCenter)
				t.window.Perform(system.ActionRaise)
				first = false
			}

			t.window.Invalidate()

			event.Frame(gtx.Ops)
		}
	}()
}

func (g *GUI) ToastSplash(header, msg string, img image.Image) waiter {
	c := &closeable{
		toast: g.toastForce(header, msg, float32(640), float32(360)),
		waitq: make(chan bool),
	}
	if c.toast == nil {
		return c
	}
	defer c.ready()

	go func() {
		notify.Debug("[UI] Toast: Opening Splash (active: %t)", g.previous.toast.active)
		defer notify.Debug("[UI] Toast: Closing Splash (active: %t)", g.previous.toast.active)
		defer c.toast.close(g)

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
	g.ToastYesNoRememberDecision(header, msg, "", y, n, nil)
}

func (g *GUI) ToastYesNoRememberDecision(header, msg, decision string, y OnToastYes, n OnToastNo, r OnToastRemember) {
	h := float32(125)
	if decision != "" {
		h = 150
	}

	t := g.toast(header, msg, 400, h)
	if t == nil {
		return
	}

	first := true

	go func() {
		notify.Debug("[UI] Toast: Opening Yes/No (active: %t)", g.previous.toast.active)
		defer notify.Debug("[UI] Toast: Closing Yes/No (active: %t)", g.previous.toast.active)
		defer t.close(g)

		check := material.CheckBox(t.bar.Collection.Calibri().Theme, &widget.Bool{}, titleFirstWord(decision))

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
				if r != nil {
					r(check.CheckBox.Value)
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
				if r != nil {
					r(check.CheckBox.Value)
				}
				t.window.Perform(system.ActionClose)
			},
		}

		remember := func() layout.FlexChild {
			if decision == "" {
				return layout.Rigid(layout.Spacer{}.Layout)
			}

			check.Color = nrgba.Discord.Color()

			return layout.Flexed(.5, func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, check.Layout)
			})
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

						remember(),

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

			if first {
				t.window.Perform(system.ActionCenter)
				t.window.Perform(system.ActionRaise)
				first = false
			}

			t.window.Invalidate()

			event.Frame(gtx.Ops)
		}
	}()
}

func titleFirstWord(s string) string {
	if len(s) < 1 {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
