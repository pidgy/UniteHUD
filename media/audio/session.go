package audio

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/gen2brain/malgo"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/media/audio/device"
	"github.com/pidgy/unitehud/media/audio/input"
	"github.com/pidgy/unitehud/media/audio/output"
)

const (
	// Default is the default audio device name.
	Default  = device.Default
	// Disabled is the sentinel name for disabled audio devices.
	Disabled = device.Disabled
)

var (
	// Current is the active audio session.
	Current *Session
)

// Session manages input/output devices and streaming context.
type Session struct {
	Input, Output device.Device

	buffer io.ReadWriter

	errorq chan error
	waitq  chan bool

	context *malgo.AllocatedContext
}

// Close stops the current audio session and frees resources.
func Close() {
	notify.System("[Audio] Closing...")
	defer notify.Debug("[Audio] Closed")

	if Current == nil {
		return
	}

	Current.Input.Close()
	Current.Output.Close()

	Current.context.Free()
}

// Input switches the capture device and restarts audio.
func Input(name string) (err error) {
	if Current == nil {
		return nil
	}

	in, err := input.New(Current.context, name)
	if err != nil {
		return err
	}

	Current.Input.Close()
	Current.Output.Close()

	Current.Input = in
	config.Current.Audio.Capture.Device.Name = in.Name()

	return Start()
}

// Inputs returns available input devices for the current context.
func Inputs() []*input.Device {
	if Current == nil {
		return nil
	}

	return input.Devices(Current.context)
}

// Label returns a user-facing summary of input and output status.
func Label() string {
	if Current == nil {
		return "🔈 Audio Disabled"
	}

	speakers := []string{"🎤", "🔊"}

	if Current.Input.IsDisabled() {
		speakers[0] = "🎤"
	}

	if Current.Output.IsDisabled() {
		speakers[1] = "🔈"
	}

	return fmt.Sprintf("%s %s → %s %s", speakers[0], strings.Split(Current.Input.Name(), " (")[0], speakers[1], strings.Split(Current.Output.Name(), " (")[0])
}

// Open initializes the audio context and devices, then starts streaming.
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
			// notify.Debug("[Audio Session] %s", strings.Split(m, "\n")[0])
		},
	)
	if err != nil {
		return err
	}

	in, err := input.New(ctx, config.Current.Audio.Capture.Device.Name)
	if err != nil {
		notify.Warn("[Audio] <ini:f:create> input (%v)", err)
	}

	out, err := output.New(ctx, config.Current.Audio.Playback.Device.Name)
	if err != nil {
		notify.Warn("[Audio] <ini:f:create> output (%v)", err)
	}

	Current = &Session{
		Input:  in,
		Output: out,

		buffer: bytes.NewBuffer(make([]byte, 0)),

		errorq: make(chan error),
		waitq:  make(chan bool),

		context: ctx,
	}

	return Start()
}

// Output switches the playback device and restarts audio.
func Output(name string) (err error) {
	if Current == nil {
		return nil
	}

	out, err := output.New(Current.context, name)
	if err != nil {
		return err
	}

	Current.Input.Close()
	Current.Output.Close()

	Current.Output = out
	config.Current.Audio.Playback.Device.Name = out.Name()

	return Start()
}

// Outputs returns available output devices for the current context.
func Outputs() []*output.Device {
	if Current == nil {
		return nil
	}

	return output.Devices(Current.context)
}

// Restart reloads the current input/output devices.
func Restart() {
	Current.Input.Close()
	Current.Output.Close()

	in, err := input.New(Current.context, config.Current.Audio.Capture.Device.Name)
	if err != nil {
		notify.Warn("[Audio] <ini:f:create> input (%v)", err)
	}

	out, err := output.New(Current.context, config.Current.Audio.Playback.Device.Name)
	if err != nil {
		notify.Warn("[Audio] <ini:f:create> output (%v)", err)
	}

	Current.Input = in
	Current.Output = out
}

// Start begins streaming from input to output.
func Start() error {
	if Current.Input.IsDisabled() || Current.Output.IsDisabled() {
		notify.System("[Audio] Disabled")
		return nil
	}

	notify.System("[Audio] Starting %s", Current)

	err := Current.Input.Start(Current.context.Context, Current.buffer)
	if err != nil {
		return err
	}

	err = Current.Output.Start(Current.context.Context, Current.buffer)
	if err != nil {
		return err
	}

	return nil
}

// String formats the session device route.
func (s *Session) String() string {
	return fmt.Sprintf("%s -> %s", s.Input, s.Output)
}
