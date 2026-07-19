package window

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const. Then remove this comment.

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"time"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/media/video/monitor"
	"github.com/pidgy/unitehud/system/win32"
	"github.com/pidgy/unitehud/system/win32/d3d11"
)

var (
	titles        = []string{}
	sources       = []win32.WindowInfoEx{}
	ErrFailedFind = fmt.Errorf("failed to find window")
	d3d           *d3d11.Capture
)

func Active() (w win32.Window) {
	i := slices.Index(titles, config.Current.Video.Capture.Window.Name)
	if i == -1 {
		return win32.Window(0)
	}
	return sources[i].Window
}

// Capture captures the desired area from a Window and returns an image.
func Capture() (*image.RGBA, error) {
	w := Active()
	switch config.Current.Video.Capture.Window.Method {
	case config.CaptureMethodDefault:
		if d3d != nil {
			notify.Debug("[Window] Window Direct3D 11 capture")
			d3d.Close()
			d3d = nil
		}

		r32, err := w.RectClient()
		if err != nil {
			return nil, fmt.Errorf("failed to determine client area")
		}

		return w.Capture(r32.Image(), monitor.DefaultResolution.Size())
	case config.CaptureMethodDirect3D11:
		if d3d == nil {
			notify.Debug("[Window] Starting Direct3D 11 capture")

			m, err := w.Monitor()
			if err != nil {
				return nil, err
			}

			c, err := d3d11.NewCapture(m.HWND)
			if err != nil {
				return nil, err
			}
			d3d = c
		}

		return d3d.CaptureWindow(w.Info().Window.Image())
	default:
		return nil, fmt.Errorf("unknown window capture method")
	}
}

func CaptureRect(r image.Rectangle) (*image.RGBA, error) {
	img, err := Capture()
	if err != nil {
		return nil, err
	}

	return img.SubImage(r).(*image.RGBA), nil
}

func Close() {
	if d3d != nil {
		d3d.Close()
		d3d = nil
	}
}

func IsActive() bool {
	return Active() != win32.Window(0)
}

func Open() error {
	err := background()
	if err != nil {
		return err
	}

	if config.Current.Video.Capture.Window.Name == config.NoDeviceName {
		return nil
	}

	if !slices.ContainsFunc(sources, func(w win32.WindowInfoEx) bool { return w.Title == config.Current.Video.Capture.Window.Name }) {
		notify.Error("[Window] <ini:f:find> \"%s\"", config.Current.Video.Capture.Window.Name)
		return ErrFailedFind
	}

	return nil
}

func Resolution() image.Point {
	r32, err := Active().RectClient()
	if err != nil {
		return image.Point{}
	}
	return r32.Image().Size()
}

func Titles() []string {
	return titles
}

func background() error {
	fn := func() error {
		windows, err := win32.FindWindows()
		if err != nil {
			return err
		}

		s := []win32.WindowInfoEx{}
		t := []string{}

		for _, w := range windows.Infos {
			if w.WindowInfo.HasVisibleStyle() && !strings.Contains(w.Title, exe.Title) {
				s = append(s, w)
				t = append(t, w.Title)
			}
		}

		sources = s
		titles = t

		return nil
	}

	go func() {
		for ; ; time.Sleep(time.Second * 5) {
			err := fn()
			if err != nil {
				notify.Error("<ini:f:find> active windows in background (%v)", err)
			}
		}
	}()

	return fn()
}
