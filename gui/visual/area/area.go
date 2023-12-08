package area

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"syscall"
	"time"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/core/fonts"
	"github.com/pidgy/unitehud/core/global"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/nrgba"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/decorate"
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/media/video"
	"github.com/pidgy/unitehud/media/video/device"
	"github.com/pidgy/unitehud/media/video/monitor"
	"github.com/pidgy/unitehud/media/video/wapi"
	"github.com/pidgy/unitehud/media/video/window"
)

const alpha = 150

var (
	Locked = nrgba.Black
	Match  = nrgba.Green
	Miss   = nrgba.Red
)

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

	Drag, Focus bool

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
}

type Capture struct {
	Option      string
	File        string
	Base        image.Rectangle
	DefaultBase image.Rectangle

	MatchedColor color.NRGBA
	MatchedText  string
}

func (a *Widget) Layout(gtx layout.Context, collection fonts.Collection, capture image.Rectangle, img image.Image, blank image.Point) (err error) {
	if img == nil || capture.Max.X == 0 || a.Base.Max.X == 0 {
		return nil
	}
	defer func() { err = a.match() }()

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
		decorate.Label(&a.titleLabel, a.titleLabel.Text)

		a.subtitleLabel = material.Body2(a.Theme, "")
		a.subtitleLabel.Font.Weight = 1000
		decorate.Label(&a.subtitleLabel, a.subtitleLabel.Text)
	}

	// Scale up or down based on area and image size.
	a.TextSize = unit.Sp(24) * unit.Sp(float32(capture.Max.X)/float32(img.Bounds().Max.X))

	rect := clip.Rect{
		Min: a.Min.Add(image.Pt(0, title.Height)),
		Max: a.Max.Add(image.Pt(0, title.Height)),
	}

	if a.Hidden {
		return nil
	}

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
			a.Focus = true
			a.NRGBA = Locked
			a.NRGBA.A = 0
		case pointer.Leave:
			a.Focus = false
			a.NRGBA.A = alpha
		case pointer.Cancel:
		case pointer.Press:
		case pointer.Release:
			if a.Drag {
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
			if !a.Drag {
				break
			}
			fallthrough
		case pointer.Drag:
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

			paint.ColorOp{Color: a.Alpha(alpha).Color()}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)

			return layout.Dimensions{Size: rect.Max.Sub(rect.Min)}
		},
	)

	area := rect.Push(gtx.Ops)
	pointer.InputOp{
		Tag:   a,
		Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Leave | pointer.Enter | pointer.Move,
		Grab:  a.Drag,
	}.Add(gtx.Ops)
	area.Pop()

	if !a.Hidden {
		layout.Inset{
			Left: unit.Dp(rect.Min.X),
			Top:  unit.Dp(rect.Min.Y),
		}.Layout(
			gtx,
			func(gtx layout.Context) layout.Dimensions {
				return widget.Border{
					Color: a.Alpha(255).Color(),
					Width: unit.Dp(2),
				}.Layout(
					gtx,
					func(gtx layout.Context) layout.Dimensions {
						defer rect.Push(gtx.Ops).Pop()
						return layout.Dimensions{Size: rect.Max.Sub(rect.Min)}
					})
			})
	}

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

func (a *Widget) match() error {
	if a.Drag || a.Focus {
		return nil
	}

	if a.readyq == nil {
		a.readyq = make(chan bool)
		go func() { a.readyq <- true }()
	}

	if !device.IsActive() && !monitor.IsDisplay() && !window.IsOpen() {
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
				a.Capture.MatchedText = fmt.Sprintf("%s (%s)", a.Text, a.Subtext)
			}

			time.Sleep(a.Cooldown)

			a.readyq <- true
		}()
	default:
	}

	return a.matched.err
}

func (c *Capture) reset() {
	notify.Debug("🖥️ Resetting %s capture area %s", c.Option, c.DefaultBase)
	c.Base = c.DefaultBase
}

func (c *Capture) Open() error {
	img, err := video.CaptureRect(c.Base)
	if err != nil {
		return fmt.Errorf("Failed to capture %s (%v)", c.File, err)
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		return fmt.Errorf("Failed to create %s (%v)", c.File, err)
	}
	defer matrix.Close()

	if !gocv.IMWrite(c.File, matrix) {
		return fmt.Errorf("Failed to save %s (%v)", c.File, err)
	}

	argv, err := syscall.UTF16PtrFromString(os.Getenv("windir") + "\\system32\\cmd.exe /C " + fmt.Sprintf("\"%s\\%s\"", global.WorkingDirectory(), c.File))
	if err != nil {
		return fmt.Errorf("Failed to open %s (%v)", c.File, err)
	}

	var sI syscall.StartupInfo
	var pI syscall.ProcessInformation

	err = syscall.CreateProcess(nil, argv, nil, nil, true, wapi.CreateProcessFlags.NoWindow, nil, nil, &sI, &pI)
	if err != nil {
		return fmt.Errorf("Failed to open %s (%v)", c.File, err)
	}

	return nil
}

func (c *Capture) Rectangle() image.Rectangle {
	return c.Base
}
