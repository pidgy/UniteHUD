package area

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"syscall"
	"time"

	"gioui.org/font/gofont"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/gui/visual/button"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/video"
	"github.com/pidgy/unitehud/video/device"
	"github.com/pidgy/unitehud/video/screen"
	"github.com/pidgy/unitehud/video/window"
)

const alpha = 150

var (
	Locked = rgba.N(rgba.Alpha(rgba.Black, alpha))
	Match  = rgba.N(rgba.Alpha(rgba.DarkSeafoam, alpha))
	Miss   = rgba.N(rgba.Alpha(rgba.Red, alpha))
)

type Area struct {
	Text          string
	TextSize      unit.Value
	TextAlignLeft bool
	Subtext       string
	Hidden        bool
	Theme         *material.Theme

	*Capture

	Match    func(*Area) bool
	Cooldown time.Duration
	readyq   chan bool

	*button.Button

	Min, Max image.Point

	color.NRGBA

	Drag, Focus bool

	lastDimsSize image.Point

	lastRelease time.Time
}

type Capture struct {
	Option string
	File   string
	Base   image.Rectangle
}

func (a *Area) Layout(gtx layout.Context, dims layout.Dimensions, img image.Image) error {
	if img == nil || dims.Size.X == 0 || a.Base.Max.X == 0 {
		return nil
	}
	defer a.match()

	a.TextSize = unit.Px(24).Scale(float32(dims.Size.X) / float32(img.Bounds().Max.X))

	if a.Theme == nil {
		a.Theme = material.NewTheme(gofont.Collection())
	}

	if !a.lastDimsSize.Eq(dims.Size) {
		minXScale := float32(a.Base.Min.X) / float32(img.Bounds().Max.X)
		maxXScale := float32(a.Base.Max.X) / float32(img.Bounds().Max.X)
		minYScale := float32(a.Base.Min.Y) / float32(img.Bounds().Max.Y)
		maxYScale := float32(a.Base.Max.Y) / float32(img.Bounds().Max.Y)

		a.Min.X = int(float32(dims.Size.X) * minXScale)
		a.Max.X = int(float32(dims.Size.X) * maxXScale)
		a.Min.Y = int(float32(dims.Size.Y) * minYScale)
		a.Max.Y = int(float32(dims.Size.Y) * maxYScale)

		a.lastDimsSize = dims.Size
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
			if !a.Active {
				break
			}
		case pointer.Release:
			if a.Drag {
				a.Drag = false

				baseMinXScale := float32(a.Min.X) * float32(img.Bounds().Max.X)
				baseMaxXScale := float32(a.Max.X) * float32(img.Bounds().Max.X)
				baseMinYScale := float32(a.Min.Y) * float32(img.Bounds().Max.Y)
				baseMaxYScale := float32(a.Max.Y) * float32(img.Bounds().Max.Y)

				a.Base.Min.X = int(baseMinXScale / float32(dims.Size.X))
				a.Base.Max.X = int(baseMaxXScale / float32(dims.Size.X))
				a.Base.Min.Y = int(baseMinYScale / float32(dims.Size.Y))
				a.Base.Max.Y = int(baseMaxYScale / float32(dims.Size.Y))
			} else {
				s := time.Since(a.lastRelease)
				if s > time.Millisecond*100 && s < time.Millisecond*500 {
					err := a.Capture.Open()
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
			area := clip.Rect{
				Min: a.Min,
				Max: a.Max,
			}.Push(gtx.Ops)
			defer area.Pop()

			paint.ColorOp{Color: a.NRGBA}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)

			return layout.Dimensions{Size: a.Max.Sub(a.Min)}
		},
	)

	area := clip.Rect{
		Min: a.Min,
		Max: a.Max,
	}.Push(gtx.Ops)
	pointer.InputOp{
		Tag:   a,
		Types: pointer.Press | pointer.Drag | pointer.Release | pointer.Leave | pointer.Enter | pointer.Move,
		Grab:  a.Drag,
	}.Add(gtx.Ops)
	area.Pop()

	layout.Inset{
		Left: unit.Px(float32(a.Min.X)),
		Top:  unit.Px(float32(a.Min.Y)),
	}.Layout(
		gtx,
		func(gtx layout.Context) layout.Dimensions {
			c := a.NRGBA
			c.A = 255
			return widget.Border{
				Color: c,
				Width: unit.Px(2),
			}.Layout(
				gtx,
				func(gtx layout.Context) layout.Dimensions {
					defer clip.Rect{Min: a.Min, Max: a.Max}.Push(gtx.Ops).Pop()
					return layout.Dimensions{Size: a.Max.Sub(a.Min)}
				})
		})

	layout.Inset{
		Left: unit.Px(float32(a.Min.X)),
		Top:  unit.Px(float32(a.Min.Y)),
	}.Layout(
		gtx,
		func(gtx layout.Context) layout.Dimensions {
			title := material.Body1(a.Theme, a.Text)
			title.TextSize = a.TextSize
			title.Font.Weight = 500
			title.Color = rgba.N(rgba.White)
			layout.Inset{
				Left: unit.Px(2),
				Top:  unit.Px(1),
			}.Layout(gtx, title.Layout)

			sub := material.Body2(a.Theme, a.Subtext)
			sub.TextSize = a.TextSize.Scale(.75)
			sub.Font.Weight = 1000
			sub.Color = rgba.N(rgba.Alpha(rgba.White, 175))

			layout.Inset{
				Left: unit.Px(2),
				Top:  unit.Px(float32(a.Max.Sub(a.Min).Y) - a.TextSize.V),
			}.Layout(gtx, sub.Layout)

			return layout.Dimensions{Size: a.Max.Sub(a.Min)}
		},
	)

	return nil
}

func (c *Capture) Rectangle() image.Rectangle {
	return c.Base
}

func (a *Area) Reset() {
	a.lastDimsSize = image.Pt(0, 0)
}

func (a *Area) match() {
	if a.Drag || a.Focus {
		return
	}

	if a.readyq == nil {
		a.readyq = make(chan bool)
		go func() { a.readyq <- true }()
	}

	if !device.IsActive() && !screen.IsDisplay() && !window.IsWindow() {
		return
	}

	select {
	case <-a.readyq:
		go func() {
			a.Match(a)
			time.Sleep(a.Cooldown)
			a.readyq <- true
		}()
	default:
	}
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

	var sI syscall.StartupInfo
	var pI syscall.ProcessInformation
	argv := syscall.StringToUTF16Ptr(os.Getenv("windir") + "\\system32\\cmd.exe /C " +
		fmt.Sprintf("\"%s\\%s\"", dir, c.File))

	err = syscall.CreateProcess(nil, argv, nil, nil, true, 0, nil, nil, &sI, &pI)
	if err != nil {
		return fmt.Errorf("Failed to open %s (%v)", c.File, err)
	}

	return nil
}
