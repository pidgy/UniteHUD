package area

import (
	"fmt"
	"image"
	"time"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/core/fonts"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/gui/cursor"
	"github.com/pidgy/unitehud/gui/ux/button"
	"github.com/pidgy/unitehud/gui/ux/decorate"
	"github.com/pidgy/unitehud/gui/ux/title"
	"github.com/pidgy/unitehud/media/video/device"
	"github.com/pidgy/unitehud/media/video/monitor"
	"github.com/pidgy/unitehud/media/video/window"
)

var (
	// Locked is the default color for inactive or locked areas.
	Locked = nrgba.Black
	// Match is the highlight color for successful matches.
	Match = nrgba.Green
	// Miss is the highlight color for failed matches.
	Miss = nrgba.Red
)

// Widget represents an interactive capture area that can be dragged and matched.
type Widget struct {
	Text          string
	TextSize      unit.Sp
	TextAlignLeft bool
	Subtext       string
	Hidden        bool
	Theme         *material.Theme

	*Capture

	Match    func(*Widget) (bool, error)
	Cooldown time.Duration
	readyq   chan bool

	*button.Widget

	Min, Max         image.Point
	baseMin, baseMax image.Point

	nrgba.NRGBA

	Drag, Draggable, Focus bool

	lastDimsSize image.Point
	lastRelease  time.Time
	lastScale    float64
	baseMinY     int

	titleLabel    material.LabelStyle
	subtitleLabel material.LabelStyle

	matched struct {
		err error
		ok  bool
	}

	frameFrequency int
}

