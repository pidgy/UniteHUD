package video

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

func (s Source) String() string {
	return string(s)
}

func Active(s Source, name string) bool {
	switch s {
	case Device:
		return device.IsActive() && (name == device.ActiveName() || name == "")
	case Monitor:
		return !device.IsActive() && (monitor.Active(name) || name == "")
	case Window:
		return !device.IsActive() && window.IsOpen()
	default:
		return false
	}
}

func Current() Source {
	switch {
	case Active(Device, ""):
		return Device
	case Active(Monitor, ""):
		return Monitor
	case Active(Window, ""):
		return Window
	default:
		return Unknown
	}
}

func Capture() (img *image.RGBA, err error) {
	if device.IsActive() {
		return device.Capture()
	}

	if monitor.IsDisplay() {
		return monitor.Capture()
	}

	img, err = window.Capture()
	if err != nil {
		return monitor.Capture()
	}

	return nil, fmt.Errorf("failed to capture video: exhausted attempts")
}

func CaptureRect(rect image.Rectangle) (img *image.RGBA, err error) {
	if device.IsActive() {
		return device.CaptureRect(rect)
	}

	if monitor.IsDisplay() {
		return monitor.Capture()
	}

	return window.Capture()
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
	case window.IsOpen():
		return -1
	default:
		return -1
	}
}

func Name() string {
	switch {
	case device.IsActive():
		return device.ActiveName()
	default:
		return config.Current.Video.Capture.Window.Name
	}
}

func Open() error {
	monitor.Open()

	err := device.Open()
	if err != nil {
		notify.Error("[Video] <ini:f:open> video capture device library")
	}

	err = window.Open()
	if err != nil {
		notify.Error("[Video] <ini:f:open> window capture library")
	}

	return err
}

func Resolution() string {
	switch {
	case device.IsActive():
		return device.Resolution()
	case window.IsOpen():
		return window.Resolution()
	default:
		return monitor.Resolution()
	}
}

func Screens() []string {
	return monitor.Sources
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

func Windows() []string {
	return window.Sources
}
