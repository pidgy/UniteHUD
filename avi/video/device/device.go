package device

import (
	"fmt"
	"image"
	"sort"
	"strings"
	"sync"
	"time"

	"gocv.io/x/gocv"
	"golang.org/x/exp/slices"

	"github.com/pidgy/unitehud/avi/img"
	"github.com/pidgy/unitehud/avi/img/splash"
	"github.com/pidgy/unitehud/avi/video/device/win32"
	"github.com/pidgy/unitehud/avi/video/fps"
	"github.com/pidgy/unitehud/avi/video/monitor"
	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pkg/errors"
)

type properties struct {
	resolution image.Point

	fps,
	bitrate float64

	codec,
	backend string

	buffersize int
	rgb        bool
}

var (
	active = struct {
		index           int
		name            string
		closeq, closedq chan bool
		fps             float64
		applied         properties
	}{}

	required = properties{
		resolution: image.Pt(1920, 1080),
	}

	mat  = splash.DeviceMat().Clone()
	size = mat.Size()
	lock = &sync.RWMutex{}

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
	reset()
	go storeAPIs()
	go storeSources()
}

func Open() error {
	if config.Current.Video.Capture.Device.Index == config.NoVideoCaptureDevice {
		notify.Debug("Device: Disabled, ignorning call to open")
		return nil
	}

	if active.index != config.NoVideoCaptureDevice {
		notify.Debug("Device: %s, ignorning call to open active device", active.name)
		return nil
	}

	active.index = config.Current.Video.Capture.Device.Index
	active.name = Name(config.Current.Video.Capture.Device.Index)
	active.closeq = make(chan bool)
	active.closedq = make(chan bool)

	notify.System("Device: %s, opening...", active.name)

	err := capture()
	if err != nil {
		reset()
		return err
	}

	return nil
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
	return CaptureRect(image.Rectangle{Max: required.resolution})
}

func CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
	lock.RLock()
	defer lock.RUnlock()

	if mat.Empty() {
		return nil, nil
	}

	if !rect.In(monitor.MainResolution) {
		return nil, fmt.Errorf("illegal boundaries: %s outside %s", rect, monitor.MainResolution)
	}

	mrect := image.Rect(0, 0, size[1], size[0])

	if !rect.In(mrect) {
		return splash.AsRGBA(splash.Invalid()), fmt.Errorf("illegal boundaries: %s outside %s", rect, mrect)
	}

	return img.RGBA(mat.Region(rect))
}

func Close() {
	if active.index == config.NoVideoCaptureDevice {
		notify.Debug("Device: Disabled, ignoring close")
		return
	}

	notify.Debug("Device: %s, closing", active.name)
	defer notify.Debug("Device: %s, closed", active.name)

	stop()

	reset()
}

func FPS() (current, quota float64) {
	return active.fps, active.fps / active.applied.fps
}

func IsActive() bool {
	return active.index != config.NoVideoCaptureDevice
}

func Name(d int) string {
	if d == config.NoVideoCaptureDevice {
		return "Disabled"
	}
	if d != config.NoVideoCaptureDevice && len(cached.devices.names) > d {
		return cached.devices.names[d]
	}
	return fmt.Sprintf("%d", d)
}

func Restart() error {
	idx := config.Current.Video.Capture.Device.Index
	Close()
	config.Current.Video.Capture.Device.Index = idx
	return Open()
}

func Sources() []int {
	return cached.devices.ids
}

func capture() error {
	api := API(config.Current.Video.Capture.Device.API)

	notify.Debug("Device: %s, capturing with %s API", active.name, APIName(api))

	device, err := gocv.OpenVideoCaptureWithAPI(active.index, gocv.VideoCaptureAPI(api))
	if err != nil {
		return errors.Wrap(errors.Errorf("this device does not support %s format", APIName(api)), active.name)
	}

	err = set(device)
	if err != nil {
		device.Close()
		return errors.Wrap(err, active.name)
	}

	config.Current.Video.Capture.Device.API = active.applied.backend

	go func() {
		defer close(active.closedq)

		ms := fps.Milliseconds(config.Current.Video.Capture.Device.FPS)
		tick := time.NewTicker(ms)
		poll := time.NewTicker(time.Second)

		for frames := float64(0); running(); frames++ {
			lock.Lock()
			ok := device.Read(&mat)
			if !ok {
				defer reset()
				notify.Error("Device: %s, failed to capture", active.name)
				lock.Unlock()
				goto close
			}
			lock.Unlock()

			size = mat.Size()

			select {
			case <-tick.C:
				tick.Reset(ms)
			case <-poll.C:
				poll.Reset(time.Second)
				active.fps = frames
				frames = 0
			}
		}

	close:
		err := device.Close()
		if err != nil {
			notify.Warn("Device: %s, failed to close (%v)", active.name, err)
		}
	}()

	return nil
}

