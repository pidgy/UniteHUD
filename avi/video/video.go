package video

import (
	"image"

	"github.com/pidgy/unitehud/avi/img/splash"
	"github.com/pidgy/unitehud/avi/video/device"
	"github.com/pidgy/unitehud/avi/video/monitor"
	"github.com/pidgy/unitehud/avi/video/window"
	"github.com/pidgy/unitehud/core/notify"
)

func Capture() (img *image.RGBA, err error) {
	if device.IsActive() {
		return device.Capture()
	}

	if monitor.IsDisplay() {
		return monitor.Capture()
	}

	return window.Capture()
}

func CaptureRect(rect image.Rectangle) (img *image.RGBA, err error) {
	if device.IsActive() {
		return device.CaptureRect(rect)
	}

	if monitor.IsDisplay() {
		return monitor.CaptureRect(rect)
	}

	return window.CaptureRect(rect)
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
		notify.Error("Video: Failed to capture area for state events (%v)", err)
		return image.Rect(0, 0, 0, 0)
	}
	if img == nil {
		img = splash.DeviceRGBA()
	}

	b := img.Bounds()
	return image.Rect(b.Max.X/3, 0, b.Max.X-b.Max.X/3, b.Max.Y)
}
