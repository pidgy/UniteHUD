package device

import (
	"fmt"
	"image"
	"sort"
	"strings"
	"time"

	"gocv.io/x/gocv"
	"golang.org/x/exp/slices"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/media/img"
	"github.com/pidgy/unitehud/media/img/splash"
	"github.com/pidgy/unitehud/media/video/device/win32"
	"github.com/pidgy/unitehud/media/video/monitor"
)

type device struct {
	id              int
	name            string
	closeq, closedq chan bool
	errq            chan error
}

var (
	active *device

	mat = splash.DeviceMat().Clone()

	cached = struct {
		devices struct {
			ids   []int
			names []string
		}
		apis struct {
			names  []string
			values map[string]int
		}
	}{}

	apis = struct {
		names  []string
		values map[string]int
	}{
		values: make(map[string]int),
	}
)

func init() {
	defer reset()
	go storeAPIs()
	go storeSources()
}

func ActiveName() string {
	return active.name
}

func API(api string) int {
	if api == "" {
		return apis.values[apis.names[0]]
	}
	return apis.values[api]
}

func APIName(api int) string {
	return strings.Title(strings.ReplaceAll(gocv.VideoCaptureAPI(api).String(), "video-capture-", ""))
}

func APIs() []string {
	return apis.names
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
	if active.id == config.NoVideoCaptureDevice {
		notify.Debug("🎥  Ignorning call to close \"%s\" (inactive)", ActiveName())
		return
	}

	active.stop()

	notify.Debug("🎥  Closed \"%s\"", active.name)

	reset()
}

func (d *device) stop() {
	for t := time.NewTimer(time.Second * 5); ; {
		select {
		case d.closeq <- true:
		case <-d.closedq:
			if !t.Stop() {
				<-t.C
			}
			return
		case <-t.C:
			notify.Error("🎥  %s failed to stop", d.name)
		}
	}
}

func IsActive() bool {
	return active.id != config.NoVideoCaptureDevice
}

func Name(d int) string {
	if d == config.NoVideoCaptureDevice {
		return "Disabled"
	}
	if d != config.NoVideoCaptureDevice && len(cached.devices.names) > d {
		return cached.devices.names[d]
	}
	return fmt.Sprintf("🎥  %d", d)
}

func Open() error {
	if config.Current.Video.Capture.Device.Index == config.NoVideoCaptureDevice {
		return nil
	}

	if active.id != config.NoVideoCaptureDevice {
		notify.Debug("🎥  Ignorning call to open \"%s\" (active)", ActiveName())
		return nil
	}

	active = &device{
		id:      config.Current.Video.Capture.Device.Index,
		name:    Name(config.Current.Video.Capture.Device.Index),
		closeq:  make(chan bool),
		closedq: make(chan bool),
		errq:    make(chan error),
	}

	go active.capture()

	err := <-active.errq
	if err != nil {
		reset()
		return err
	}

	return nil
}

func Sources() []int {
	return cached.devices.ids
}

func (d *device) capture() {
	defer close(d.closedq)

	api := APIName(API(config.Current.Video.Capture.Device.API))

	notify.System("🎥  Opening \"%s\" with API \"%s\"", d.name, api)
	defer notify.System("🎥  Closing \"%s\"...", d.name)

	device, err := gocv.OpenVideoCaptureWithAPI(config.Current.Video.Capture.Device.Index, gocv.VideoCaptureAPI(API(config.Current.Video.Capture.Device.API)))
	if err != nil {
		d.errq <- fmt.Errorf("%s does not support %s encoding", d.name, api)
		return
	}
	defer device.Close()

	notify.System("🎥  Applying %dx%d dimensions for %s", monitor.MainResolution.Max.X, monitor.MainResolution.Max.Y, d.name)

	device.Set(gocv.VideoCaptureFrameWidth, float64(monitor.MainResolution.Dx()))
	device.Set(gocv.VideoCaptureFrameHeight, float64(monitor.MainResolution.Dy()))
	capture := image.Rect(
		0,
		0,
		int(device.Get(gocv.VideoCaptureFrameWidth)),
		int(device.Get(gocv.VideoCaptureFrameHeight)),
	)
	if !capture.Eq(monitor.MainResolution) {
		d.errq <- fmt.Errorf("%s has illegal dimensions %s", d.name, monitor.MainResolution)
		return
	}

	area := image.Rect(0, 0, int(device.Get(gocv.VideoCaptureFrameWidth)), int(device.Get(gocv.VideoCaptureFrameHeight)))
	if !area.Eq(monitor.MainResolution) {
		mat = splash.DeviceMat().Clone()
		d.errq <- fmt.Errorf("%s has invalid dimensions %s", d.name, area.String())
		return
	}

	close(d.errq)

	for d.running() {
		time.Sleep(time.Millisecond)
		if !device.Read(&mat) || mat.Empty() {
			notify.Warn("🎥  Failed to capture from \"%s\"", d.name)
			return
		}
	}
}

func (d *device) running() bool {
	select {
	case <-d.closeq:
		return false
	default:
		return true
	}
}

func storeAPIs() {
	base := []string{APIName(0)}
	for i := gocv.VideoCaptureAPI(1); i < 5000; i++ {
		api := i.String()
		if api == "" {
			continue
		}
		api = APIName(int(i))

		apis.values[api] = int(i)
		apis.names = append(apis.names, api)
	}
	sort.Strings(apis.names)

	apis.names = append(base, apis.names...)
}

func storeSources() {
	for ; ; time.Sleep(time.Second * 5) {
		ids := []int{}
		names := []string{}

		for i := 0; i < 10; i++ {
			name, err := win32.VideoCaptureDeviceName(i)
			if err != nil {
				notify.Error("🎥  Failed to read properties of device at index %d", i)
				break
			}
			if name == "" {
				break
			}

			ids = append(ids, i)
			names = append(names, name)
		}

		for _, name := range names {
			if !slices.Contains(cached.devices.names, name) {
				cached.devices.ids, cached.devices.names = ids, names
				break
			}
		}
	}
}

func reset() {
	config.Current.Video.Capture.Window.Name = config.MainDisplay
	config.Current.Video.Capture.Device.Index = config.NoVideoCaptureDevice

	active = &device{
		id:      config.NoVideoCaptureDevice,
		name:    "Disabled",
		closeq:  make(chan bool),
		closedq: make(chan bool),
		errq:    make(chan error),
	}
}
