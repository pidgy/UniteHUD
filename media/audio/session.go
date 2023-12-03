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

var SessionClosed = fmt.Errorf("session closed")

type Session struct {
	In  device.Device
	Out device.Device

	buffer io.ReadWriter

	errorq chan error
	waitq  chan bool

	closed  bool
	context *malgo.AllocatedContext
}

func New(i, o string) (*Session, error) {
	ctx, err := malgo.InitContext(
		[]malgo.Backend{
			malgo.BackendDsound,
		},
		malgo.ContextConfig{
			CoreAudio: malgo.CoreAudioConfig{
				SessionCategory: malgo.IOSSessionCategoryAmbient,
			},
		},
		func(message string) {
			notify.Debug("Audio Session: %s", strings.Split(message, "\n")[0])
		},
	)
	if err != nil {
		return nil, err
	}

	in, err := input.New(ctx, i)
	if err != nil {
		return nil, err
	}

	out, err := output.New(ctx, o)
	if err != nil {
		return nil, err
	}

	s := &Session{
		In:  in,
		Out: out,

		buffer: bytes.NewBuffer(make([]byte, 0)),

		errorq: make(chan error),
		waitq:  make(chan bool),

		closed:  true,
		context: ctx,
	}

	return s, nil
}

func (s *Session) Inputs() []*input.Device {
	return input.Devices(s.context)
}

func (s *Session) Outputs() []*output.Device {
	return output.Devices(s.context)
}

func (s *Session) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true

	defer notify.System("%s (Closed)", s.String())

	for err := s.Error(); err != nil && err != SessionClosed; err = s.Error() {
		notify.Error("Audio session close error (%v)", err)
	}

	s.In.Close()
	s.Out.Close()

	return nil
}

func (s *Session) Error() error {
	select {
	case err := <-s.errorq:
		return err
	default:
		if s.closed {
			return SessionClosed
		}
		return nil
	}
}

func (s *Session) Input(name string) error {
	s.In.Close()
	s.Out.Close()

	in, err := input.New(s.context, name)
	if err != nil {
		return err
	}
	s.In = in

	return nil
}

func (s *Session) Output(name string) error {
	s.In.Close()
	s.Out.Close()

	p, err := output.New(s.context, name)
	if err != nil {
		return err
	}
	s.Out = p

	return nil
}

func (s *Session) Start() error {
	if s.In.IsDisabled() || s.Out.IsDisabled() {
		return nil
	}

	defer notify.System("%s (Started)", s.String())

	s.In.Start(s.context.Context, s.buffer, s.errorq)
	s.Out.Start(s.context.Context, s.buffer, s.errorq)

	s.closed = false

	return nil
}

func (s *Session) String() string {
	return fmt.Sprintf("Audio Session: %s -> %s", s.In, s.Out)
}
