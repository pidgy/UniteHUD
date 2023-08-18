package audio

import (
	"bytes"
	"fmt"
	"io"

	"github.com/gen2brain/malgo"

	"github.com/pidgy/unitehud/audio/device"
	"github.com/pidgy/unitehud/audio/input"
	"github.com/pidgy/unitehud/audio/output"
	"github.com/pidgy/unitehud/notify"
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
	in, err := input.New(i)
	if err != nil {
		return nil, err
	}

	out, err := output.New(o)
	if err != nil {
		return nil, err
	}

	mctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(string) {})
	if err != nil {
		return nil, err
	}

	s := &Session{
		In:  in,
		Out: out,

		closed:  true,
		context: mctx,
	}

	return s, nil
}

func Devices() (in, out []string) {
	for _, d := range input.Devices() {
		in = append(in, d.Name())
	}
	for _, d := range output.Devices() {
		out = append(out, d.Name())
	}
	return
}

func Inputs() []*input.Device {
	return input.Devices()
}

func Outputs() []*output.Device {
	return output.Devices()
}

func (s *Session) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true

	s.In.Close()
	s.Out.Close()

	return device.Free(s.context)
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
	err := s.Close()
	if err != nil {
		return err
	}

	in, err := input.New(name)
	if err != nil {
		return err
	}
	s.In = in

	out, err := output.New(s.Out.Name())
	if err != nil {
		return err
	}
	s.Out = out

	return s.Start()
}

func (s *Session) Output(name string) error {
	err := s.Close()
	if err != nil {
		return err
	}

	p, err := output.New(name)
	if err != nil {
		return err
	}
	s.Out = p

	in, err := input.New(s.In.Name())
	if err != nil {
		return err
	}
	s.In = in

	return s.Start()
}

func (s *Session) Start() error {
	defer notify.System("%s", s)

	if s.In.IsDisabled() || s.Out.IsDisabled() {
		return nil
	}

	mctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(string) {})
	if err != nil {
		return err
	}

	s.waitq = make(chan bool)

	s.context = mctx
	s.closed = false
	s.buffer = bytes.NewBuffer(make([]byte, 0))

	go s.In.Start(s.context.Context, s.buffer, s.errorq, s.waitq)
	go s.Out.Start(s.context.Context, s.buffer, s.errorq, s.waitq)

	return nil
}

func (s *Session) String() string {
	return fmt.Sprintf("Audio Session: %s -> %s", s.In, s.Out)
}
