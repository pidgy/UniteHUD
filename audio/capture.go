package audio

import (
	"fmt"
	"io"

	"github.com/gen2brain/malgo"
	"github.com/pidgy/unitehud/notify"
)

type capture struct {
	ID      string
	Formats []malgo.DataFormat

	name      string
	isDefault bool

	config malgo.DeviceConfig
}

func (c *capture) IsDefault() bool { return c.isDefault }
func (c *capture) Name() string    { return c.name }

func newCapture(name string) (*capture, error) {
	if name == DeviceDisabled {
		return &capture{name: name}, nil
	}

	for _, d := range captureDevices() {
		if !isDevice(d, name) {
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

func (c *capture) start(mctx malgo.Context, w io.Writer, errq chan error, closeq, waitq chan bool) {
	ready := false
	aborted := false

	callbacks := malgo.DeviceCallbacks{
		Data: func(outputSamples, inputSamples []byte, frameCount uint32) {
			if aborted {
				return
			}

			if !ready {
				ready = true
				close(waitq)
			}

			_, err := w.Write(inputSamples)
			if err != nil {
				aborted = true
				errq <- err
				return
			}
		},
	}

	device, err := malgo.InitDevice(mctx, c.config, callbacks)
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

	<-closeq

	errq <- err
	aborted = true
}

func captureDevices() (captures []*capture) {
	context, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		notify.Error("Failed to initialize audio devices (%v)", err)
		return nil
	}
	defer free(context)

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

		captures = append(captures, &capture{
			ID:      info.ID.String(),
			Formats: full.Formats,

			name:      info.Name(),
			isDefault: info.IsDefault != 0,
		})
	}

	return captures
}