func poll(device *gocv.VideoCapture) properties {
	device.Get(gocv.VideoCaptureGUID)
	defaults := properties{
		resolution: image.Pt(
			int(device.Get(gocv.VideoCaptureFrameWidth)),
			int(device.Get(gocv.VideoCaptureFrameHeight)),
		),
		fps:        device.Get(gocv.VideoCaptureFPS),
		codec:      device.CodecString(),
		backend:    APIName(int(device.Get(gocv.VideoCaptureBackend))),
		bitrate:    device.Get(gocv.VideoCaptureBitrate),
		buffersize: int(device.Get(gocv.VideoCaptureBufferSize)),
		rgb:        bool(int(device.Get(gocv.VideoCaptureConvertRGB)) == 1),
	}
	return defaults
}

func reset() {
	notify.Debug("Device: %s, resetting", active.name)

	lock.Lock()
	defer lock.Unlock()

	mat = splash.DeviceMat().Clone()
	size = mat.Size()

	config.Current.Video.Capture.Window.Name = config.MainDisplay
	config.Current.Video.Capture.Device.Index = config.NoVideoCaptureDevice

	active.index = config.NoVideoCaptureDevice
	active.name = "Disabled"
	active.fps = -1
	active.closeq = make(chan bool)
	active.closedq = make(chan bool)
}

func running() bool {
	select {
	case <-active.closeq:
		return false
	default:
		return true
	}
}

func set(device *gocv.VideoCapture) error {
	p := poll(device)

	required.fps = float64(config.Current.Video.Capture.Device.FPS)

	device.Set(gocv.VideoCaptureFrameWidth, float64(required.resolution.X))
	device.Set(gocv.VideoCaptureFrameHeight, float64(required.resolution.Y))
	device.Set(gocv.VideoCaptureFPS, float64(config.Current.Video.Capture.Device.FPS))
	device.Set(gocv.VideoCaptureConvertRGB, 1)
	device.Set(gocv.VideoCaptureFOURCC, device.ToCodec("MJPG"))
	active.applied = poll(device)

	if !active.applied.resolution.Eq(required.resolution) {
		return errors.Wrapf(fmt.Errorf("%s resolution", required.resolution), "failed to set property")
	}

	if int(active.applied.fps) != int(required.fps) {
		return errors.Wrapf(fmt.Errorf("%.0f FPS", required.fps), "failed to set property")
	}

	notify.System("Device: %s, configured", active.name)
	notify.System("Device:   Codec       %s → %s", p.codec, active.applied.codec)
	notify.System("Device:   FPS         %.0f FPS → %.0f FPS", p.fps, active.applied.fps)
	notify.System("Device:   Resolution  %s → %s", p.resolution, active.applied.resolution)
	notify.System("Device:   Backend     %s → %s", p.backend, active.applied.backend)
	notify.System("Device:   Bitrate     %.0f kb/s", active.applied.bitrate)
	notify.System("Device:   BufferSize  %d", active.applied.buffersize)
	notify.System("Device:   RGB         %t → %t", p.rgb, active.applied.rgb)

	return nil
}

func stop() {
	for t := time.NewTimer(time.Second * 5); ; {
		select {
		case active.closeq <- true:
		case <-active.closedq:
			if !t.Stop() {
				<-t.C
			}
			return
		case <-t.C:
			notify.Error("Device: %s, failed to stop", active.name)
			return
		}
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
			d, err := win32.NewVideoCaptureDevice(i)
			if err != nil {
				notify.Warn("Device: %d, Failed to read properties of device", i)
				break
			}
			if d.Name == "" {
				break
			}

			ids = append(ids, i)
			names = append(names, d.Name)
		}

		for _, name := range names {
			if !slices.Contains(cached.devices.names, name) {
				cached.devices.ids, cached.devices.names = ids, names
				break
			}
		}
	}
}
