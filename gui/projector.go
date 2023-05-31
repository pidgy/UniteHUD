package gui

import (
	"fmt"
	"image"
	"time"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/audio"
	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/gui/visual/dropdown"
	"github.com/pidgy/unitehud/nrgba"
	"github.com/pidgy/unitehud/splash"
	"github.com/pidgy/unitehud/video"
	"github.com/pidgy/unitehud/video/device"
	"github.com/pidgy/unitehud/video/screen"
)

var projecting bool

func (g *GUI) Project() {
	w := app.NewWindow(
		app.Title(config.ProjectorWindow),
		app.Size(unit.Dp(1280), unit.Dp(720)),
		app.WindowMode.Option(app.Windowed),
	)

	var ops op.Ops

	session, err := audio.New(audio.DeviceDisabled, audio.DeviceDisabled)
	if err != nil {
		g.ToastOK("UniteHUD Audio Device Error", fmt.Sprintf("Failed to route audio input to output (%v)", err))
		return
	}

	capList := &dropdown.List{
		WidthModifier: 1,
		Items: []*dropdown.Item{
			{
				Text:    audio.DeviceDisabled,
				Checked: widget.Bool{Value: true},
				Callback: func(i *dropdown.Item) {
					err := session.SetCapture(audio.DeviceDisabled)
					if err != nil {
						g.ToastError(err)
						return
					}
					i.Checked.Value = true
				},
			},
			{
				Text: audio.DeviceDefault,
				Callback: func(i *dropdown.Item) {
					err := session.SetCapture(audio.DeviceDefault)
					if err != nil {
						g.ToastError(err)
						return
					}
					i.Checked.Value = true
				},
			},
		},
		Callback: func(i *dropdown.Item, d *dropdown.List) {
			for _, item := range d.Items {
				item.Checked.Value = false
				if item == i {
					item.Checked.Value = true
				}
			}
		},
	}

	playList := &dropdown.List{
		WidthModifier: 1,
		Items: []*dropdown.Item{
			{
				Text:    audio.DeviceDisabled,
				Checked: widget.Bool{Value: true},
				Callback: func(i *dropdown.Item) {
					err := session.SetPlayback(audio.DeviceDisabled)
					if err != nil {
						g.ToastError(err)
						return
					}
					i.Checked.Value = true
				},
			},
			{
				Text: audio.DeviceDefault,
				Callback: func(i *dropdown.Item) {
					err := session.SetPlayback(audio.DeviceDefault)
					if err != nil {
						g.ToastError(err)
						return
					}
					i.Checked.Value = true
				},
			},
		},
		Callback: func(i *dropdown.Item, d *dropdown.List) {
			for _, item := range d.Items {
				item.Checked.Value = false
				if item == i {
					item.Checked.Value = true
				}
			}
		},
	}

	caps, plays := audio.DeviceNames()

	for _, name := range caps {
		capList.Items = append(capList.Items, &dropdown.Item{
			Text: name,
			Callback: func(i *dropdown.Item) {
				err := session.SetCapture(i.Text)
				if err != nil {
					g.ToastError(err)
					return
				}

				i.Checked.Value = true
			},
		})
	}

	for _, name := range plays {
		playList.Items = append(playList.Items, &dropdown.Item{
			Text: name,
			Callback: func(i *dropdown.Item) {
				err := session.SetPlayback(i.Text)
				if err != nil {
					g.ToastError(err)
					return
				}

				i.Checked.Value = true
			},
		})
	}

	fullscreen := false

	p := projected{
		img:   splash.Invalid(),
		tag:   new(bool),
		theme: g.normal,
		since: time.Now(),
		hover: true,
	}

	w.Perform(system.ActionRaise)
	w.Perform(system.ActionCenter)

	for e := range w.Events() {
		size := float32(0.0)
		if p.hover {
			size = 0.2
		}

		switch e := e.(type) {
		case app.ViewEvent:
		case system.DestroyEvent:
			projecting = false

			err := session.Close()
			if err != nil {
				g.ToastErrorForce(err)
			}

			return
		case system.FrameEvent:
			err := session.Error()
			if err != nil {
				if err != audio.SessionClosed {
					g.ToastError(err)
					return
				}
			}

			//ops.Reset()
			gtx := layout.NewContext(&ops, e)

			p.cursor.Add(gtx.Ops)
			p.q = gtx.Queue

			colorBox(gtx, gtx.Constraints.Max, nrgba.DarkGray)

			{
				layout.Flex{
					Spacing:   layout.SpaceEnd,
					Alignment: layout.Baseline,
					Axis:      layout.Vertical,
				}.Layout(gtx,
					layout.Flexed(0.8, func(gtx layout.Context) layout.Dimensions {
						colorBox(gtx, gtx.Constraints.Max, nrgba.BackgroundAlt)

						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return p.Layout(gtx)
						})
					}),
					//g.drawProjectorScreen(gtx, p.img),
					layout.Flexed(size, func(gtx layout.Context) layout.Dimensions {
						colorBox(gtx, gtx.Constraints.Max, nrgba.Background)

						return layout.Flex{
							Axis: layout.Horizontal,
						}.Layout(gtx,
							p.drawSpacer(5, 5),

							layout.Flexed(0.33, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										label := material.Label(p.theme, unit.Sp(14), "Audio In (Capture)")
										label.Color = nrgba.Highlight.Color()
										label.Font.Weight = 100

										return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return label.Layout(gtx)
											})
										})
									}),
									layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
										return capList.Layout(gtx, p.theme)
									}),
								)
							}),

							p.drawSpacer(5, 5),

							layout.Flexed(0.33, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										label := material.Label(p.theme, unit.Sp(14), "Audio Out (Playback)")
										label.Color = nrgba.Highlight.Color()
										label.Font.Weight = 100

										return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return label.Layout(gtx)
											})
										})
									}),
									layout.Flexed(.9, func(gtx layout.Context) layout.Dimensions {
										return playList.Layout(gtx, p.theme)
									}),
								)
							}),

							p.drawSpacer(5, 5),

							layout.Flexed(0.1, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										label := material.Label(p.theme, unit.Sp(12), g.cpu)
										label.Color = nrgba.Highlight.Color()
										label.Font.Weight = 100
										label.Alignment = text.Start

										return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return label.Layout(gtx)
											})
										})
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										label := material.Label(p.theme, unit.Sp(12), g.ram)
										label.Color = nrgba.Highlight.Color()
										label.Font.Weight = 100
										label.Alignment = text.Start

										return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return label.Layout(gtx)
											})
										})
									}),
								)
							}),
						)
					}),
				)
			}

			if device.IsActive() || screen.IsDisplay() {
				p.img, err = video.Capture()
				if err != nil {
					g.ToastError(err)
					return
				}
			} else {
				p.img = splash.Invalid()
			}

			//w.Invalidate()

			w.Invalidate()

			e.Frame(gtx.Ops)
		case app.ConfigEvent:
			println(e.Config.Size.String())
		case key.Event:
			if e.State != key.Release {
				continue
			}

			switch e.Name {
			case "F11":
				if !fullscreen {
					println("F11", "fullscreen =", fullscreen, "->", !fullscreen)
					fullscreen = true
					w.Perform(system.ActionFullscreen)
					break
				}
				fallthrough
			case key.NameEscape:
				if fullscreen {
					println("ESC", "fullscreen =", fullscreen, "->", !fullscreen)
					fullscreen = false
					w.Perform(system.ActionUnmaximize)

					break
				}
			default:
				continue
			}
		case pointer.Event:

		case system.StageEvent:
		default:
		}

	}
}

