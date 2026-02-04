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

	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/gui/ux/button"
	"github.com/pidgy/unitehud/gui/ux/decorate"
	"github.com/pidgy/unitehud/gui/ux/keys"
	"github.com/pidgy/unitehud/gui/ux/title"
	"github.com/pidgy/unitehud/media/img/splash"
	"github.com/pidgy/unitehud/system/ini"
)

type (
	// toastOnYes is the callback for affirmative toast actions.
	toastOnYes      func()
	// toastOnNo is the callback for negative toast actions.
	toastOnNo       func()
	// toastOnOK is the callback for OK-only toast actions.
	toastOnOK       func()
	// toastOnRemember is the callback for "remember my choice" actions.
	toastOnRemember func(b bool)
	// toastOnClose is the callback for toast close events.
	toastOnClose    func()
)

type (
	// bulletin defines a multi-section notice used in newsletters.
	bulletin struct {
		Title string

		Topics []struct {
			Subtitle string
			Points   []string
		}
	}

	// closeable wraps a toast with a ready signal.
	closeable struct {
		*toast
		waitq chan bool
	}

	// toast represents a dialog window with a title bar and message label.
	toast struct {
		g *GUI

		label  material.LabelStyle
		nav    *title.Widget
		ops    op.Ops
		window *app.Window

		forced bool
		open   bool

		keybinds keys.Bind
	}

	// waiter exposes toast readiness and close behavior.
	waiter interface {
		close()
		wait() waiter
	}
)

const (
	// toastTextSize is the default font size for toast labels.
	toastTextSize = unit.Sp(15)
)

// ToastError shows an error dialog if the message is new.
func (g *GUI) ToastError(err error) {
	if g.previous.toast.err != nil && err.Error() == g.previous.toast.err.Error() {
		return
	}

	err = fmt.Errorf("%s", ini.Format(err.Error()))
	g.previous.toast.err = err
	g.previous.toast.time = time.Now()

	t := g.makeToastForce("Error", err.Error(), 400, 125)
	if t == nil {
		notify.Error("[UI] Toast: Failed to show error: %v", err)
		return
	}
	defer t.close()

	if len(err.Error()) > 100 {
		t.window.Option(
			app.Size(unit.Dp(600), unit.Dp(125)),
			app.MinSize(unit.Dp(600), unit.Dp(125)),
			app.MaxSize(unit.Dp(600), unit.Dp(125)),
		)
	}

	g.toastOK(t, nil)
}

func (g *GUI) ToastErrorf(format string, a ...interface{}) {
	g.ToastError(fmt.Errorf(format, a...))
}

