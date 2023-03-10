package gui

import (
	"fmt"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/video"
	"github.com/pidgy/unitehud/video/device"
	"github.com/pidgy/unitehud/video/screen"
)

var projecting bool

func (g *GUI) Project() {
	w := app.NewWindow(
		app.Title(config.ProjectorWindow),
		app.Size(unit.Px(1280), unit.Px(720)),
		app.WindowMode.Option(app.Windowed),
	)

	var ops op.Ops

	mat1 := gocv.IMRead(fmt.Sprintf(`%s/splash/invalid.png`, config.Current.Assets()), gocv.IMReadColor)
	splash, err := mat1.ToImage()
	if err != nil {
		g.ToastError(err)
		return
	}

	mat2 := gocv.IMRead(fmt.Sprintf(`%s/splash/invalid.png`, config.Current.Assets()), gocv.IMReadColor)
	invalid, err := mat2.ToImage()
	if err != nil {
		g.ToastError(err)
		return
	}

	img := splash

	go func() {
		for e := range w.Events() {
			switch e := e.(type) {
			case app.ViewEvent:
				notify.System("%s is now open", config.ProjectorWindow)
			case system.DestroyEvent:
				projecting = false
				return
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)

				widget.Image{
					Src:   paint.NewImageOp(img),
					Scale: float32(e.Size.X) / float32(img.Bounds().Max.X),
					// +(float32(img.Bounds().Dy()) / float32(gtx.Constraints.Max.Y))) / 2,
				}.Layout(gtx)

				if device.IsActive() || screen.IsDisplay() {
					img, err = video.Capture()
					if err != nil {
						g.ToastError(err)
						return
					}
				} else {
					img = invalid
				}

				e.Frame(gtx.Ops)

				w.Invalidate()
			case app.ConfigEvent:
			case key.Event:
				if e.State != key.Release {
					continue
				}

				switch e.Name {
				case "F11":
					w.Option(app.WindowMode.Option(app.Fullscreen))
					w.Invalidate()
				case key.NameEscape:
					w.Option(app.WindowMode.Option(app.Windowed))
					w.Close()
				default:
					continue
				}
			case pointer.Event:
			case system.StageEvent:
			default:
			}
		}
	}()
}
