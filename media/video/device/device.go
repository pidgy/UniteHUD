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
	"github.com/pidgy/unitehud/media/video/fps"
	"github.com/pidgy/unitehud/media/video/monitor"
	"github.com/pkg/errors"
)

type properties struct {
	resolution  image.Point
	fps         float64
	buffersize  int
	codec, guid string
	backend     string
	bitrate     float64
	rgb         bool
}

var (
	active = struct {
		id              int
		name            string
		closeq, closedq chan bool
		fps             float64
		applied         properties
	}{}

	required = properties{
		resolution: image.Pt(1920, 1080),
		fps:        60,
	}

	mat  = splash.DeviceMat().Clone()
	size = mat.Size()

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

	errProperty = errors.New("failed to set property")
)

func init() {
	reset()
	go storeAPIs()
	go storeSources()
}

func Open() error {
	if config.Current.Video.Capture.Device.Index == config.NoVideoCaptureDevice {
		return nil
	}

	if active.id != config.NoVideoCaptureDevice {
		notify.Debug("Device: Ignorning call to open \"%s\" (active)", ActiveName())
		return nil
	}

	active.id = config.Current.Video.Capture.Device.Index
	active.name = Name(config.Current.Video.Capture.Device.Index)
	active.closeq = make(chan bool)
	active.closedq = make(chan bool)

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
	if active.id == config.NoVideoCaptureDevice {
		notify.Debug("Device: Ignoring close")
		return
	}

	notify.Debug("Device: Closing %s", active.name)
	defer notify.Debug("Device: Closed %s", active.name)

	stop()

	reset()
}

func FPS() (current, quota float64) {
	return active.fps, active.fps / active.applied.fps
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
	return fmt.Sprintf("%d", d)
}

func Sources() []int {
	return cached.devices.ids
}

func capture() error {
	api := API(config.Current.Video.Capture.Device.API)

	notify.System("Device: Capturing from %s with %s API", active.name, APIName(api))

	device, err := gocv.OpenVideoCaptureWithAPI(active.id, gocv.VideoCaptureAPI(api))
	if err != nil {
		return errors.Wrap(errors.Errorf("this device does not support %s format", APIName(api)), active.name)
	}

	err = set(required, device)
	if err != nil {
		device.Close()
		return errors.Wrap(err, active.name)
	}

	go func() {
		defer close(active.closedq)

		frames := float64(0)
		tick := time.NewTicker(fps.Sixty)
		rate := time.Duration(0)

		for running() {
			if !device.Read(&mat) || mat.Empty() {
				defer reset()
				notify.Error("Device: %s failed to capture", active.name)
				break
			}

			size = mat.Size()

			<-tick.C
			tick.Reset(fps.Sixty)

			rate += fps.Sixty

			frames++
			if rate >= time.Second {
				active.fps = frames
				frames = 0
				rate = 0
			}
		}

		notify.System("Device: %s closing...", active.name)

		err := device.Close()
		if err != nil {
			notify.Warn("Device: %s failed to close (%v)", active.name, err)
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

func running() bool {
	select {
	case <-active.closeq:
		return false
	default:
		return true
	}
}

func set(requested properties, device *gocv.VideoCapture) error {
	p := poll(device)

	device.Set(gocv.VideoCaptureFrameWidth, float64(requested.resolution.X))
	device.Set(gocv.VideoCaptureFrameHeight, float64(requested.resolution.Y))
	device.Set(gocv.VideoCaptureFPS, float64(requested.fps))
	device.Set(gocv.VideoCaptureConvertRGB, 1)
	device.Set(gocv.VideoCaptureFOURCC, device.ToCodec("MJPG"))
	active.applied = poll(device)

	if !active.applied.resolution.Eq(requested.resolution) {
		return errors.Wrapf(fmt.Errorf("%s resolution", requested.resolution), "failed to set property")
	}

	if active.applied.fps != requested.fps {
		return errors.Wrapf(fmt.Errorf("%.0f FPS", requested.fps), "failed to set property")
	}

	p.diff()

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
			notify.Error("Device: %s failed to stop", active.name)
			return
		}
	}
}

func (p properties) diff() {
	notify.System("Device: %s", active.name)
	notify.System("Device:  Codec       %s -> %s", p.codec, active.applied.codec)
	notify.System("Device:  FPS         %.0f FPS -> %.0f FPS", p.fps, active.applied.fps)
	notify.System("Device:  Resolution  %s -> %s", p.resolution, active.applied.resolution)
	notify.System("Device:  Backend     %s -> %s", p.backend, active.applied.backend)
	notify.System("Device:  Bitrate     %.0f kb/s", active.applied.bitrate)
	notify.System("Device:  BufferSize  %d", active.applied.buffersize)
	notify.System("Device:  RGB         %t -> %t", p.rgb, active.applied.rgb)
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
				notify.Warn("Device: %d, Failed to read properties of device", i)
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
	notify.Debug("Device: Resetting %s", active.name)

	mat = splash.DeviceMat().Clone()

	config.Current.Video.Capture.Window.Name = config.MainDisplay
	config.Current.Video.Capture.Device.Index = config.NoVideoCaptureDevice

	active.id = config.NoVideoCaptureDevice
	active.name = "Disabled"
	active.fps = -1
	active.closeq = make(chan bool)
	active.closedq = make(chan bool)
}