type projected struct {
	img image.Image

	hover  bool
	since  time.Time
	cursor pointer.Cursor

	theme *material.Theme

	tag *bool
	q   event.Queue
}

func (p *projected) Layout(gtx layout.Context) layout.Dimensions {
	return widget.Image{
		Src:   paint.NewImageOp(p.img),
		Scale: float32(gtx.Constraints.Max.Y) / float32(p.img.Bounds().Max.Y),
	}.Layout(gtx)

	for _, ev := range p.q.Events(p.tag) {
		_, ok := ev.(pointer.Event)
		if !ok {
			continue
		}

		p.hover = true
		p.since = time.Now()
		p.cursor = pointer.CursorDefault
	}

	if time.Since(p.since) > time.Second*3 {
		p.hover = false
		p.cursor = pointer.CursorNone
	}

	area := clip.Rect{
		Min: gtx.Constraints.Min,
		Max: gtx.Constraints.Max,
	}.Push(gtx.Ops)
	pointer.InputOp{
		Tag:   p.tag,
		Types: pointer.Move,
	}.Add(gtx.Ops)
	area.Pop()

	return widget.Image{
		Src:   paint.NewImageOp(p.img),
		Scale: float32(gtx.Constraints.Max.Y) / float32(p.img.Bounds().Max.Y),
	}.Layout(gtx)
}

func (p *projected) drawSpacer(x, y float32) layout.FlexChild {
	return layout.Rigid(layout.Spacer{Width: unit.Dp(x), Height: unit.Dp(y)}.Layout)
}
