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

func Reattach() error {
	if screen.IsDisplay() {
		return nil
	}

	if device.IsActive() {
		return nil
	}

	return window.Reattach()

}

func Resize16x9() error {
	if screen.IsDisplay() {
		return nil
	}

	if device.IsActive() {
		return nil
	}

	return window.Resize16x9()
}

func Sources() (windows []string, devices []int, screens []string) {
	return window.Sources, device.Sources, screen.Sources
}
