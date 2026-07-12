package device

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const. Then remove this comment.

import (
	"fmt"
	"image"
	"runtime"
	"slices"
	"strings"
	"time"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/media/device"
	"github.com/pidgy/unitehud/media/img"
	"github.com/pidgy/unitehud/media/img/splash"
	"github.com/pidgy/unitehud/media/video/fps"
	"github.com/pidgy/unitehud/media/video/monitor"
	"github.com/pidgy/unitehud/system/lang"
)

const (
	Disabled = "Disabled"
)

type (
	api struct {
		gocv gocv.VideoCaptureAPI
		name string
	}

	cache struct {
		videos []string
		apis   []api
		codecs []codec
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

		fourcc  int64
		codec   codec
		backend api

		buffersize    int
		rgb           bool
		convertRGB    bool
		hwcceleration bool
	}
)

var (
	active = &dev{}

	codecAny = codec{"Any"}

	required = properties{
		resolution: image.Pt(1920, 1080),
	}

	mat  = splash.DeviceMat().Clone()
	size = mat.Size()
	// lock = &sync.RWMutex{}

	cached = cache{
		videos: make([]string, 100),
		apis:   []api{},
		codecs: []codec{
			codecAny,
			{"NV12"},
			{"YUY2"},
			{"UYVY"},
			{"YVYU"},
			{"VYUY"},
			{"I420"},
			{"YV12"},
			{"P010"},
			{"MJPG"},
			{"JPEG"},
			{"H264"},
			{"AVC1"},
		},
	}

	captures float32

	apiUnknown = api{
		name: "Unknown",
		gocv: -1,
	}
)

func init() {
	active.reset()

	if exe.Debug {
		go func() {
			last := captures
			for range time.NewTicker(time.Minute).C {
				if int(last) == int(captures) {
					continue
				}

				notify.Debug("[Device] Captures per second: %.1f", captures/60)
				last = captures
				captures = 0
			}
		}()
	}

	go func() {
		cached.apis = append(cached.apis,
			api{
				gocv: gocv.VideoCaptureAny,
				name: "Any",
			},
		)

		vrt := gocv.VideoRegistryType{}
		for _, b := range vrt.GetCameraBackends() {
			cached.apis = append(cached.apis,
				api{
					gocv: b,
					name: lang.Title(strings.ReplaceAll(b.String(), "video-capture-", "")),
				},
			)
		}

		// Max API value: gocv.VideoCaptureXINE.
		// for i := gocv.VideoCaptureAny; i < gocv.VideoCaptureXINE; i++ {
		// 	s := i.String()
		// 	if s != "" {
		// 		cached.apis = append(cached.apis, api{
		// 			gocv: i,
		// 			name: strings.ReplaceAll(i.String(), "video-capture-", ""),
		// 		})
		// 	}
		// }

		for ; ; time.Sleep(time.Second * 5) {
			names := []string{}
			for i := 0; ; i++ {
				n, err := device.VideoCaptureDeviceName(i)
				if err != nil {
					break
				}
				names = append(names, n)
			}
			cached.videos = names
		}
	}()
}

