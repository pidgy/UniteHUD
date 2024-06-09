package device

import (
	"fmt"
	"image"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/avi/device"
	"github.com/pidgy/unitehud/avi/img"
	"github.com/pidgy/unitehud/avi/img/splash"
	"github.com/pidgy/unitehud/avi/video/fps"
	"github.com/pidgy/unitehud/avi/video/monitor"
	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
)

type (
	api struct {
		gocv gocv.VideoCaptureAPI
		name string
	}

	cache struct {
		devices []string
		apis    []api
		codecs  []codec
	}

	codec struct {
		name string
	}

	dev struct {
		index           int
		name            string
		closeq, closedq chan bool
		fps             float64
		applied         properties
	}

	properties struct {
		resolution image.Point

		fps,
		bitrate float64

		codec   codec
		backend api

		buffersize int
		rgb        bool
	}
)

const (
	Disabled = "Disabled"
)

var (
	CodecAny  = codec{"Any"}
	CodecXRGB = codec{"XRGB"}
	CodecNV12 = codec{"NV12"}
	CodecYUY2 = codec{"YUY2"}
	CodecMJPG = codec{"MJPG"}
)

var (
	active = &dev{}

	required = properties{
		resolution: image.Pt(1920, 1080),
	}

	mat  = splash.DeviceMat().Clone()
	size = mat.Size()
	lock = &sync.RWMutex{}

	cached = cache{
		devices: make([]string, 100),
		apis:    make([]api, int(gocv.VideoCaptureXINE)+1), // Max API value: gocv.VideoCaptureXINE.
		codecs:  []codec{CodecAny, CodecXRGB, CodecNV12, CodecYUY2, CodecMJPG},
	}
)

func init() {
	active.reset()

	go func() {
		for i := gocv.VideoCaptureAny; i < gocv.VideoCaptureXINE; i++ {
			s := i.String()
			if s != "" {
				cached.apis[i] = api{
					gocv: i,
					name: strings.Title(strings.ReplaceAll(i.String(), "video-capture-", "")),
				}
			}
		}

		for ; ; time.Sleep(time.Second * 5) {
			d := []string{}
			for i := 0; ; i++ {
				n, err := device.VideoCaptureDeviceName(i)
				if err != nil {
					break
				}
				d = append(d, n)
			}
			cached.devices = d
		}
	}()
}

func API(name string) api {
	for i, n := range cached.apis {
		if name == n.name {
			return cached.apis[i]
		}
	}
	return cached.apis[0]
}

func APIs() (apis []string) {
	for _, api := range cached.apis {
		if api.name != "" {
			apis = append(apis, api.name)
		}
	}
	return apis
}

func ActiveName() string {
	return active.name
}

func Capture() (*image.RGBA, error) {
	return CaptureRect(image.Rectangle{Max: required.resolution})
}

func CaptureRect(r image.Rectangle) (*image.RGBA, error) {
	lock.RLock()
	defer lock.RUnlock()

	if mat.Empty() {
		return nil, nil
	}

	if !r.In(monitor.MainResolution) {
		return nil, errors.Errorf("illegal boundaries: %s outside %s", r, monitor.MainResolution)
	}

	mrect := image.Rect(0, 0, size[1], size[0])

	if !r.In(mrect) {
		return splash.AsRGBA(splash.Invalid()), errors.Errorf("illegal boundaries: %s outside %s", r, mrect)
	}

	return img.RGBA(mat.Region(r))
}

func Close() {
	if active.index == config.NoVideoCaptureDevice {
		notify.Debug("[Video] Device disabled, ignoring close")
		return
	}

	notify.Debug("[Video] Closing %s", active.name)
	defer notify.Debug("[Video] %s closed", active.name)

	stop()

	active.reset()
}

func Codec(name string) codec {
	for _, codec := range cached.codecs {
		if name == codec.String() {
			return codec
		}
	}
	return cached.codecs[0]
}

func Codecs() []codec {
	return cached.codecs
}

func FPS() float64 {
	return active.fps
}

func IsActive() bool {
	return active.index != config.NoVideoCaptureDevice
}

func Name(index int) string {
	if index == config.NoVideoCaptureDevice {
		return Disabled
	}

	if cached.devices[index] != "" {
		return cached.devices[index]
	}

	var err error

	cached.devices[index], err = device.VideoCaptureDeviceName(index)
	if err != nil {
		notify.Error("[Video] Failed to find device %d name (%v)", index, err)
		return fmt.Sprintf("%d", index)
	}

	return cached.devices[index]
}

func Open() error {
	if config.Current.Video.Capture.Device.Index == config.NoVideoCaptureDevice {
		notify.Debug("[Video] Disabled, ignorning call to open")
		return nil
	}

	if active.index != config.NoVideoCaptureDevice {
		notify.Debug("[Video] Open ignored, %s is already active", active.name)
		return nil
	}

	idx := index(config.Current.Video.Capture.Device.Name)
	if idx == config.NoVideoCaptureDevice {
		active.reset()
		return errors.Errorf("%s was not found", config.Current.Video.Capture.Device.Name)
	}
	if idx != config.Current.Video.Capture.Device.Index {
		config.Current.Video.Capture.Device.Index = idx
		notify.Warn("[Video] Invalid index for %s", config.Current.Video.Capture.Device.Name)
	}

	active.index = config.Current.Video.Capture.Device.Index
	active.name = Name(config.Current.Video.Capture.Device.Index)
	active.closeq = make(chan bool)
	active.closedq = make(chan bool)

	notify.System("[Video] Opening %s", active.name)

	err := capture()
	if err != nil {
		active.reset()
		return err
	}

	return nil
}

