package area

import (
	"fmt"
	"image"
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

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/fonts"
	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/nrgba"
	"github.com/pidgy/unitehud/video"
	"github.com/pidgy/unitehud/video/device"
	"github.com/pidgy/unitehud/video/monitor"
	"github.com/pidgy/unitehud/video/proc"
	"github.com/pidgy/unitehud/video/window"
)

const alpha = 150

var (
	Locked = nrgba.Black
	Match  = nrgba.Green
	Miss   = nrgba.Red
)

type Area struct {
	Text          string
	TextSize      unit.Sp
	TextAlignLeft bool
	Subtext       string
	Hidden        bool
	Theme         *material.Theme

	*Capture

	Match    func(*Area) (bool, error)
	Cooldown time.Duration
	readyq   chan bool

	*button.Button

	Min, Max         image.Point
	baseMin, baseMax image.Point

	nrgba.NRGBA

	Drag, Focus bool

	lastDimsSize image.Point
	lastRelease  time.Time
	lastScale    float64

	matched struct {
		err error
		ok  bool
	}
}

type Capture struct {
	Option string
	File   string
	Base   image.Rectangle

	Matched *Area
}

func (a *Area) Layout(gtx layout.Context, dims layout.Constraints, img image.Image) (err error) {
	if img == nil || dims.Max.X == 0 || a.Base.Max.X == 0 {
		return nil
	}
	defer func() { err = a.match() }()

	if a.Button == nil {
		a.Button = &button.Button{}
	}

	if a.Theme == nil {
		a.Theme = fonts.Default().Theme
	}

	// Scale
	a.TextSize = unit.Sp(24) * unit.Sp(float32(dims.Max.X)/float32(img.Bounds().Max.X))

	rect := clip.Rect{
		Min: a.Min.Add(image.Pt(0, title.Height)),
		Max: a.Max.Add(image.Pt(0, title.Height)),
	}

	if a.Hidden {
		return nil
	}

	if !a.lastDimsSize.Eq(dims.Max) {
		minXScale := float32(a.Base.Min.X) / float32(img.Bounds().Max.X)
		maxXScale := float32(a.Base.Max.X) / float32(img.Bounds().Max.X)
		minYScale := float32(a.Base.Min.Y) / float32(img.Bounds().Max.Y)
		maxYScale := float32(a.Base.Max.Y) / float32(img.Bounds().Max.Y)

		a.Min.X = int(float32(dims.Max.X) * minXScale)
		a.Max.X = int(float32(dims.Max.X) * maxXScale)
		a.Min.Y = int(float32(dims.Max.Y) * minYScale)
		a.Max.Y = int(float32(dims.Max.Y) * maxYScale)

		a.lastDimsSize = dims.Max

		if a.lastScale == 0 {
			a.baseMin, a.baseMax = a.Min, a.Max
		}
	}

	if config.Current.Scale != a.lastScale {
		a.lastScale = config.Current.Scale

		a.Min = image.Pt(int((float64(a.baseMin.X) * config.Current.Scale)), int((float64(a.baseMin.Y) * config.Current.Scale)))
		a.Max = image.Pt(int((float64(a.baseMax.X) * config.Current.Scale)), int((float64(a.baseMax.Y) * config.Current.Scale)))
	}

	for _, ev := range gtx.Events(a) {
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}

		switch e.Type {
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

				a.Base.Min.X = int(baseMinXScale / float32(dims.Max.X))
				a.Base.Max.X = int(baseMaxXScale / float32(dims.Max.X))
				a.Base.Min.Y = int(baseMinYScale / float32(dims.Max.Y))
				a.Base.Max.Y = int(baseMaxYScale / float32(dims.Max.Y))
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
		Types: pointer.Press | pointer.Drag | pointer.Release | pointer.Leave | pointer.Enter | pointer.Move,
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
			title := material.Body1(a.Theme, a.Text)
			title.TextSize = a.TextSize
			title.Font.Weight = 500
			title.Color = nrgba.White.Color()
			layout.Inset{
				Left: unit.Dp(2),
				Top:  unit.Dp(1),
			}.Layout(gtx, title.Layout)

			sub := material.Body2(a.Theme, a.Subtext)

			// Scale.
			sub.TextSize = a.TextSize * unit.Sp(.75)
			sub.Font.Weight = 1000
			sub.Color = nrgba.White.Alpha(175).Color()

			layout.Inset{
				Left: unit.Dp(2),
				Top:  unit.Dp(unit.Sp(rect.Max.Sub(rect.Min).Y) - a.TextSize),
			}.Layout(gtx, sub.Layout)

			return layout.Dimensions{Size: rect.Max.Sub(rect.Min)}
		},
	)

	return
}

func (c *Capture) Rectangle() image.Rectangle {
	return c.Base
}

func (a *Area) Reset() {
	a.lastDimsSize = image.Pt(0, 0)
}

func (a *Area) match() error {
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

			a.Capture.Matched = nil
			if a.matched.ok {
				a.Capture.Matched = a
			}

			time.Sleep(a.Cooldown)
			a.readyq <- true
		}()
	default:
	}

	return a.matched.err
}

func (c *Capture) Open() error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("Failed to find current directory (%v)", err)
	}

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

	argv, err := syscall.UTF16PtrFromString(os.Getenv("windir") + "\\system32\\cmd.exe /C " + fmt.Sprintf("\"%s\\%s\"", dir, c.File))
	if err != nil {
		return fmt.Errorf("Failed to open %s (%v)", c.File, err)
	}

	var sI syscall.StartupInfo
	var pI syscall.ProcessInformation

	err = syscall.CreateProcess(nil, argv, nil, nil, true, proc.CreateNoWindow, nil, nil, &sI, &pI)
	if err != nil {
		return fmt.Errorf("Failed to open %s (%v)", c.File, err)
	}

	return nil
}
