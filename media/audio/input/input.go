package input

import (
	"fmt"
	"io"

	"github.com/gen2brain/malgo"
	"github.com/pkg/errors"

	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/media/audio/device"
)

type Device struct {
	ID      string
	Formats []malgo.DataFormat

	name      string
	isDefault bool

	reconnects int

	active            bool
	closingq, closedq chan bool

	config malgo.DeviceConfig
}

func (d *Device) Is(name string) bool { return device.Is(d, name) }
func (d *Device) IsDefault() bool     { return d.isDefault }
func (d *Device) IsDisabled() bool    { return d == nil || d.name == device.Disabled }
func (d *Device) Name() string        { return d.name }

func New(ctx *malgo.AllocatedContext, name string) (*Device, error) {
	if name == device.Disabled {
		return &Device{name: device.Disabled}, nil
	}

	for _, d := range Devices(ctx) {
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

		return d, nil
	}

	return nil, fmt.Errorf("failed to find capture device with term: %s", name)
}

func (d *Device) Active() bool {
	return d.active
}

func (d *Device) Close() {
	notify.System("Audio: Closing input %s", d.name)

	if !d.Active() {
		return
	}

	close(d.closingq)
	<-d.closedq
}

func (d *Device) Start(mctx malgo.Context, w io.ReadWriter) error {
	if d.Active() {
		return errors.Wrap(fmt.Errorf("already active"), d.name)
	}

	errq := make(chan error)
	go func() {
		d.closingq = make(chan bool)
		d.closedq = make(chan bool)
		d.active = true

		defer func() {
			d.active = false
		}()
		defer close(d.closedq)

		callbacks := malgo.DeviceCallbacks{
			Data: func(outputSamples, inputSamples []byte, frameCount uint32) {
				if !d.Active() {
					return
				}

				_, err := w.Write(inputSamples)
				if err != nil {
					if err == io.EOF || err == io.ErrUnexpectedEOF {
						d.reconnects++
						return
					}
					notify.Error("Audio: Capture failed (%v)", errors.Wrap(err, d.name))
				}
			},
		}

		device, err := malgo.InitDevice(mctx, d.config, callbacks)
		if err != nil {
			errq <- errors.Wrap(err, d.name)
			return
		}
		defer device.Uninit()

		err = device.Start()
		if err != nil {
			errq <- errors.Wrap(err, d.name)
			return
		}
		defer device.Stop()

		close(errq)
		<-d.closingq
	}()

	return <-errq
}

func (d *Device) String() string {
	return device.String(d)
}

func (d *Device) Type() device.Type {
	return device.Input
}

func Devices(ctx *malgo.AllocatedContext) (captures []*Device) {
	d, err := ctx.Devices(malgo.Capture)
	if err != nil {
		notify.Error("Failed to discover audio capture devices (%v)", err)
		return nil
	}

	for _, info := range d {
		full, err := ctx.DeviceInfo(malgo.Capture, info.ID, malgo.Shared)
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
