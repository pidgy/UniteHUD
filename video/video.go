package video

import (
	"image"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/video/device"
	"github.com/pidgy/unitehud/video/screen"
	"github.com/pidgy/unitehud/video/window"
)

var HD1080 = device.HD1080

func Capture() (img *image.RGBA, err error) {
	if config.Current.VideoCaptureDevice != config.NoVideoCaptureDevice {
		return device.Capture()
	}

	if config.Current.Window == config.MainDisplay {
		return screen.Capture()
	}

	return window.Capture()
}

func CaptureRect(rect image.Rectangle) (img *image.RGBA, err error) {
	if config.Current.VideoCaptureDevice != config.NoVideoCaptureDevice {
		return device.CaptureRect(rect)
	}

	if config.Current.Window == config.MainDisplay {
		return screen.CaptureRect(rect)
	}

	return window.CaptureRect(rect)
}

func Close() {
	device.Close()
}

func Load() error {
	device.Load()
	return window.Load()
}

func Reattach() error {
	if config.Current.Window == config.MainDisplay {
		return nil
	}

	if config.Current.VideoCaptureDevice == config.NoVideoCaptureDevice {
		return nil
	}

	return window.Reattach()

}

func Resize16x9() error {
	if config.Current.Window == config.MainDisplay {
		return nil
	}

	if config.Current.VideoCaptureDevice == config.NoVideoCaptureDevice {
		return nil
	}

	return window.Resize16x9()
}

func Sources() (windows []string, devices []int) {
	return window.Sources, device.Sources
}