func API(name string) api {
	for i, n := range cached.apis {
		if strings.EqualFold(name, n.name) {
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
	// lock.RLock()
	// defer lock.RUnlock()

	if mat.Empty() {
		return nil, nil
	}

	if !r.In(monitor.DefaultResolution) {
		return nil, fmt.Errorf("illegal boundaries: %s", r)
	}

	mrect := image.Rect(0, 0, size[1], size[0])

	if !r.In(mrect) {
		return splash.AsRGBA(splash.Invalid()), fmt.Errorf("illegal boundaries: %s", r)
	}

	captures++

	return img.RGBA(mat.Region(r))
}

func Close() {
	config.Current.SetDefaultVideoCaptureDevice()

	if active.index == config.NoVideoCaptureDevice {
		notify.Debug("[Device] Device disabled, ignoring close")
		return
	}

	notify.System("[Device] Closing %s...", active.name)
	defer notify.System("[Device] Closed %s", active.name)

	stop()

	active.reset()
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

	if cached.videos[index] != "" {
		return cached.videos[index]
	}

	var err error

	cached.videos[index], err = device.VideoCaptureDeviceName(index)
	if err != nil {
		notify.Error("[Device] <ini:f:find> device %d name (%v)", index, err)
		return fmt.Sprintf("%d", index)
	}

	return cached.videos[index]
}

func Open() error {
	if config.Current.Video.Capture.Device.Index == config.NoVideoCaptureDevice {
		notify.Debug("[Device] Disabled, ignorning call to open")
		return nil
	}

	if active.index != config.NoVideoCaptureDevice {
		notify.Debug("[Device] %s is already active", active.name)
		return nil
	}

	idx := index(config.Current.Video.Capture.Device.Name)
	if idx == config.NoVideoCaptureDevice {
		active.reset()
		return fmt.Errorf("%s was not found", config.Current.Video.Capture.Device.Name)
	}
	if idx != config.Current.Video.Capture.Device.Index {
		config.Current.Video.Capture.Device.Index = idx
		notify.Warn("[Device] Invalid index for %s", config.Current.Video.Capture.Device.Name)
	}

	active.index = config.Current.Video.Capture.Device.Index
	active.name = Name(config.Current.Video.Capture.Device.Index)
	active.closeq = make(chan bool)
	active.closedq = make(chan bool)

	notify.System("[Device] Opening %s", active.name)

	err := start()
	if err != nil {
		active.reset()
		return err
	}

	return nil
}

func Resolution() image.Point {
	return active.applied.resolution
}

func Restart() error {
	notify.Debug("[Device] Restarting with API: %s", config.Current.Video.Capture.Device.API)
	prev := config.Current.Video.Capture.Device
	Close()
	config.Current.Video.Capture.Device = prev
	return Open()
}

func Sources() (indexes []int) {
	for i, name := range cached.videos {
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

func (c *cache) api(f gocv.VideoCaptureAPI) api {
	for _, a := range c.apis {
		if a.gocv == f {
			return a
		}
	}
	return apiUnknown
}

func (c codec) String() string {
	return c.name
}

func codecByName(name string) codec {
	for _, codec := range cached.codecs {
		if strings.EqualFold(name, codec.String()) {
			return codec
		}
	}
	return codecAny
}

var toCodec = (&gocv.VideoCapture{}).ToCodec

func fourcc(name string) float64 {
	c := codecByName(name)
	if c != codecAny {
		if slices.Contains(cached.codecs, c) {
			return toCodec(c.String())
		}
	}
	return -1
}

func (d *dev) reset() {
	notify.Debug("[Device] Resetting %s", d.name)

	// lock.Lock()
	// defer lock.Unlock()

	mat = splash.DeviceMat().Clone()
	size = mat.Size()

	config.Current.SetDefaultVideoCaptureDevice()

	notify.System("[Device] Capturing %s", config.Current.Video.Capture.Monitor.Name)

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

	for i := range 10 {
		n, err := device.VideoCaptureDeviceName(i)
		if err != nil {
			notify.Error("[Device] <ini:f:find> %s (%v)", name, err)
			return config.NoVideoCaptureDevice
		}
		if n == name {
			return i
		}
	}

	return config.NoVideoCaptureDevice
}

func (p properties) String() string {
	out := ""
	out += fmt.Sprintf("[Device] Properties: %s", active.name)
	out += fmt.Sprintf("\n    → Codec:          %s", active.applied.codec)
	out += fmt.Sprintf("\n    → FPS:            %.0f FPS", active.applied.fps)
	out += fmt.Sprintf("\n    → Resolution:     %s", active.applied.resolution)
	out += fmt.Sprintf("\n    → API:            %s", active.applied.backend)
	out += fmt.Sprintf("\n    → Bitrate:        %.0f kb/s", active.applied.bitrate)
	out += fmt.Sprintf("\n    → BufferSize:     %d", active.applied.buffersize)
	out += fmt.Sprintf("\n    → RGB:            %t", active.applied.rgb)
	out += fmt.Sprintf("\n    → Convert RGB:    %t", active.applied.convertRGB)
	out += fmt.Sprintf("\n    → HWAcceleration: %t", active.applied.hwcceleration)
	return out
}

func (p *properties) poll(v *gocv.VideoCapture) *properties {
	p.resolution = image.Pt(
		int(v.Get(gocv.VideoCaptureFrameWidth)),
		int(v.Get(gocv.VideoCaptureFrameHeight)),
	)
	p.fps = v.Get(gocv.VideoCaptureFPS)
	p.codec = codecByName(v.CodecString())
	p.fourcc = int64(v.Get(gocv.VideoCaptureFOURCC))
	p.backend = cached.api(gocv.VideoCaptureAPI(v.Get(gocv.VideoCaptureBackend)))
	p.bitrate = v.Get(gocv.VideoCaptureBitrate)
	p.buffersize = int(v.Get(gocv.VideoCaptureBufferSize))
	p.rgb = bool(int(v.Get(gocv.VideoCaptureConvertRGB)) == 1)
	p.hwcceleration = bool(v.Get(gocv.VideoCaptureHWAcceleration) > 0)
	p.convertRGB = bool(v.Get(gocv.VideoCaptureConvertRGB) > 0)

	// p = &properties{
	// 	resolution: image.Pt(
	// 		int(v.Get(gocv.VideoCaptureFrameWidth)),
	// 		int(v.Get(gocv.VideoCaptureFrameHeight)),
	// 	),
	// 	fps:           v.Get(gocv.VideoCaptureFPS),
	// 	codec:         codecByName(v.CodecString()),
	// 	backend:       cached.api(gocv.VideoCaptureAPI(v.Get(gocv.VideoCaptureBackend))),
	// 	bitrate:       v.Get(gocv.VideoCaptureBitrate),
	// 	buffersize:    int(v.Get(gocv.VideoCaptureBufferSize)),
	// 	rgb:           bool(int(v.Get(gocv.VideoCaptureConvertRGB)) == 1),
	// 	hwcceleration: bool(v.Get(gocv.VideoCaptureHWAcceleration) > 0),
	// 	convertRGB:    bool(v.Get(gocv.VideoCaptureConvertRGB) > 0),
	// }
	return p
}

// func set(vc *gocv.VideoCapture) error {
// 	required.fps = float64(config.Current.Video.Capture.Device.FPS)

// 	// Image detection is only enabled for 1920x1080.
// 	vc.Set(gocv.VideoCaptureFrameWidth, float64(required.resolution.X))
// 	vc.Set(gocv.VideoCaptureFrameHeight, float64(required.resolution.Y))
// 	vc.Set(gocv.VideoCaptureFPS, float64(config.Current.Video.Capture.Device.FPS))
// 	vc.Set(gocv.VideoCaptureBackend, float64(API(config.Current.Video.Capture.Device.API).gocv))
// 	vc.Set(gocv.VideoCaptureConvertRGB, 1)
// 	vc.Set(gocv.VideoCaptureFOURCC, codecByName(config.Current.Video.Capture.Device.Codec).fourcc())

// 	active.applied = poll(vc)

// 	if !active.applied.resolution.Eq(required.resolution) {
// 		return fmt.Errorf("property: invalid resolution %s", active.applied.resolutionString)
// 	}

// 	if int(active.applied.fps) != int(required.fps) {
// 		return fmt.Errorf("<ini:f:set> property: %.0f FPS", required.fps)
// 	}

// 	return nil
// }

func start() error {
	if config.Current.Video.Capture.Device.API == "" {
		config.Current.Video.Capture.Device.API = config.DefaultVideoCaptureAPI
	}

	api := API(config.Current.Video.Capture.Device.API)

	notify.Debug("[Device] Capturing %s with API: %s, Codec: %s (%.0f)",
		active.name,
		api.name,
		config.Current.Video.Capture.Device.Codec,
		fourcc(config.Current.Video.Capture.Device.Codec),
	)

	video, err := gocv.OpenVideoCaptureWithAPI(active.index, api.gocv)
	if err != nil {
		return fmt.Errorf("%s: %v", config.Current.Video.Capture.Device.Name, err)
	}

	video.Set(gocv.VideoCaptureFrameWidth, float64(required.resolution.X))
	video.Set(gocv.VideoCaptureFrameHeight, float64(required.resolution.Y))
	video.Set(gocv.VideoCaptureFPS, float64(config.Current.Video.Capture.Device.FPS))
	video.Set(gocv.VideoCaptureBackend, float64(API(config.Current.Video.Capture.Device.API).gocv))
	video.Set(gocv.VideoCaptureConvertRGB, 1)
	video.Set(gocv.VideoCaptureHWDevice, float64(active.index))
	video.Set(gocv.VideoCaptureHWAcceleration, 1)
	video.Set(gocv.VideoCaptureConvertRGB, 1)
	if config.Current.Video.Capture.Device.Codec != config.DefaultVideoCaptureCodec {
		video.Set(gocv.VideoCaptureFOURCC, fourcc(config.Current.Video.Capture.Device.Codec))
	}

	notify.System("%s", active.applied.poll(video))

	// err = set(video)
	// if err != nil {
	// 	active.reset()
	// 	return err
	// }

	// notify.System("[Device] Configured %s", active.name)
	// notify.System("[Device]   Codec       %s → %s", before.codec, active.applied.codec)
	// notify.System("[Device]   FPS         %.0f FPS → %.0f FPS", before.fps, active.applied.fps)
	// notify.System("[Device]   Resolution  %s → %s", before.resolution, active.applied.resolution)
	// notify.System("[Device]   API         %s → %s", before.backend, active.applied.backend)
	// notify.System("[Device]   Bitrate     %.0f kb/s", active.applied.bitrate)
	// notify.System("[Device]   BufferSize  %d", active.applied.buffersize)
	// notify.System("[Device]   RGB         %t → %t", before.rgb, active.applied.rgb)

	config.Current.Video.Capture.Device.API = active.applied.backend.name
	config.Current.Video.Capture.Device.Codec = active.applied.codec.name

	go func(video *gocv.VideoCapture) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		defer close(active.closedq)

		ms := fps.Milliseconds(config.Current.Video.Capture.Device.FPS)
		tick := time.NewTicker(ms)
		tock := time.NewTicker(time.Second)

		running := func() bool {
			select {
			case <-active.closeq:
				return false
			default:
				return true
			}
		}

		for frames := float64(0); running(); frames++ {
			// lock.Lock()
			ok := video.Read(&mat)
			if !ok {
				defer active.reset()
				notify.Error("[Device] <ini:f:capture> %s", active.name)
				// lock.Unlock()
				goto close
			}
			// lock.Unlock()

			size = mat.Size()

			select {
			case <-tick.C:
				tick.Reset(ms)
			case <-tock.C:
				tock.Reset(time.Second)
				active.fps = frames
				frames = 0
			}
		}

	close:
		err := video.Close()
		if err != nil {
			notify.Error("[Device] <ini:f:close> %s (%v)", active.name, err)
		}
	}(video)

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
			notify.Error("[Device] <ini:f:stop> %s", active.name)
			return
		}
	}
}
