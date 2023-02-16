package device

import (
	"fmt"
	"image"
	"time"

	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/video/device/win32"
)

var (
	Hide           = true
	Sources, names = sources()

	HD1080 = image.Rect(0, 0, 1920, 1080)

	active = config.NoVideoCaptureDevice
	base   = gocv.IMRead(fmt.Sprintf(`%s/splash/paused2.png`, config.Current.Assets()), gocv.IMReadColor) // Global matrix is more efficient?
	mat    = base.Clone()

	running = false
	stopped = true
)

func init() {
	go func() {
		for range time.NewTicker(time.Second * 5).C {
			// Sources, names = sources()
		}
	}()
}

func Capture() (*image.RGBA, error) {
	return CaptureRect(HD1080)
}

func CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
	if mat.Empty() {
		return nil, nil
	}

	if !rect.In(HD1080) {
		return nil, fmt.Errorf("Requested capture area is outside of the legal capture area %s > %s", rect, HD1080)
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

func Load() error {
	if active == config.Current.VideoCaptureDevice {
		return nil
	}

	err := openCaptureDevice()
	if err != nil {
		notify.Error("Failed to open Video Capture Device (%v)", err)
		reset()
		return err
	}

	return nil
}

func Name(d int) string {
	if len(names) > d {
		return names[d]
	}
	return fmt.Sprintf("Video Capture Device %d", d)
}

func convert(mat gocv.Mat) (*image.RGBA, error) {
	i, err := mat.ToImage()
	if err != nil {
		notify.Error("Failed to convert image for %s (%v)", Name(config.Current.VideoCaptureDevice), err)
		return nil, err
	}

	img, ok := i.(*image.RGBA)
	if !ok {
		notify.Error("Failed to colorize image for %s (%v)", Name(config.Current.VideoCaptureDevice), err)
		return nil, err
	}

	return img, nil
}

func closeCaptureDevice() {
	log.Debug().Int("device", config.Current.VideoCaptureDevice).Msg("closing capture device")

	running = false
	for !stopped {
		time.Sleep(time.Nanosecond)
	}

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

func sources() ([]int, []string) {
	s := []int{}
	n := []string{}

	for i := 0; i < 10; i++ {
		name := win32.VideoCaptureDeviceName(i)
		if name == "" {
			break
		}

		s = append(s, i)
		n = append(n, name)
	}

	return s, n
}

func startCaptureDevice(errq chan error) {
	running = true

	stopped = false
	defer func() {
		stopped = true
		resetActive()
	}()

	name := Name(config.Current.VideoCaptureDevice)

	notify.System("Capturing from %s", name)

	log.Debug().Str("name", name).Int("device", config.Current.VideoCaptureDevice).Msg("starting capture")

	device, err := gocv.OpenVideoCaptureWithAPI(config.Current.VideoCaptureDevice, gocv.VideoCaptureDshow)
	if err != nil {
		errq <- err
		return
	}
	defer device.Close()

	device.Set(gocv.VideoCaptureFrameWidth, float64(HD1080.Dx()))
	device.Set(gocv.VideoCaptureFrameHeight, float64(HD1080.Dy()))

	area := image.Rect(0, 0, int(device.Get(gocv.VideoCaptureFrameWidth)), int(device.Get(gocv.VideoCaptureFrameHeight)))
	if !area.Eq(HD1080) {
		mat = base.Clone()
		errq <- fmt.Errorf("%s has invalid dimensions: %s", name, area.String())
		return
	}

	close(errq)

	for running && active == config.Current.VideoCaptureDevice {
		if !device.Read(&mat) {
			notify.Warn("Failed to read from %s", name)
		}

		if mat.Empty() {
			notify.Warn("Failed to read from %s", name)
			continue
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
