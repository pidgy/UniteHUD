package input

import (
	"fmt"
	"io"

	"github.com/gen2brain/malgo"

	"github.com/pidgy/unitehud/audio/device"
	"github.com/pidgy/unitehud/notify"
)

type Device struct {
	ID      string
	Formats []malgo.DataFormat

	name      string
	isDefault bool

	reconnects int

	closed          bool
	closeq, closedq chan bool

	config malgo.DeviceConfig
}

func (d *Device) Is(name string) bool { return device.Is(d, name) }
func (d *Device) IsDefault() bool     { return d.isDefault }
func (d *Device) IsDisabled() bool    { return d == nil || d.name == device.Disabled }
func (d *Device) Name() string        { return d.name }

func New(name string) (*Device, error) {
	if name == device.Disabled {
		return &Device{name: device.Disabled}, nil
	}

	for _, d := range Devices() {
		if !device.Is(d, name) {
			continue
		}

		d.config = malgo.DefaultDeviceConfig(malgo.Capture)
		d.config.Capture.Format = malgo.FormatS16
		d.config.Capture.Channels = 1
		d.config.Playback.Format = malgo.FormatS16
		d.config.Playback.Channels = 1
		d.config.SampleRate = 44100
		d.config.Alsa.NoMMap = 1
		d.closed = true

		return d, nil
	}
	return nil, fmt.Errorf("failed to find capture device with term: %s", name)
}

func (d *Device) Active() bool {
	return !d.closed
}

func (d *Device) Close() {
	if !d.Active() {
		return
	}

	close(d.closeq)
	<-d.closedq
}

func (d *Device) Start(mctx malgo.Context, w io.ReadWriter, errq chan error, waitq chan bool) {
	d.closeq = make(chan bool)
	d.closedq = make(chan bool)
	d.closed = false

	defer func() { d.closed = true }()
	defer close(d.closedq)

	ready := false
	callbacks := malgo.DeviceCallbacks{
		Data: func(outputSamples, inputSamples []byte, frameCount uint32) {
			if d.closed {
				return
			}

			if !ready {
				ready = true
				close(waitq)
			}

			_, err := w.Write(inputSamples)
			if err != nil {
				if err == io.EOF {
					d.reconnects++
					return
				}

				errq <- err
			}
		},
	}

	device, err := malgo.InitDevice(mctx, d.config, callbacks)
	if err != nil {
		errq <- err
		return
	}
	defer device.Uninit()

	err = device.Start()
	if err != nil {
		errq <- err
		return
	}
	defer device.Stop()

	<-d.closeq
}

func (d *Device) String() string {
	return device.String(d)
}

func (d *Device) Type() device.Type {
	return device.Input
}

func Devices() (captures []*Device) {
	context, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		notify.Error("Failed to initialize audio devices (%v)", err)
		return nil
	}
	defer device.Free(context)

	d, err := context.Devices(malgo.Capture)
	if err != nil {
		notify.Error("Failed to discover audio capture devices (%v)", err)
		return nil
	}

	for _, info := range d {
		full, err := context.DeviceInfo(malgo.Capture, info.ID, malgo.Shared)
		if err != nil {
			notify.Warn("Failed to poll audio playback device \"%s\" info (%v)", info.ID, err)
		}

		captures = append(captures, &Device{
			ID:      info.ID.String(),
			Formats: full.Formats,

			name:      info.Name(),
			isDefault: info.IsDefault != 0,
		})
	}

	return captures
}
