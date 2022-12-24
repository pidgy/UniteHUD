package video

import (
	"image"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/video/device"
	"github.com/pidgy/unitehud/video/screen"
	"github.com/pidgy/unitehud/video/window"
)

func Capture() (img *image.RGBA, err error) {
	if config.Current.VideoCaptureDevice != config.NoVideoCaptureDevice {
		return device.Capture()
	}

	for _, s := range window.Sources {
		if config.Current.Window == s {
			return window.Capture()
		}
	}

	return screen.Capture()
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
	err := device.Load()
	if err != nil {
		return err
	}

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
