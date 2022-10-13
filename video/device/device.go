package device

import (
	"fmt"
	"image"

	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/notify"
)

var (
	Hide    = true
	Sources = []int{0, 1, 2, 3, 4}

	HD1080 = image.Rect(0, 0, 1920, 1080)

	mat  = gocv.IMRead("img/splash/paused2.png", gocv.IMReadColor) // Global matrix is more efficient?
	area = HD1080

	active = config.NoVideoCaptureDevice

	closeDeviceq       = make(chan bool)
	closedcloseDeviceq = make(chan bool)
)

func Capture() (*image.RGBA, error) {
	return CaptureRect(HD1080)
}

func CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
	if mat.Empty() {
		return nil, nil
	}

	if !rect.In(area) {
		return nil, fmt.Errorf("rectangle is outside of the legal capture area %s > %s", rect, area)
	}

	i, err := convert(mat.Region(rect))
	if err != nil {
		return nil, err
	}

	return i, nil
}

func Close() {
	if active == config.NoVideoCaptureDevice {
		return
	}

	closeCaptureDevice()
}

func Load() {
	if active != config.Current.VideoCaptureDevice {
		err := openCaptureDevice()
		if err != nil {
			notify.Error("Failed to open Video Capture Device %d (%v)", config.Current.VideoCaptureDevice, err)
			reset()
			return
		}
	}
}

func convert(mat gocv.Mat) (*image.RGBA, error) {
	i, err := mat.ToImage()
	if err != nil {
		notify.Error("Failed to convert Video Capture Device %d's image (%v)", config.Current.VideoCaptureDevice, err)
		return nil, err
	}

	img, ok := i.(*image.RGBA)
	if !ok {
		notify.Error("Failed to colorize Video Capture Device %d's image (%v)", config.Current.VideoCaptureDevice, err)
		return nil, err
	}

	return img, nil
}

func closeCaptureDevice() {
	log.Debug().Int("device", config.Current.VideoCaptureDevice).Msg("closing capture device")
	closeDeviceq <- true
	<-closedcloseDeviceq
}

func closedCaptureDevice() {
	closedcloseDeviceq <- true
	log.Debug().Int("device", config.Current.VideoCaptureDevice).Msg("closed capture device")
}

func openCaptureDevice() error {
	if active != config.NoVideoCaptureDevice {
		return nil
	}
	active = config.Current.VideoCaptureDevice

	errq := make(chan error)
	go startCaptureDevice(errq)
	return <-errq
}

func reset() {
	config.Current.Window = config.MainDisplay
	config.Current.VideoCaptureDevice = config.NoVideoCaptureDevice
}

func resetActive() {
	active = config.NoVideoCaptureDevice
}

func startCaptureDevice(errq chan error) {
	defer closedCaptureDevice()
	defer resetActive()

	notify.System("Capturing from Video Capture Device %d", config.Current.VideoCaptureDevice)

	device, err := gocv.OpenVideoCaptureWithAPI(config.Current.VideoCaptureDevice, gocv.VideoCaptureDshow)
	if err != nil {
		errq <- err
		return
	}
	defer device.Close()

	device.Set(gocv.VideoCaptureFrameWidth, 1920)
	device.Set(gocv.VideoCaptureFrameHeight, 1080)

	errq <- nil

	for active == config.Current.VideoCaptureDevice {
		select {
		case <-closeDeviceq:
			return
		default:
			if !device.Read(&mat) {
				notify.Warn("Failed to read from Video Capture Device %d", config.Current.VideoCaptureDevice)
			}

			if mat.Empty() {
				notify.Warn("Failed to read from Video Capture Device %d", config.Current.VideoCaptureDevice)
				continue
			}

			area = image.Rect(0, 0, mat.Cols(), mat.Rows())
		}
	}
}

// deadcode ignore invertMatrix
func invertMatrix() gocv.Mat {
	mat2 := gocv.NewMatWithSizeFromScalar(gocv.NewScalar(255, 255, 255, 255), mat.Rows(), mat.Cols(), mat.Type())
	mat3 := gocv.NewMat()
	gocv.Subtract(mat2, mat, &mat3)
	return mat3
}
