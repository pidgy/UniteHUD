package video

import (
	"image"

	"github.com/pidgy/unitehud/avi/img/splash"
	"github.com/pidgy/unitehud/avi/video/device"
	"github.com/pidgy/unitehud/avi/video/monitor"
	"github.com/pidgy/unitehud/avi/video/window"
	"github.com/pidgy/unitehud/core/notify"
)

type Input string

const (
	Monitor Input = "monitor"
	Device  Input = "video-capture-device"
	Window  Input = "window"
)

func Active(i Input, name string) bool {
	switch i {
	case Device:
		return device.IsActive() && device.ActiveName() == name
	case Monitor:
		return !device.IsActive() && monitor.Active(name)
	default:
		return !device.IsActive() && window.IsOpen()
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

	return
}

func CaptureRect(rect image.Rectangle) (img *image.RGBA, err error) {
	if device.IsActive() {
		return device.CaptureRect(rect)
	}

	if monitor.IsDisplay() {
		return monitor.CaptureRect(rect)
	}

	return window.Capture()
}

func Close() {
	device.Close()
}

func Devices() []int {
	return device.Sources()
}

func Open() error {
	monitor.Open()
	return device.Open()
}

func Windows() []string {
	return window.Sources
}

func Screens() []string {
	return monitor.Sources
}

func StateArea() image.Rectangle {
	img, err := Capture()
	if err != nil {
		notify.Error("[Video] Failed to capture area for state events (%v)", err)
		return image.Rect(0, 0, 0, 0)
	}
	if img == nil {
		img = splash.DeviceRGBA()
	}

	b := img.Bounds()
	return image.Rect(b.Max.X/3, 0, b.Max.X-b.Max.X/3, b.Max.Y/2)
}
