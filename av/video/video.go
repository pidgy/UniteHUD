package video

import (
	"image"

	"github.com/pidgy/unitehud/av/img/splash"
	"github.com/pidgy/unitehud/av/video/device"
	"github.com/pidgy/unitehud/av/video/monitor"
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
		return device.IsActive() && (name == device.ActiveName() || name == "")
	case Monitor:
		return !device.IsActive() && (monitor.Active(name) || name == "")
	// case Window:
	// 	return !device.IsActive() && window.IsOpen()
	default:
		return false
	}
}

func Capture() (img *image.RGBA, err error) {
	if device.IsActive() {
		return device.Capture()
	}

	return monitor.Capture()

	// if monitor.IsDisplay() {
	// 	return monitor.Capture()
	// }

	// img, err = window.Capture()
	// if err != nil {
	// 	return monitor.Capture()
	// }

	// return
}

func CaptureRect(rect image.Rectangle) (img *image.RGBA, err error) {
	if device.IsActive() {
		return device.CaptureRect(rect)
	}

	return monitor.CaptureRect(rect)

	// if monitor.IsDisplay() {
	// 	return monitor.CaptureRect(rect)
	// }

	// return window.Capture()
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

// func Windows() []string {
// 	return window.Sources
// }

func Screens() []string {
	return monitor.Sources
}

func StateArea() image.Rectangle {
	img, err := Capture()
	if err != nil {
		notify.Error("[Video] <ini:failed:capture> area for state events (%v)", err)
		return image.Rect(0, 0, 0, 0)
	}

	if img == nil {
		img = splash.DeviceRGBA()

	}

	b := img.Bounds()
	r := image.Rect(b.Max.X/3, 0, b.Max.X-b.Max.X/3, b.Max.Y/2)

	return r
}