// ToastNewsletter shows a scrollable bulletin with multiple topics.
func (g *GUI) ToastNewsletter(header string, bulletin bulletin, c toastOnClose) {
	if c != nil {
		defer c()
	}

	t := g.makeToastForce(header, bulletin.Title, float32(650), float32(450))
	if t == nil {
		notify.Error("[UI] Toast: Failed to show Newsletter")
		return
	}
	defer t.close()

	t.label.TextSize = toastTextSize * 1.5

	notify.Debug("[UI] Toast: Opening Newsletter")
	defer notify.Debug("[UI] Toast: Closing Newsletter...")

	okButton := &button.Widget{
		Text:            "OK",
		TextSize:        unit.Sp(16),
		Font:            t.nav.Calibri(),
		Pressed:         nrgba.Transparent80,
		Released:        nrgba.DarkGray,
		Size:            image.Pt(96, 32),
		TextInsetBottom: -2,

		Click: func(this *button.Widget) {
			defer this.Deactivate()

			t.window.Perform(system.ActionClose)
		},
	}

	topic := func(i int) (topics []layout.FlexChild) {
		topics = append(topics, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			subtitle := material.Label(t.nav.Calibri().Theme, toastTextSize, bulletin.Topics[i].Subtitle)
			decorate.Label(&subtitle, "%s", subtitle.Text)
			subtitle.TextSize = toastTextSize * 1.25
			subtitle.Font.Weight = font.Bold
			return subtitle.Layout(gtx)
		}))

		point := func(p string) layout.FlexChild {
			return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(t.nav.Calibri().Theme, toastTextSize, "")
				decorate.Label(&l, "⬥ %s", p)
				l.Font.Style = font.Italic
				l.Alignment = text.Start
				l.Font.Weight = font.Thin
				return l.Layout(gtx)
			})
		}

		for _, p := range bulletin.Topics[i].Points {
			topics = append(topics, point(p))
		}

		return append(topics, layout.Rigid(layout.Spacer{Height: 25}.Layout))
	}

	children := []layout.FlexChild{}
	for i := range bulletin.Topics {
		children = append(children, topic(i)...)
	}

	list := material.List(
		t.nav.Calibri().Theme,
		&widget.List{
			Scrollbar: widget.Scrollbar{},
			List: layout.List{
				Axis:        layout.Vertical,
				ScrollToEnd: false,
				Alignment:   layout.Start,
			},
		},
	)

	// decorate.List(&list)
	decorate.Scrollbar(&list.ScrollbarStyle)

	first := true

	for {
		switch event := t.window.NextEvent().(type) {
		case app.ViewEvent, system.StageEvent:
			t.window.Perform(system.ActionRaise)
		case system.DestroyEvent:
			notify.Debug("[UI] Toast: DestroyEvent Newsletter \"%s\"", t.label.Text)
			return
		case system.FrameEvent:
			gtx := layout.NewContext(&t.ops, event)

			t.nav.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{
						Axis: layout.Horizontal,
					}.Layout(gtx,
						layout.Flexed(.1, layout.Spacer{Width: 0}.Layout),

						layout.Flexed(.8, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{
								Axis:      layout.Vertical,
								Alignment: layout.Middle,
							}.Layout(gtx,
								layout.Rigid(layout.Spacer{Height: 10}.Layout),

								layout.Flexed(.2, func(gtx layout.Context) layout.Dimensions {
									return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return decorate.Underline(gtx, decorate.Label(&t.label, "%s", t.label.Text).Layout)
									})
								}),

								layout.Flexed(.8, func(gtx layout.Context) layout.Dimensions {
									return decorate.UnderlineBorder(gtx, func(gtx layout.Context) layout.Dimensions {
										return list.Layout(gtx, 1, func(gtx layout.Context, index int) layout.Dimensions {
											return layout.Flex{
												Axis:      layout.Vertical,
												Alignment: layout.Middle,
												WeightSum: float32(len(children)),
											}.Layout(gtx, children...)
										})
									})
								}),

								layout.Rigid(layout.Spacer{Height: 10}.Layout),

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
						}),

						layout.Flexed(.1, layout.Spacer{Width: 0}.Layout),
					)
				})
			})

			if first {
				t.window.Perform(system.ActionCenter)
				t.window.Perform(system.ActionRaise)
				first = false
			}

			if t.keybinds.Escape(gtx, t) {
				t.window.Perform(system.ActionClose)
			}

			t.window.Invalidate()

			event.Frame(gtx.Ops)
		default:
			notify.Missed(event, "ToastNewsletter")
		}
	}
}

func (g *GUI) ToastOK(header, msg string, ok toastOnOK, c toastOnClose) {
	if c != nil {
		defer c()
	}

	t := g.makeToast(header, msg, float32(400), float32(125))
	if t == nil {
		notify.Error("[UI] Toast: Failed to open dialog")
		return
	}
	defer t.close()

	g.toastOK(t, ok)
}

func (g *GUI) ToastSplash(header, msg string, img image.Image) waiter {
	c := &closeable{
		toast: g.makeToastForce(header, msg, float32(640), float32(360)),
		waitq: make(chan bool),
	}
	if c.toast == nil {
		notify.Error("[UI] Toast: Failed to open Splash")
		return c
	}
	defer c.ready()

	go func() {
		defer c.toast.close()

		notify.Debug("[UI] Toast: Opening Splash \"%s\"", c.label.Text)
		defer notify.Debug("[UI] Toast: Closing Splash: \"%s\"", c.label.Text)

		c.toast.nav.Hide = true
		c.label.TextSize = toastTextSize * 1.5
		c.label.Color = nrgba.White.Color()

		op := paint.NewImageOp(img)

		for {
			switch event := c.toast.window.NextEvent().(type) {
			case system.DestroyEvent:
				c.toast.window.Perform(system.ActionClose)
				return
			case system.FrameEvent:
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
			default:
				notify.Missed(event, "ToastSplash")
			}
		}
	}()

	return c
}

// ToastYesNo shows a Yes/No dialog without a remember option.
func (g *GUI) ToastYesNo(header, msg string, y toastOnYes, n toastOnNo) {
	g.ToastYesNoRemember(header, msg, "", y, n, nil, nil)
}

