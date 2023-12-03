package audio

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/gen2brain/malgo"

	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/media/audio/device"
	"github.com/pidgy/unitehud/media/audio/input"
	"github.com/pidgy/unitehud/media/audio/output"
)

const (
	Default  = device.Default
	Disabled = device.Disabled
)

var (
	current *session

	SessionClosed = fmt.Errorf("session closed")
)

type session struct {
	In  device.Device
	Out device.Device

	buffer io.ReadWriter

	errorq chan error
	waitq  chan bool

	context *malgo.AllocatedContext
}

func Open() error {
	notify.System("Audio: Opening session")

	ctx, err := malgo.InitContext(
		[]malgo.Backend{
			malgo.BackendDsound,
		},
		malgo.ContextConfig{
			CoreAudio: malgo.CoreAudioConfig{
				SessionCategory: malgo.IOSSessionCategoryPlayback,
			},
		},
		func(m string) {
			// notify.Debug("Audio Session: %s", strings.Split(m, "\n")[0])
		},
	)
	if err != nil {
		return err
	}

	in, err := input.New(ctx, Disabled)
	if err != nil {
		return err
	}

	out, err := output.New(ctx, Default)
	if err != nil {
		return err
	}

	current = &session{
		In:  in,
		Out: out,

		buffer: bytes.NewBuffer(make([]byte, 0)),

		errorq: make(chan error),
		waitq:  make(chan bool),

		context: ctx,
	}

	return Start()
}

func Inputs() []*input.Device {
	return input.Devices(current.context)
}

func Label() string {
	if current == nil {
		return "🔉"
	}

	mic := "🎤"
	if current.In.IsDisabled() {
		mic = "❌"
	}

	speaker := "🔊"
	if current.Out.IsDisabled() {
		speaker = "🔈"
	}

	return fmt.Sprintf("%s %s / %s %s", mic, strings.Split(current.In.Name(), " (")[0], speaker, strings.Split(current.Out.Name(), " (")[0])
}

func Outputs() []*output.Device {
	return output.Devices(current.context)
}

func Close() error {
	notify.System("Audio: Closing...")

	if current == nil {
		return nil
	}

	current.In.Close()
	current.Out.Close()

	return nil
}

func Input(name string) (err error) {
	if current == nil {
		return nil
	}

	notify.System("Audio: Swapping input %s", name)

	current.In.Close()
	current.Out.Close()

	current.In, err = input.New(current.context, name)
	if err != nil {
		return err
	}

	return Start()
}

func Output(name string) (err error) {
	if current == nil {
		return nil
	}

	notify.System("Audio: Swapping output %s", name)

	current.In.Close()
	current.Out.Close()

	current.Out, err = output.New(current.context, name)
	if err != nil {
		return err
	}

	return Start()
}

func Start() error {
	if current.In.IsDisabled() || current.Out.IsDisabled() {
		return nil
	}

	notify.System("Audio: Starting session (%s)", current)

	err := current.In.Start(current.context.Context, current.buffer)
	if err != nil {
		return err
	}

	err = current.Out.Start(current.context.Context, current.buffer)
	if err != nil {
		return err
	}

	return nil
}

func (s *session) String() string {
	return fmt.Sprintf("%s -> %s", s.In, s.Out)
}
