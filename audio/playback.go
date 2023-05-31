package audio

import (
	"fmt"
	"io"

	"github.com/gen2brain/malgo"
	"github.com/pidgy/unitehud/notify"
)

type playback struct {
	ID      string
	Formats []malgo.DataFormat

	name      string
	isDefault bool

	config malgo.DeviceConfig

	reconnects int
}

func (p *playback) IsDefault() bool { return p.isDefault }
func (p *playback) Name() string    { return p.name }

func newPlayback(name string) (*playback, error) {
	if name == DeviceDisabled {
		return &playback{name: name}, nil
	}

	for _, d := range playbackDevices() {
		if !isDevice(d, name) {
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
	return nil, fmt.Errorf("failed to find playback device with term: %s", name)
}

// Playback streams samples from a reader to the sound device.
// The function initializes a playback device in the default context using
// provide stream configuration.
// Playback will commence playing the samples provided from the reader until either the
// reader returns an error, or the context signals done.
func (p *playback) start(mctx malgo.Context, r io.Reader, errq chan error, closeq, waitq chan bool) {
	<-waitq

	aborted := false

	callbacks := malgo.DeviceCallbacks{
		Data: func(outputSamples, inputSamples []byte, frameCount uint32) {
			if aborted {
				return
			}

			if frameCount == 0 {
				return
			}

			_, err := io.ReadFull(r, outputSamples)
			if err != nil {
				if err == io.EOF {
					p.reconnects++
					return
				}

				aborted = true
				errq <- err
			}
			return
		},
	}

	device, err := malgo.InitDevice(mctx, p.config, callbacks)
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

	aborted = true
}

func playbackDevices() (playbacks []*playback) {
	context, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		notify.Error("Failed to initialize audio devices (%v)", err)
		return nil
	}
	defer free(context)

	d, err := context.Devices(malgo.Playback)
	if err != nil {
		notify.Error("Failed to discover audio playback devices (%v)", err)
		return nil
	}

	for _, info := range d {
		full, err := context.DeviceInfo(malgo.Playback, info.ID, malgo.Shared)
		if err != nil {
			notify.Warn("Failed to poll audio playback device \"%s\" info (%v)", info.ID, err)
		}

		playbacks = append(playbacks, &playback{
			ID:      info.ID.String(),
			Formats: full.Formats,

			name:      info.Name(),
			isDefault: info.IsDefault != 0,
		})
	}

	return playbacks
}
