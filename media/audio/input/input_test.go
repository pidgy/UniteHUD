package input

import (
	"testing"

	"github.com/gen2brain/malgo"
	"github.com/pidgy/unitehud/core/config"
)

func TestNewFromVideoCaptureDevice(t *testing.T) {
	config.Current.Video.Capture.Device.Index = 1

	ctx, err := malgo.InitContext(
		[]malgo.Backend{
			malgo.BackendWasapi,
		},
		malgo.ContextConfig{
			ThreadPriority: malgo.ThreadPriorityHigh,
		},
		func(m string) {
			// t.Log(m)
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Free()

	d, err := NewFromVideoCaptureDevice(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if d.IsDisabled() {
		t.Fatalf("device %d is disabled", config.Current.Video.Capture.Device.Index)
	}
}

// TestDevices to parse discovered devices.
func TestDevices(t *testing.T) {
	for i := malgo.BackendWasapi; i < malgo.BackendWebaudio; i++ {
		println(i, "----------------------------")

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
