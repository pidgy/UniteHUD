package audio

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/gen2brain/malgo"

	"github.com/pidgy/unitehud/avi/audio/device"
	"github.com/pidgy/unitehud/avi/audio/input"
	"github.com/pidgy/unitehud/avi/audio/output"
	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
)

const (
	Default  = device.Default
	Disabled = device.Disabled
)

var (
	current *session
)

type session struct {
	input, output device.Device

	buffer io.ReadWriter

	errorq chan error
	waitq  chan bool

	context *malgo.AllocatedContext
}

func Open() error {
	ctx, err := malgo.InitContext(
		[]malgo.Backend{
			malgo.BackendDsound,
			malgo.BackendWasapi,
			malgo.BackendWinmm,
		},
		malgo.ContextConfig{
			ThreadPriority: malgo.ThreadPriorityDefault,
		},
		func(m string) {
			// notify.Debug("Audio Session: %s", strings.Split(m, "\n")[0])
		},
	)
	if err != nil {
		return err
	}

	in, err := input.New(ctx, config.Current.Audio.Capture.Device.Name)
	if err != nil {
		notify.Warn("Audio: Failed to create input (%v)", err)
	}

	out, err := output.New(ctx, config.Current.Audio.Playback.Device.Name)
	if err != nil {
		notify.Warn("Audio: Failed to create output (%v)", err)
	}

	current = &session{
		input:  in,
		output: out,

		buffer: bytes.NewBuffer(make([]byte, 0)),

		errorq: make(chan error),
		waitq:  make(chan bool),

		context: ctx,
	}

	return Start()
}

func Close() {
	notify.System("Audio: Closing...")
	defer notify.Debug("Audio: Closed")

	if current == nil {
		return
	}

	current.input.Close()
	current.output.Close()

	current.context.Free()
}

func Input(name string) (err error) {
	if current == nil {
		return nil
	}

	in, err := input.New(current.context, name)
	if err != nil {
		return err
	}

	current.input.Close()
	current.output.Close()

	current.input = in
	config.Current.Audio.Capture.Device.Name = in.Name()

	return Start()
}

func Inputs() []*input.Device {
	if current == nil {
		return nil
	}

	return input.Devices(current.context)
}

func Label() string {
	if current == nil {
		return "🔈 Audio Disabled"
	}

	speakers := []string{"🎤", "🔊"}

	if current.input.IsDisabled() {
		speakers[0] = "🎤"
	}

	if current.output.IsDisabled() {
		speakers[1] = "🔈"
	}

	return fmt.Sprintf("%s %s → %s %s", speakers[0], strings.Split(current.input.Name(), " (")[0], speakers[1], strings.Split(current.output.Name(), " (")[0])
}

func Output(name string) (err error) {
	if current == nil {
		return nil
	}

	out, err := output.New(current.context, name)
	if err != nil {
		return err
	}

	current.input.Close()
	current.output.Close()

	current.output = out
	config.Current.Audio.Playback.Device.Name = out.Name()

	return Start()
}

func Outputs() []*output.Device {
	if current == nil {
		return nil
	}

	return output.Devices(current.context)
}

func Restart() {
	current.input.Close()
	current.output.Close()

	in, err := input.New(current.context, config.Current.Audio.Capture.Device.Name)
	if err != nil {
		notify.Warn("Audio: Failed to create input (%v)", err)
	}

	out, err := output.New(current.context, config.Current.Audio.Playback.Device.Name)
	if err != nil {
		notify.Warn("Audio: Failed to create output (%v)", err)
	}

	current.input = in
	current.output = out
}

func Start() error {
	notify.System("Audio: Starting %s", current)

	if current.input.IsDisabled() || current.output.IsDisabled() {
		notify.Debug("Audio: Ignoring session, i/o disabled")
		return nil
	}

	err := current.input.Start(current.context.Context, current.buffer)
	if err != nil {
		return err
	}

	err = current.output.Start(current.context.Context, current.buffer)
	if err != nil {
		return err
	}

	return nil
}

func (s *session) String() string {
	return fmt.Sprintf("%s -> %s", s.input, s.output)
}
