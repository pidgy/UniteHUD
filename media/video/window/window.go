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
	winRT         *d3d11.Window
	dx11          *d3d11.Desktop
)

func Active() win32.Window {
	i := slices.Index(titles, config.Current.Video.Capture.Window.Name)
	if i == -1 {
		return win32.Window(0)
	}
	return sources[i].Window
}

func ActiveName() string {
	i := slices.Index(titles, config.Current.Video.Capture.Window.Name)
	if i == -1 {
		return ""
	}
	return titles[i]
}

// Capture captures the desired area from a Window and returns an image.
func Capture() (*image.RGBA, error) {
	if config.Current.Video.Capture.Window.Method != config.CaptureMethodWinRT && winRT != nil {
		notify.Debug("[Window] Closing WinRT capture")
		winRT.Close()
		winRT = nil
	}
	if config.Current.Video.Capture.Window.Method != config.CaptureMethodDirectX11 && dx11 != nil {
		notify.Debug("[Window] Closing DirectX 11 capture")
		dx11.Close()
		dx11 = nil
	}

	w := Active()

	r32, err := w.RectClient()
	if err != nil {
		return nil, fmt.Errorf("failed to determine client area")
	}
	area := r32.Image()

	switch config.Current.Video.Capture.Window.Method {
	case config.CaptureMethodDefault:
		if !r32.Eq(monitor.DefaultResolution32) {
			return w.CaptureResized(area, monitor.DefaultResolution.Size())
		}
		return w.Capture(area)
	case config.CaptureMethodDirectX11:
		if dx11 == nil {
			notify.Debug("[Window] Starting DirectX 11 capture")

			m, err := w.Monitor()
			if err != nil {
				return nil, err
			}

			dx11, err = d3d11.NewDesktop(m.HWND)
			if err != nil {
				return nil, err
			}
		}

		return dx11.Capture(area)
	case config.CaptureMethodWinRT:
		if winRT == nil {
			notify.Debug("[Window] Starting WinRT capture")

			winRT, err = d3d11.NewWindow(w.HWND())
			if err != nil {
				return nil, err
			}
		}

		return winRT.Capture()
	case config.CaptureMethodWin32:
		return w.CapturePrintWindow(area)
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
	notify.Debug("[Window] Closing captures (DirectX 11: %t) (WinRT: %t)", dx11 != nil, winRT != nil)

	config.Current.SetDefaultWindowCapture()

	if dx11 != nil {
		dx11.Close()
		dx11 = nil
	}

	if winRT != nil {
		winRT.Close()
		winRT = nil
	}
}

func IsActive() bool {
	return Active() != win32.Window(0)
}

func Name(index int) string {
	if len(titles) > index {
		return ""
	}
	return titles[index]
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
