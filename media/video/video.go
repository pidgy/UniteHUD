package video

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const. Then remove this comment.

import (
	"fmt"
	"image"

	"github.com/pidgy/unitehud/core/config"
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

func Active(s Source, name string) bool {
	switch s {
	case Device:
		return device.IsActive()
	case Monitor:
		return monitor.IsActive()
	case Window:
		return window.IsActive()
	default:
		return false
	}
}

func Current() Source {
	switch {
	case device.IsActive():
		return Device
	case monitor.IsActive():
		return Monitor
	case window.IsActive():
		return Window
	default:
		return Unknown
	}
}

func Capture() (*image.RGBA, error) {
	if device.IsActive() {
		return device.Capture()
	}

	if window.IsActive() {
		return window.Capture()
	}

	if monitor.IsActive() {
		return monitor.Capture()
	}

	return nil, fmt.Errorf("failed to capture video: exhausted sources")
}

func CaptureRect(r image.Rectangle) (*image.RGBA, error) {
	if device.IsActive() {
		return device.CaptureRect(r)
	}

	if window.IsActive() {
		return window.CaptureRect(r)
	}

	if monitor.IsActive() {
		return monitor.CaptureRect(r)
	}

	return nil, fmt.Errorf("failed to capture video area: exhausted sources")
}

func Close() {
	device.Close()
}

func Devices() []int {
	return device.Sources()
}

func FPS() float64 {
	switch {
	case device.IsActive():
		return device.FPS()
	case window.IsActive():
		return -1
	default:
		return -1
	}
}

func Monitors() []string {
	return monitor.Sources
}

func Name() string {
	switch {
	case device.IsActive():
		return device.ActiveName()
	default:
		return config.Current.Video.Capture.Monitor.Name
	}
}

func Open() error {
	defer monitor.Open()

	err := device.Open()
	if err != nil {
		notify.Error("[Video] <ini:f:open> video capture device (%v)", err)
	}

	err = window.Open()
	if err != nil {
		if err != window.ErrFailedFind {
			notify.Error("[Video] <ini:f:open> window capture library (%v)", err)
		}
	}

	return err
}

func Resolution() image.Point {
	switch {
	case device.IsActive():
		return device.Resolution()
	case window.IsActive():
		return window.Resolution()
	default:
		return monitor.Resolution()
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
