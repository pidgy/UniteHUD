package input

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gen2brain/malgo"

	"github.com/pidgy/unitehud/core/config"
)

// ffmpeg -f dshow -i audio="AVerMedia HD Capture GC573 1" -vn -sn -f mp3 pipe:1 -v none
func TestFFMpeg(t *testing.T) {
	// args := []string{`-f`, `dshow`, `-i`, `audio="AVerMedia`, `HD`, `Capture`, `GC573`, `1"`, `-vn`, `-sn`, `-f`, `mp3`, `pipe:1`, `-v`, `quiet`}
	cmd := exec.Command("ffmpeg", strings.Split(`-f dshow -i audio="AVerMedia HD Capture GC573 1" -vn -sn -f mp3 pipe:1 -v quiet`, " ")...)
	cmd.Stdout = bytes.NewBuffer(make([]byte, 4096))

	err := cmd.Start()
	if err != nil {
		t.Fatal(err)
	}

	println(cmd.String())
	println("pid:", cmd.Process.Pid)

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := cmd.Stdout.Write(buf)
			if err != nil {
				break
			}

			println("read", n, "bytes")
		}
	}()

	time.AfterFunc(time.Second*2, func() {
		t.Fatalf("timeout")
	})

	err = cmd.Wait()
	if err != nil {
		e, ok := err.(*exec.ExitError)
		if ok {
			println("stderr:", string(e.Stderr))
		}

		t.Fatal(err)
	}
}

func TestNewFromVideoCaptureDevice(t *testing.T) {
	config.Current.Video.Capture.Device.Index = 1

	ctx, err := malgo.InitContext(
		[]malgo.Backend{malgo.BackendDsound, malgo.BackendWasapi},
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
