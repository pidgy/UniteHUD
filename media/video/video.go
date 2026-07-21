package video

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const. Then remove this comment.

import (
	"fmt"
	"image"
	"slices"

	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/media/img/splash"
	"github.com/pidgy/unitehud/media/video/device"
	"github.com/pidgy/unitehud/media/video/monitor"
	"github.com/pidgy/unitehud/media/video/window"
)

type Source string

const (
	Monitor Source = "Monitor"
	Device  Source = "Video Capture Device"
	Window  Source = "Window"
	Unknown Source = "Unknown"
)

func Active() Source {
	switch {
	case device.IsActive():
		return Device
	case window.IsActive():
		return Window
	case monitor.IsActive():
		return Monitor
	default:
		return Unknown
	}
}

func Capture() (*image.RGBA, error) {
	switch Active() {
	case Device:
		return device.Capture()
	case Window:
		return window.Capture()
	case Monitor:
		return monitor.Capture()
	case Unknown:
		fallthrough
	default:
		return nil, fmt.Errorf("failed to capture video: exhausted sources")
	}
}

func CaptureRect(r image.Rectangle) (*image.RGBA, error) {
	switch Active() {
	case Device:
		return device.CaptureRect(r)
	case Window:
		return window.CaptureRect(r)
	case Monitor:
		return monitor.CaptureRect(r)
	case Unknown:
		fallthrough
	default:
		return nil, fmt.Errorf("failed to capture video area: exhausted sources")
	}
}

func Close() {
	device.Close()
	window.Close()
	monitor.Close()
}

func Devices() []int {
	return device.Sources()
}

func FPS() float64 {
	switch Active() {
	case Device:
		return device.FPS()
	case Window:
		return 0
	case Monitor:
		return 0
	case Unknown:
		fallthrough
	default:
		return -1
	}
}

func Is(s Source) bool {
	return Active() == s
}

func Monitors() []string {
	return monitor.Sources
}

func Name() string {
	switch Active() {
	case Device:
		return device.ActiveName()
	case Window:
		return window.ActiveName()
	case Monitor:
		return monitor.ActiveName()
	case Unknown:
		fallthrough
	default:
		return Unknown.String()
	}
}

func NameOf(s Source, index int) string {
	switch s {
	case Device:
		return device.Name(index)
	case Window:
		return window.Name(index)
	case Monitor:
		return monitor.Name(index)
	case Unknown:
		fallthrough
	default:
		return Unknown.String()
	}
}

func Open(sources ...Source) error {
	if len(sources) == 0 {
		sources = []Source{Device, Window, Monitor}
	}

	if slices.Contains(sources, Monitor) {
		defer monitor.Open()
	}

	if slices.Contains(sources, Device) {
		err := device.Open()
		if err != nil {
			notify.Error("[Video] <ini:f:open> video capture device (%v)", err)
		}
	}

	if slices.Contains(sources, Window) {
		err := window.Open()
		if err != nil {
			if err != window.ErrFailedFind {
				notify.Error("[Video] <ini:f:open> window capture library (%v)", err)
			}
		}
	}

	return nil
}

func Resolution() image.Point {
	switch Active() {
	case Device:
		return device.Resolution()
	case Window:
		return window.Resolution()
	case Monitor:
		return monitor.Resolution()
	default:
		return image.Pt(-1, -1)
	}
}

func (s Source) String() string {
	return string(s)
}

func StateArea() image.Rectangle {
	img, err := Capture()
	if err != nil {
		notify.Error("[Video] <ini:f:capture> area for state events (%v)", err)
		return image.Rect(0, 0, 0, 0)
	}

	if img == nil {
		img = splash.DeviceRGBA()

	}

	b := img.Bounds()
	r := image.Rect(b.Max.X/3, 0, b.Max.X-b.Max.X/3, b.Max.Y/2)

	return r
}
