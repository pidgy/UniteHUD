package video

import (
	"image"

	"github.com/pidgy/unitehud/video/device"
	"github.com/pidgy/unitehud/video/screen"
	"github.com/pidgy/unitehud/video/window"
)

func Capture() (img *image.RGBA, err error) {
	if device.IsActive() {
		return device.Capture()
	}

	if screen.IsDisplay() {
		return screen.Capture()
	}

	return window.Capture()
}

func CaptureRect(rect image.Rectangle) (img *image.RGBA, err error) {
	if device.IsActive() {
		return device.CaptureRect(rect)
	}

	if screen.IsDisplay() {
		return screen.CaptureRect(rect)
	}

	return window.CaptureRect(rect)
}

func Close() {
	device.Close()
}

func Open() error {
	screen.Open()

	err := device.Open()
	if err != nil {
		return err
	}

	return window.Open()
}

func Windows() []string {
	return window.Sources
}

func Devices() []int {
	return device.Sources
}

func Screens() []string {
	return screen.Sources
}
