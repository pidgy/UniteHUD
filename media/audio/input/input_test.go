package input

import (
	"testing"

	"github.com/gen2brain/malgo"
	"github.com/pidgy/unitehud/core/config"
)

func TestNewFromVideoCaptureDevice(t *testing.T) {
	backends := []malgo.Backend{}
	for i := malgo.BackendWasapi; i < malgo.BackendWebaudio; i++ {
		backends = append(backends, malgo.Backend(i))
	}

	config.Current.Video.Capture.Device.Index = 1

	ctx, err := malgo.InitContext(
		backends,
		malgo.ContextConfig{
			ThreadPriority: malgo.ThreadPriorityHigh,
			Alsa: malgo.AlsaContextConfig{
				UseVerboseDeviceEnumeration: 1,
			},
		},
		func(m string) {
			// t.Log(m)
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Free()

	d, err := ctx.Devices(malgo.Capture)
	if err != nil {
		t.Fatal(err)
	}

	for _, d := range d {
		malgo.InitDevice(ctx.Context, malgo.DeviceConfig{}, malgo.DeviceCallbacks{})
		println("\t", d.String())
	}
}

// TestDevices to parse discovered devices.
func TestDevices(t *testing.T) {
	for i := malgo.BackendWasapi; i < malgo.BackendWebaudio; i++ {
		println("Backend:", i)

		ctx, err := malgo.InitContext([]malgo.Backend{malgo.Backend(i)}, malgo.ContextConfig{}, nil)
		if err != nil {
			println("\t", err.Error())
			continue
		}
		defer ctx.Free()

		d, err := ctx.Devices(malgo.Capture)
		if err != nil {
			println("\t", err.Error())
			continue
		}

		for _, d := range d {
			println("\t", d.Name())
		}
	}
}
