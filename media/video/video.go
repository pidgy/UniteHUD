package video

import (
	"image"

	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/media/video/device"
	"github.com/pidgy/unitehud/media/video/monitor"
	"github.com/pidgy/unitehud/media/video/window"
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

func Open() error {
	monitor.Open()

	err := device.Open()
	if err != nil {
		return err
	}

	return nil
}

func Windows() []string {
	return window.Sources
}

func Devices() []int {
	return device.Sources()
}

func Screens() []string {
	return monitor.Sources
}

func StateArea() image.Rectangle {
	i, err := Capture()
	if err != nil {
		notify.Error("📽️  Failed to capture area for state events (%v)", err)
		return image.Rect(0, 0, 0, 0)
	}

	b := i.Bounds()
	return image.Rect(b.Max.X/3, 0, b.Max.X-b.Max.X/3, b.Max.Y)
	//return i.Bounds()
}
