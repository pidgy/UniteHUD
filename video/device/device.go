package device

import (
	"fmt"
	"image"
	"runtime"
	"time"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/img"
	"github.com/pidgy/unitehud/img/splash"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/video/device/win32"
	"github.com/pidgy/unitehud/video/monitor"
)

var (
	Sources, names = sources()

	active = config.NoVideoCaptureDevice
	mat    = splash.DeviceMat().Clone()

	running = false
	stopped = true
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
					notify.Debug("Video Capture Device: Discovered \"%s\"", got)
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
	return fmt.Sprintf("Video Capture Device: \"%d\"", active)
}

func Capture() (*image.RGBA, error) {
	return CaptureRect(monitor.MainResolution)
}

func CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
	if mat.Empty() {
		return nil, nil
	}

	if !rect.In(monitor.MainResolution) {
		return nil, fmt.Errorf("illegal boundaries %s intersects %s", rect, monitor.MainResolution)
	}

	s := mat.Size()
	mrect := image.Rect(0, 0, s[1], s[0])

	if !rect.In(mrect) {
		return nil, fmt.Errorf("illegal boundaries %s, %s", rect, mrect)
	}

	return img.RGBA(mat.Region(rect))
}

func Close() {
	if !running {
		notify.Debug("Video Capture Device: Ignorning call to close \"%s\"", ActiveName())
		return
	}

	running = false
	for !stopped {
		time.Sleep(time.Microsecond)
	}

	config.Current.VideoCaptureDevice = config.NoVideoCaptureDevice
}

func IsActive() bool {
	return !deviceChanged() && !deviceNotActive()
}

func Name(d int) string {
	if d != config.NoVideoCaptureDevice && len(names) > d {
		return names[d]
	}
	return fmt.Sprintf("Video Capture Device: %d", d)
}

func Open() error {
	if running || deviceNotActive() {
		notify.Debug("Video Capture Device: Ignorning call to open \"%s\"", ActiveName())
		return nil
	}

	active = config.Current.VideoCaptureDevice

	err := startCaptureDevice()
	if err != nil {
		reset()
		return err
	}

	return nil
}

func deviceChanged() bool {
	return active != config.Current.VideoCaptureDevice
}

func deviceNotActive() bool {
	return config.Current.VideoCaptureDevice == config.NoVideoCaptureDevice
}

func reset() {
	config.Current.VideoCaptureWindow = config.MainDisplay
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
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		running = true

		stopped = false
		defer func() {
			stopped = true
			resetActive()
		}()

		name := Name(config.Current.VideoCaptureDevice)

		notify.System("Video Capture Device: Starting \"%s\"", name)
		defer notify.System("Video Capture Device: Closing \"%s\"", name)

		api := gocv.VideoCaptureDshow
		if config.Current.VideoCaptureGenericAPI {
			api = gocv.VideoCaptureAny
		}

		device, err := gocv.OpenVideoCaptureWithAPI(config.Current.VideoCaptureDevice, api)
		if err != nil {
			errq <- err
			return
		}
		defer device.Close()

		notify.System("Video Capture Device: Applying dimensions (%s)", monitor.MainResolution)

		device.Set(gocv.VideoCaptureFrameWidth, float64(monitor.MainResolution.Dx()))
		device.Set(gocv.VideoCaptureFrameHeight, float64(monitor.MainResolution.Dy()))
		capture := image.Rect(0, 0,
			int(device.Get(gocv.VideoCaptureFrameWidth)),
			int(device.Get(gocv.VideoCaptureFrameHeight)),
		)
		if !capture.Eq(monitor.MainResolution) {
			errq <- fmt.Errorf("illegal dimensions %s", monitor.MainResolution)
			return
		}

		area := image.Rect(0, 0, int(device.Get(gocv.VideoCaptureFrameWidth)), int(device.Get(gocv.VideoCaptureFrameHeight)))
		if !area.Eq(monitor.MainResolution) {
			mat = splash.DeviceMat().Clone()
			errq <- fmt.Errorf("invalid dimensions \"%s\": %s", name, area.String())
			return
		}

		close(errq)

		defer Close()

		for ; running; time.Sleep(time.Millisecond) {

			if deviceNotActive() {
				return
			}

			if !device.Read(&mat) || mat.Empty() {
				notify.Warn("Video Capture Device: Failed to capture \"%s\"", name)
				return
			}
		}

		// defer fps.NewLoop(&fps.LoopOptions{
		// 	FPS: 20,
		// 	Render: func(min, max, avg time.Duration) (close bool) {
		// 		if !running || deviceChanged() {
		// 			return true
		// 		}

		// 		if deviceNotActive() {
		// 			go Close()
		// 			return true
		// 		}

		// 		if !device.Read(&mat) || mat.Empty() {
		// 			notify.Warn("Video Capture Device: Failed to capture \"%s\"", name)
		// 			return true
		// 		}

		// 		return false
		// 	},
		// }).Stop()
	}()

	return <-errq
}