// Layout lays out and renders the widget.
func (a *Widget) Layout(gtx layout.Context, collection *fonts.Collection, capture image.Rectangle, img image.Image, blank image.Point) (err error) {
	if img == nil || capture.Max.X == 0 || a.Base.Max.X == 0 {
		return nil
	}
	defer func() {
		a.frameFrequency++
		if a.frameFrequency >= 120 {
			a.frameFrequency = 0
			err = a.match()
		}
	}()

	if a.Widget == nil {
		a.Widget = &button.Widget{
			Font: collection.Calibri(),
		}
	}

	if a.Theme == nil {
		a.Theme = collection.Calibri().Theme
	}

	if a.titleLabel.TextSize == 0 {
		a.titleLabel = material.Body1(a.Theme, "")
		a.titleLabel.Font.Weight = 500
		decorate.Label(&a.titleLabel, "%s", a.titleLabel.Text)

		a.subtitleLabel = material.Body2(a.Theme, "")
		a.subtitleLabel.Font.Weight = 1000
		decorate.Label(&a.subtitleLabel, "%s", a.subtitleLabel.Text)

		a.frameFrequency = 120
	}

	// Scale up or down based on area and image size.
	a.TextSize = unit.Sp(24) * unit.Sp(float32(capture.Max.X)/float32(img.Bounds().Max.X))

	rect := clip.Rect{
		Min: a.Min.Add(image.Pt(0, title.Height)),
		Max: a.Max.Add(image.Pt(0, title.Height)),
	}

	// if a.Hidden {
	// 	return nil
	// }

	if a.baseMinY == 0 {
		a.baseMinY = capture.Min.Y
	}

	if !a.lastDimsSize.Eq(capture.Max) {
		a.lastDimsSize = capture.Max

		scale := float32(0)
		if blank.X > blank.Y {
			scale = float32(capture.Dy()) / float32(img.Bounds().Max.Y)
		} else {
			scale = float32(capture.Dx()) / float32(img.Bounds().Max.X)
		}

		a.Min.X = int(float32(a.Base.Min.X) * scale)
		a.Max.X = int(float32(a.Base.Max.X) * scale)
		a.Min.Y = int(float32(a.Base.Min.Y) * scale)
		a.Max.Y = int(float32(a.Base.Max.Y) * scale)

		if blank.X > 0 {
			a.Min.X += blank.X
			a.Max.X += blank.X
		}

		if blank.Y > 0 {
			a.Min.Y += blank.Y
			a.Max.Y += blank.Y
		}

		if a.lastScale == 0 {
			a.baseMin, a.baseMax = a.Min, a.Max
		}
	}

	for _, ev := range gtx.Events(a) {
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}

		switch e.Kind {
		case pointer.Enter:
			if a.Hidden {
				cursor.Is(pointer.CursorNotAllowed)
				continue
			}

			if !a.Draggable {
				cursor.Is(pointer.CursorNotAllowed)
				continue
			}

			cursor.Is(pointer.CursorPointer)

			a.Focus = true
			a.NRGBA = Locked
			a.NRGBA.A = 0
		case pointer.Leave:
			cursor.Is(pointer.CursorDefault)

			if a.Hidden {
				continue
			}

			a.Focus = false
			a.NRGBA.A = a.opacity()
		case pointer.Cancel:
		case pointer.Press:
			if a.Hidden {
				cursor.Is(pointer.CursorNotAllowed)
				continue
			}

			if !a.Draggable {
				cursor.Is(pointer.CursorNotAllowed)
				continue
			}

			cursor.Is(pointer.CursorCrosshair)
		case pointer.Release:
			if a.Hidden {
				continue
			}

			if a.Draggable && a.Drag {
				a.Drag = false

				baseMinXScale := float32(a.Min.X) * float32(img.Bounds().Max.X)
				baseMaxXScale := float32(a.Max.X) * float32(img.Bounds().Max.X)
				baseMinYScale := float32(a.Min.Y) * float32(img.Bounds().Max.Y)
				baseMaxYScale := float32(a.Max.Y) * float32(img.Bounds().Max.Y)

				a.Base.Min.X = int(baseMinXScale/float32(capture.Max.X)) - capture.Min.X
				a.Base.Max.X = int(baseMaxXScale/float32(capture.Max.X)) - capture.Min.X
				a.Base.Min.Y = int(baseMinYScale/float32(capture.Max.Y)) - capture.Min.Y
				a.Base.Max.Y = int(baseMaxYScale/float32(capture.Max.Y)) - capture.Min.Y

				if blank.Y > 0 {
					a.Base.Min.Y += blank.Y
					a.Base.Max.Y += blank.Y
				}
			} else {
				s := time.Since(a.lastRelease)
				if s > time.Millisecond*100 && s < time.Millisecond*500 {
					err = a.Capture.Open()
					if err != nil {
						return err
					}
				}
				a.lastRelease = time.Now()
			}
		case pointer.Move:
			if a.Hidden {
				continue
			}

			if !a.Draggable {
				cursor.Is(pointer.CursorNotAllowed)
				continue
			}

			if !a.Drag {
				break
			}

			fallthrough
		case pointer.Drag:
			if a.Hidden {
				continue
			}

			if !a.Draggable {
				cursor.Is(pointer.CursorNotAllowed)
				continue
			}

			cursor.Is(pointer.CursorCrosshair)

			a.Drag = true

			e.Position.Y -= float32(title.Height)

			half := a.Max.Sub(a.Min).Div(2)
			a.Min = image.Pt(int(e.Position.X)-half.X, int(e.Position.Y)-half.Y)
			a.Max = image.Pt(int(e.Position.X)+half.X, int(e.Position.Y)+half.Y)
		}
	}

	layout.UniformInset(unit.Dp(0)).Layout(
		gtx,
		func(gtx layout.Context) layout.Dimensions {
			area := rect.Push(gtx.Ops)
			defer area.Pop()

			paint.ColorOp{Color: a.Alpha(a.opacity()).Color()}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)

			return layout.Dimensions{Size: rect.Max.Sub(rect.Min)}
		},
	)

	area := rect.Push(gtx.Ops)
	pointer.InputOp{
		Tag:   a,
		Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Leave | pointer.Enter | pointer.Move,
		Grab:  !a.Hidden,
	}.Add(gtx.Ops)
	area.Pop()

	// if !a.Hidden {
	layout.Inset{
		Left: unit.Dp(rect.Min.X),
		Top:  unit.Dp(rect.Min.Y),
	}.Layout(
		gtx,
		func(gtx layout.Context) layout.Dimensions {
			return widget.Border{
				Color: a.Alpha(a.opacity()).Color(),
				Width: unit.Dp(2),
			}.Layout(
				gtx,
				func(gtx layout.Context) layout.Dimensions {
					defer rect.Push(gtx.Ops).Pop()
					return layout.Dimensions{Size: rect.Max.Sub(rect.Min)}
				})
		})
	// }

	layout.Inset{
		Left: unit.Dp(rect.Min.X),
		Top:  unit.Dp(rect.Min.Y),
	}.Layout(
		gtx,
		func(gtx layout.Context) layout.Dimensions {
			a.titleLabel.TextSize = a.TextSize
			a.titleLabel.Text = a.Text
			layout.Inset{
				Left: unit.Dp(2),
				Top:  unit.Dp(1),
			}.Layout(gtx, a.titleLabel.Layout)

			a.subtitleLabel.TextSize = a.TextSize
			a.subtitleLabel.Text = a.Subtext
			layout.Inset{
				Left: unit.Dp(2),
				Top:  unit.Dp(unit.Sp(rect.Max.Sub(rect.Min).Y) - a.TextSize),
			}.Layout(gtx, a.subtitleLabel.Layout)

			return layout.Dimensions{Size: rect.Max.Sub(rect.Min)}
		},
	)

	return
}

func (a *Widget) Reset() {
	a.lastDimsSize = image.Pt(0, 0)
	a.Capture.reset()
}

// match runs the configured matcher and updates capture match metadata.
func (a *Widget) match() error {
	if a.Drag || a.Focus {
		return nil
	}

	if a.readyq == nil {
		a.readyq = make(chan bool, 1)
		a.readyq <- true
	}

	if !device.IsActive() && !monitor.IsActive() && !window.IsActive() {
		return nil
	}

	select {
	case <-a.readyq:
		go func() {
			a.matched.ok, a.matched.err = a.Match(a)

			a.Capture.MatchedColor = Miss.Color()
			a.Capture.MatchedText = a.Capture.Option
			if a.matched.ok {
				a.Capture.MatchedColor = Match.Color()
				a.Capture.MatchedText = fmt.Sprintf("%s: %s", a.Text, a.Subtext)
			}

			time.Sleep(a.Cooldown)

			a.readyq <- true
		}()
	default:
	}

	return a.matched.err
}

// opacity returns the base alpha for area rendering.
func (a *Widget) opacity() uint8 {
	alpha := uint8(150)

	if a.Hidden {
		return alpha / 8
	}

	return alpha
}