// ToastYesNoRemember shows a Yes/No dialog with an optional remember checkbox.
func (g *GUI) ToastYesNoRemember(header, msg, decision string, y toastOnYes, n toastOnNo, c toastOnClose, r toastOnRemember) {
	if c != nil {
		defer c()
	}

	decision = ini.Format(decision)

	h := float32(125)
	if decision != "" {
		h = 150
	}

	t := g.makeToast(header, msg, 400, h)
	if t == nil {
		return
	}
	defer t.close()

	notify.Debug("[UI] Toast: Opening ToastYesNoRemember: \"%s\"", t.label.Text)
	defer notify.Debug("[UI] Toast: Closing ToastYesNoRemember: \"%s\"", t.label.Text)

	check := material.CheckBox(t.nav.Calibri().Theme, &widget.Bool{}, titleFirstWord(decision))

	yButton := &button.Widget{
		Text:            "Yes",
		TextSize:        unit.Sp(16),
		Font:            t.nav.Calibri(),
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
		Font:            t.nav.Calibri(),
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

	first := true

	for {
		switch event := t.window.NextEvent().(type) {
		case system.StageEvent:
			t.window.Perform(system.ActionRaise)
		case system.DestroyEvent:
			notify.Debug("[UI] Toast: DestroyEvent ToastYesNoRemember \"%s\"", t.label.Text)
			return
		case system.FrameEvent:
			gtx := layout.NewContext(&t.ops, event)

			t.nav.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{
						Axis:      layout.Vertical,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Rigid(layout.Spacer{Height: 10}.Layout),

						layout.Flexed(.5, func(gtx layout.Context) layout.Dimensions {
							return layout.Center.Layout(gtx, decorate.Label(&t.label, "%s", t.label.Text).Layout)
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

			if t.keybinds.Escape(gtx, t) {
				t.window.Perform(system.ActionClose)
			}

			t.window.Invalidate()

			event.Frame(gtx.Ops)
		default:
			notify.Missed(event, "ToastYesNoRemember")
		}
	}
}

// makeToast creates a toast if no other toast is active.
func (g *GUI) makeToast(header, msg string, width, height float32) *toast {
	msg = ini.Format(msg)
	header = ini.Format(header)

	notify.Debug("[Toast] %s: %s", header, msg)

	if g.previous.toast.active {
		return nil
	}
	g.previous.toast.active = true

	return g.makeToastForce(header, msg, width, height)
}

// makeToastForce creates a toast window regardless of existing toasts.
func (g *GUI) makeToastForce(header, msg string, width, height float32) *toast {
	t := &toast{
		g: g,

		window: app.NewWindow(
			app.Title(header),
			app.Decorated(false),
			app.Size(unit.Dp(width), unit.Dp(height)),
			app.MinSize(unit.Dp(width), unit.Dp(height)),
			app.MaxSize(unit.Dp(width), unit.Dp(height)),
		),

		forced: true,
		open:   true,

		keybinds: keys.New().Bind(keys.NoMod, keys.Escape()),
	}
	t.nav = title.New(header, nil, nil, func() {
		t.window.Perform(system.ActionClose)
	})
	t.nav.NoTip = true
	t.nav.NoDrag = true

	t.label = material.Label(t.nav.Calibri().Theme, toastTextSize, titleFirstWord(msg))
	t.label.Alignment = text.Middle

	return t
}

// toastOK runs the OK-only toast event loop.
func (g *GUI) toastOK(t *toast, ok toastOnOK) {
	notify.Debug("[UI] Toast: Opening OK \"%s\"", t.label.Text)
	defer notify.Debug("[UI] Toast: Closing OK: \"%s\"", t.label.Text)

	if len(t.label.Text) > 50 {
		t.window.Option(
			app.Size(unit.Dp(600), unit.Dp(125)),
			app.MinSize(unit.Dp(600), unit.Dp(125)),
			app.MaxSize(unit.Dp(600), unit.Dp(125)),
		)
	}

	okButton := &button.Widget{
		Text:            "OK",
		TextSize:        unit.Sp(16),
		Font:            t.nav.Calibri(),
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

	first := true

	for {
		switch event := t.window.NextEvent().(type) {
		case system.DestroyEvent:
			notify.Debug("[UI] Toast: DestroyEvent OK \"%s\"", t.label.Text)
			return
		case system.FrameEvent:
			gtx := layout.NewContext(&t.ops, event)

			t.nav.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return decorate.BackgroundAlt(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{
						Axis:      layout.Vertical,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Rigid(layout.Spacer{Height: 10}.Layout),

						layout.Flexed(.5, func(gtx layout.Context) layout.Dimensions {
							decorate.Label(&t.label, "%s", t.label.Text)
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

			if t.keybinds.Escape(gtx, t) {
				t.window.Perform(system.ActionClose)
			}

			event.Frame(gtx.Ops)
		default:
			notify.Missed(event, "ToastOk")
		}
	}
}

// titleFirstWord uppercases the first rune of a string.
func titleFirstWord(s string) string {
	if len(s) < 1 {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// close closes the toast and clears its active state.
func (t *toast) close() {
	defer notify.Debug("[UI] Toast: Closed \"%s\") (active: %t)", t.label.Text, t.g.previous.toast.active)

	t.window.Perform(system.ActionClose)

	t.g.previous.toast.err = nil
	t.g.previous.toast.active = false

	t.open = false
}

// ready signals that the splash toast is ready to be waited on.
func (c *closeable) ready() {
	go func() { c.waitq <- true }()
}

// wait blocks until the closeable toast is ready.
func (c *closeable) wait() waiter {
	<-c.waitq
	return c
}
