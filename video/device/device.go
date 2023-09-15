package device

import (
	"fmt"
	"image"
	"time"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/img"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/splash"
	"github.com/pidgy/unitehud/video/device/win32"
	"github.com/pidgy/unitehud/video/monitor"
)

var (
	Sources, names = sources()

	active = config.NoVideoCaptureDevice
	mat    = splash.DeviceMat().Clone()

	running = false
	stopped = true

	lastRequest = time.Now()
)

func init() {
	go func() {
		for ; ; time.Sleep(time.Second * 5) {
			s, n := sources()
			for _, got := range n {
				found := false
				for _, have := range names {
					if have == got {
						found = true
						break
					}
				}
				if !found {
					Sources, names = s, n
					notify.Debug("New video capture device detected (%s)", got)
					break
				}
			}
		}
	}()
}

func ActiveName() string {
	if active == config.NoVideoCaptureDevice {
		return "Disabled"
	}
	if len(names) > active {
		return names[active]
	}
	return fmt.Sprintf("Video Capture Device: %d", active)
}

func Capture() (*image.RGBA, error) {
	return CaptureRect(monitor.MainResolution())
}

func CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
	lastRequest = time.Now()

	if mat.Empty() {
		return nil, nil
	}

	err := outside(rect, monitor.MainResolution())
	if err != nil {
		return nil, err
	}

	if !rect.In(monitor.MainResolution()) {
		return nil, fmt.Errorf("Illegal boundaries %s intersects %s", rect, monitor.MainResolution())
	}

	s := mat.Size()
	mrect := image.Rect(0, 0, s[0], s[1])

	err = outside(rect, mrect)
	if err != nil {
		return nil, err
	}

	i, err := img.RGBA(mat.Region(rect))
	if err != nil {
		return nil, err
	}

	return i, nil
}

func outside(r1, r2 image.Rectangle) error {
	if r1.In(r2) {
		return nil
	}

	return fmt.Errorf("Illegal boundaries %s", r1.Intersect(r2))
}

func Close() {
	if !running {
		notify.Debug("Ignorning call to close %s video capture device", ActiveName())
		return
	}

	running = false
	for !stopped {
		time.Sleep(time.Microsecond)
	}

	config.Current.VideoCaptureDevice = config.NoVideoCaptureDevice
}

func IsActive() bool {
	return config.Current.VideoCaptureDevice == active && active != config.NoVideoCaptureDevice
}

func Name(d int) string {
	if d != config.NoVideoCaptureDevice && len(names) > d {
		return names[d]
	}
	return fmt.Sprintf("Video Capture Device: %d", d)
}

func Open() error {
	if running || config.Current.VideoCaptureDevice == config.NoVideoCaptureDevice {
		notify.Debug("Ignorning call to open video capture device (%s)", ActiveName())
		return nil
	}

	active = config.Current.VideoCaptureDevice

	err := startCaptureDevice()
	if err != nil {
		notify.Error("Failed to open video capture device (%v)", err)
		reset()
		return err
	}

	return nil
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

func startCaptureDevice() error {
	errq := make(chan error)

	go func() {
		running = true

		stopped = false
		defer func() {
			stopped = true
			resetActive()
		}()

		name := Name(config.Current.VideoCaptureDevice)

		notify.System("Starting video capture (%s)", name)
		defer notify.System("Closing video capture (%s)", name)

		device, err := gocv.OpenVideoCaptureWithAPI(config.Current.VideoCaptureDevice, gocv.VideoCaptureDshow)
		if err != nil {
			errq <- err
			return
		}
		defer device.Close()

		notify.System("Applying video capture dimensions: %s", monitor.MainResolution())
		device.Set(gocv.VideoCaptureFrameWidth, float64(monitor.MainResolution().Dx()))
		device.Set(gocv.VideoCaptureFrameHeight, float64(monitor.MainResolution().Dy()))
		x, y := int(device.Get(gocv.VideoCaptureFrameWidth)), int(device.Get(gocv.VideoCaptureFrameHeight))

		if !image.Rect(0, 0, x, y).Eq(monitor.MainResolution()) {
			errq <- fmt.Errorf("Illegal dimensions %s", monitor.MainResolution())
			return
		}

		area := image.Rect(0, 0, int(device.Get(gocv.VideoCaptureFrameWidth)), int(device.Get(gocv.VideoCaptureFrameHeight)))
		if !area.Eq(monitor.MainResolution()) {
			mat = splash.DeviceMat().Clone()
			errq <- fmt.Errorf("%s has invalid dimensions: %s", name, area.String())
			return
		}

		close(errq)

		for running && active == config.Current.VideoCaptureDevice {
			time.Sleep(time.Microsecond)

			if config.Current.VideoCaptureDevice == config.NoVideoCaptureDevice {
				go Close()
			}

			if time.Since(lastRequest) > time.Second {
				continue
			}

			if !device.Read(&mat) || mat.Empty() {
				notify.Warn("Failed to read from %s", name)
				continue
			}
		}
	}()

	return <-errq
}
