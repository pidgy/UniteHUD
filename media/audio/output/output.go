package output

import (
	"fmt"
	"io"

	"github.com/gen2brain/malgo"

	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/media/audio/device"
)

// Device represents an audio playback device.
type Device struct {
	ID      string
	Formats []malgo.DataFormat

	name      string
	isDefault bool

	config malgo.DeviceConfig

	active            bool
	closingq, closedq chan bool

	reconnects int
}

// New returns a playback device by name or disabled when requested.
func New(ctx *malgo.AllocatedContext, name string) (*Device, error) {
	if name == device.Disabled {
		return &Device{name: device.Disabled}, nil
	}
	if name == "" {
		return &Device{name: device.Disabled}, nil
	}

	for _, d := range Devices(ctx) {
		if !device.Is(d, name) {
			continue
		}

		d.config = malgo.DefaultDeviceConfig(malgo.Playback)
		d.config.Capture.Format = malgo.FormatS16
		d.config.Capture.Channels = 1
		d.config.Playback.Format = malgo.FormatS16
		d.config.Playback.Channels = 1
		d.config.SampleRate = 44100
		d.config.Alsa.NoMMap = 1

		return d, nil
	}

	return nil, fmt.Errorf("<ini:f:find> playback device: %s", name)
}

// Active reports whether the device is playing.
func (d *Device) Active() bool {
	return d == nil || d.active
}

// Close stops the playback device if active.
func (d *Device) Close() {
	if !d.Active() {
		return
	}

	notify.System("[Audio Output] Closing %s", d.name)

	close(d.closingq)
	<-d.closedq
}

// Is reports whether this device matches the given name.
func (d *Device) Is(name string) bool {
	return device.Is(d, name)
}

// IsDefault reports whether this device is the system default.
func (d *Device) IsDefault() bool {
	return d.isDefault
}

// IsDisabled reports whether playback is disabled.
func (d *Device) IsDisabled() bool {
	return d.name == device.Disabled
}

// Name returns the device display name.
func (d *Device) Name() string {
	return d.name
}


// Start begins playback by reading samples from r.
func (d *Device) Start(mctx malgo.Context, r io.ReadWriter) error {
	if d.IsDisabled() {
		return nil
	}

	if d.Active() {
		return fmt.Errorf("%s: already active", d.name)
	}

	defer notify.Debug("[Audio Output] Started %s", d.Name())

	errq := make(chan error)

	go func() {
		defer notify.Debug("[Audio Output] Closed %s", d.Name())

		d.closingq = make(chan bool)
		d.closedq = make(chan bool)
		d.active = true

		defer func() { d.active = false }()
		defer close(d.closedq)

		callbacks := malgo.DeviceCallbacks{
			Data: func(outputSamples, inputSamples []byte, frameCount uint32) {
				if !d.Active() {
					return
				}

				if frameCount == 0 {
					return
				}

				_, err := io.ReadFull(r, outputSamples)
				if err != nil {
					if err == io.EOF || err == io.ErrUnexpectedEOF {
						d.reconnects++
						return
					}
					notify.Warn("[Audio Output] Playback error (%s: %v)", d.name, err)
				}
			},
		}

		device, err := malgo.InitDevice(mctx, d.config, callbacks)
		if err != nil {
			errq <- fmt.Errorf("%s: %v", d.name, err)
			return
		}
		defer device.Uninit()

		err = device.Start()
		if err != nil {
			errq <- fmt.Errorf("%s: %v", d.name, err)
			return
		}
		defer func() {
			err := device.Stop()
			if err != nil {
				notify.Error("[Audio Output] <ini:f:stop> device (%v)", err)
				return
			}
		}()

		close(errq)
		<-d.closingq
	}()

	return <-errq
}

// String returns a safe display string for the device.
func (d *Device) String() string {
	return device.String(d)
}

// Type identifies this device as a playback output.
func (d *Device) Type() device.Type {
	return device.Output
}

// Devices enumerates available playback devices for the context.
func Devices(ctx *malgo.AllocatedContext) (playbacks []*Device) {
	d, err := ctx.Devices(malgo.Playback)
	if err != nil {
		notify.Error("[Audio Output] <ini:f:find> devices (%v)", err)
		return nil
	}

	for _, info := range d {
		full, err := ctx.DeviceInfo(malgo.Playback, info.ID, malgo.Shared)
		if err != nil {
			notify.Warn("[Audio Output] <ini:f:find> device information for %s (%v)", info.ID, err)
		}

		playbacks = append(playbacks, &Device{
			ID:      info.ID.String(),
			Formats: full.Formats,

			name:      info.Name(),
			isDefault: info.IsDefault != 0,
		})
	}

	return playbacks
}
