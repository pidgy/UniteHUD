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
	"github.com/pkg/errors"
)

type device struct {
	id              int
	name            string
	closeq, closedq chan bool
	errq            chan error
	fps             int
}

type properties struct {
	resolution      image.Point
	fps, buffersize int
	codec, guid     string
	backend         string
	bitrate         float64
	rgb             bool
}

var (
	active *device

	required = properties{
		resolution: image.Pt(1920, 1080),
	}

	min = properties{
		resolution: required.resolution,
		fps:        30,
	}

	max = properties{
		resolution: required.resolution,
		fps:        60,
	}

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
	go storeAPIs()
	go storeSources()
	reset()
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

	s := mat.Size()
	mrect := image.Rect(0, 0, s[1], s[0])

	if !rect.In(mrect) {
		return splash.AsRGBA(splash.Invalid()), fmt.Errorf("illegal boundaries: %s outside %s", rect, mrect)
	}

	return img.RGBA(mat.Region(rect))
}

func Close() {
	if active.id == config.NoVideoCaptureDevice {
		notify.Debug("🎥 %s: Ignorning call to close (inactive)", ActiveName())
		return
	}

	active.stop()

	notify.Debug("🎥 %s: Closed", active.name)

	reset()
}

func FPS() int {
	return active.fps
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

func Open() error {
	if config.Current.Video.Capture.Device.Index == config.NoVideoCaptureDevice {
		return nil
	}

	if active.id != config.NoVideoCaptureDevice {
		notify.Debug("🎥 Ignorning call to open \"%s\" (active)", ActiveName())
		return nil
	}

	active = &device{
		id:      config.Current.Video.Capture.Device.Index,
		name:    Name(config.Current.Video.Capture.Device.Index),
		closeq:  make(chan bool),
		closedq: make(chan bool),
		errq:    make(chan error),
	}

	err := active.capture()
	if err != nil {
		reset()
		return err
	}

	return nil
}

func Sources() []int {
	return cached.devices.ids
}

func (d *device) capture() error {
	api := config.Current.Video.Capture.Device.API

	notify.System("🎥 %s: Opening with API %s", d.name, api)

	device, err := gocv.OpenVideoCaptureWithAPI(d.id, gocv.VideoCaptureAPI(API(api)))
	if err != nil {
		return errors.Wrap(errors.Errorf("this device does not support %s format", api), d.name)
	}

	defaults := d.properties(device)
	defer defaults.diff(d, device)

	err = d.set(max, device)
	if err != nil {
		notify.Warn("🎥 %s: Failed to apply maximum properties", d.name)

		err = d.set(min, device)
		if err != nil {
			return errors.Wrap(err, d.name)
		}
	}

	go func() {
		defer close(d.closedq)

		ps := time.Now()
		frames := 0
		for d.running() {
			if !device.Read(&mat) || mat.Empty() {
				defer reset()
				notify.Error("🎥 %s: Failed to capture", d.name)
				break
			}
			frames++

			since := time.Since(ps)
			if since >= time.Second {
				active.fps = frames
				ps = time.Now()
				frames = 0
			}
		}

		notify.System("🎥 %s: Closing...", d.name)

		err := device.Close()
		if err != nil {
			notify.Warn("🎥 %s: Failed to close (%v)", d.name, err)
		}
	}()

	return nil
}

func (d *device) properties(device *gocv.VideoCapture) properties {
	device.Get(gocv.VideoCaptureGUID)
	defaults := properties{
		resolution: image.Pt(
			int(device.Get(gocv.VideoCaptureFrameWidth)),
			int(device.Get(gocv.VideoCaptureFrameHeight)),
		),
		fps:        int(device.Get(gocv.VideoCaptureFPS)),
		codec:      device.CodecString(),
		backend:    APIName(int(device.Get(gocv.VideoCaptureBackend))),
		bitrate:    device.Get(gocv.VideoCaptureBitrate),
		buffersize: int(device.Get(gocv.VideoCaptureBufferSize)),
		rgb:        bool(int(device.Get(gocv.VideoCaptureConvertRGB)) == 1),
	}
	return defaults
}

func (d *device) running() bool {
	select {
	case <-d.closeq:
		return false
	default:
		return true
	}
}

func (d *device) set(requested properties, device *gocv.VideoCapture) error {
	device.Set(gocv.VideoCaptureFrameWidth, float64(requested.resolution.X))
	device.Set(gocv.VideoCaptureFrameHeight, float64(requested.resolution.Y))
	device.Set(gocv.VideoCaptureFPS, float64(requested.fps))
	device.Set(gocv.VideoCaptureConvertRGB, 1)
	device.Set(gocv.VideoCaptureBufferSize, 0)
	device.Set(gocv.VideoCaptureFOURCC, device.ToCodec("MJPG"))
	applied := d.properties(device)

	if !applied.resolution.Eq(requested.resolution) {
		return errors.Wrap(errors.Errorf("resolution %s, %s required", applied.resolution, requested.resolution), "Invalid property")
	}

	if applied.fps != requested.fps {
		return errors.Wrap(errors.Errorf("%d FPS, %d FPS required", applied.fps, requested.fps), "Invalid property")
	}

	return nil
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
			notify.Error("🎥 %s failed to stop", d.name)
		}
	}
}

func (p properties) diff(d *device, device *gocv.VideoCapture) {
	applied := d.properties(device)
	notify.System("🎥 %s: Codec       %s -> %s", d.name, p.codec, applied.codec)
	notify.System("🎥 %s: FPS         %d FPS -> %d FPS", d.name, p.fps, applied.fps)
	notify.System("🎥 %s: Resolution  %s -> %s", d.name, p.resolution, applied.resolution)
	notify.System("🎥 %s: Backend     %s -> %s", d.name, p.backend, applied.backend)
	notify.System("🎥 %s: Bitrate     %.0f kb/s", d.name, applied.bitrate)
	notify.System("🎥 %s: BufferSize  %d", d.name, applied.buffersize)
	notify.System("🎥 %s: RGB         %t -> %t", d.name, p.rgb, applied.rgb)
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
				notify.Warn("🎥 Device %d: Failed to read properties of device", i)
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
	notify.Debug("🎥 Resetting")

	mat = splash.DeviceMat().Clone()

	config.Current.Video.Capture.Window.Name = config.MainDisplay
	config.Current.Video.Capture.Device.Index = config.NoVideoCaptureDevice

	active = &device{
		id:      config.NoVideoCaptureDevice,
		name:    "Disabled",
		closeq:  make(chan bool),
		closedq: make(chan bool),
		errq:    make(chan error),
		fps:     -1,
	}
}