func Restart() error {
	prev := config.Current.Video.Capture.Device
	Close()
	config.Current.Video.Capture.Device = prev
	return Open()
}

func Sources() (indexes []int) {
	for i, name := range cached.devices {
		if name != "" {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func (a api) String() string {
	return a.name
}

func (a api) Value() int {
	return int(a.gocv)
}

func (c *cache) api(f float64) api {
	i := int(f)
	if len(cached.apis) > i {
		return cached.apis[i]
	}
	return cached.apis[0]
}

func capture() error {
	if config.Current.Video.Capture.Device.API == "" {
		config.Current.Video.Capture.Device.API = config.DefaultVideoCaptureAPI
	}

	api := API(config.Current.Video.Capture.Device.API)

	notify.Debug("[Video] Capturing %s with %s API", active.name, config.Current.Video.Capture.Device.API)

	device, err := gocv.OpenVideoCaptureWithAPI(active.index, api.gocv)
	if err != nil {
		return errors.Errorf("this device does not support %s", api.name)
	}

	err = set(device)
	if err != nil {
		active.reset()
		return err
	}

	config.Current.Video.Capture.Device.API = active.applied.backend.name

	go func() {
		defer close(active.closedq)

		ms := fps.Milliseconds(config.Current.Video.Capture.Device.FPS)
		tick := time.NewTicker(ms)
		poll := time.NewTicker(time.Second)

		for frames := float64(0); running(); frames++ {
			lock.Lock()
			ok := device.Read(&mat)
			if !ok {
				defer active.reset()
				notify.Error("[Video] Failed to capture from %s", active.name)
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
			notify.Warn("[Video] Failed to close %s (%v)", active.name, err)
		}
	}()

	return nil
}

func (c codec) String() string {
	return c.name
}

func (c codec) cc4(vc *gocv.VideoCapture) float64 {
	for _, codec := range cached.codecs {
		if c == codec && c != CodecAny {
			return vc.ToCodec(c.String())
		}
	}
	return -1
}

func (d *dev) reset() {
	notify.Debug("[Video] Resetting %s device", d.name)

	lock.Lock()
	defer lock.Unlock()

	mat = splash.DeviceMat().Clone()
	size = mat.Size()

	config.Current.Video.Capture.Window.Name = config.MainDisplay
	config.Current.Video.Capture.Device.Index = config.NoVideoCaptureDevice
	config.Current.Video.Capture.Device.API = config.DefaultVideoCaptureAPI
	config.Current.Video.Capture.Device.Codec = config.DefaultVideoCaptureCodec

	notify.System("[Video] Capturing %s", config.Current.Video.Capture.Window.Name)

	d.name = Disabled
	d.index = config.NoVideoCaptureDevice
	d.fps = -1
	d.closeq = make(chan bool)
	d.closedq = make(chan bool)
}

func index(name string) int {
	if name == Disabled {
		return config.NoVideoCaptureDevice
	}

	for i := 0; i < 10; i++ {
		n, err := device.VideoCaptureDeviceName(i)
		if err != nil {
			notify.Error("[Video] Failed to find %s (%v)", name, err)
			return config.NoVideoCaptureDevice
		}
		if n == name {
			return i
		}
	}

	return config.NoVideoCaptureDevice
}

func poll(v *gocv.VideoCapture) properties {
	defaults := properties{
		resolution: image.Pt(
			int(v.Get(gocv.VideoCaptureFrameWidth)),
			int(v.Get(gocv.VideoCaptureFrameHeight)),
		),
		fps:        v.Get(gocv.VideoCaptureFPS),
		codec:      Codec(v.CodecString()),
		backend:    cached.api(v.Get(gocv.VideoCaptureBackend)),
		bitrate:    v.Get(gocv.VideoCaptureBitrate),
		buffersize: int(v.Get(gocv.VideoCaptureBufferSize)),
		rgb:        bool(int(v.Get(gocv.VideoCaptureConvertRGB)) == 1),
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

func set(vc *gocv.VideoCapture) error {
	p := poll(vc)

	required.fps = float64(config.Current.Video.Capture.Device.FPS)

	vc.Set(gocv.VideoCaptureFrameWidth, float64(required.resolution.X))
	vc.Set(gocv.VideoCaptureFrameHeight, float64(required.resolution.Y))
	vc.Set(gocv.VideoCaptureFPS, float64(config.Current.Video.Capture.Device.FPS))
	vc.Set(gocv.VideoCaptureConvertRGB, 1)

	c := Codec(config.Current.Video.Capture.Device.Codec)
	if c != CodecAny {
		vc.Set(gocv.VideoCaptureFOURCC, c.cc4(vc))
	}

	active.applied = poll(vc)

	if !active.applied.resolution.Eq(required.resolution) {
		return errors.Errorf("failed to set property: resolution %s", required.resolution)
	}

	if int(active.applied.fps) != int(required.fps) {
		return errors.Errorf("failed to set property: %.0f FPS", required.fps)
	}

	notify.System("[Video] Configured %s", active.name)
	notify.System("[Video]   Codec       %s → %s", p.codec, active.applied.codec)
	notify.System("[Video]   FPS         %.0f FPS → %.0f FPS", p.fps, active.applied.fps)
	notify.System("[Video]   Resolution  %s → %s", p.resolution, active.applied.resolution)
	notify.System("[Video]   Backend     %s → %s", p.backend, active.applied.backend)
	notify.System("[Video]   Bitrate     %.0f kb/s", active.applied.bitrate)
	notify.System("[Video]   BufferSize  %d", active.applied.buffersize)
	notify.System("[Video]   RGB         %t → %t", p.rgb, active.applied.rgb)

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
			notify.Error("[Video] Failed to stop %s", active.name)
			return
		}
	}
}
