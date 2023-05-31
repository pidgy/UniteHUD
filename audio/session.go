package audio

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/gen2brain/malgo"
	"github.com/pidgy/unitehud/notify"
)

const (
	DeviceDefault  = "Default"
	DeviceDisabled = "Disabled"
)

var SessionClosed = fmt.Errorf("session closed")

type Session struct {
	*capture
	*playback

	buffer io.ReadWriter

	errorq chan error
	closeq chan bool
	waitq  chan bool

	closed  bool
	context *malgo.AllocatedContext
}

type Device interface {
	Name() string
	IsDefault() bool
}

func New(i, o string) (*Session, error) {
	capture, err := newCapture(i)
	if err != nil {
		return nil, err
	}

	playback, err := newPlayback(o)
	if err != nil {
		return nil, err
	}

	mctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(string) {})
	if err != nil {
		return nil, err
	}

	s := &Session{
		capture:  capture,
		playback: playback,

		buffer: bytes.NewBuffer(make([]byte, 0)),

		errorq: make(chan error, 2),
		closeq: make(chan bool, 2),
		waitq:  make(chan bool),

		closed: true,

		context: mctx,
	}

	return s, nil
}

func DeviceNames() (capture, playback []string) {
	for _, d := range captureDevices() {
		capture = append(capture, d.name)
	}
	for _, d := range playbackDevices() {
		playback = append(playback, d.name)
	}
	return
}

func (s *Session) Close() error {
	if s.closed {
		return nil
	}

	s.closed = true

	go func() {
		for range []Device{s.capture, s.playback} {
			s.closeq <- true
		}
	}()

	return free(s.context)
}

func (s *Session) Error() error {
	select {
	case err := <-s.errorq:
		return err
	default:
		if s.closed {
			return SessionClosed
		}
	}
	return nil
}

func (s *Session) Restart() error {
	if s.capture.name == DeviceDisabled || s.playback.name == DeviceDisabled {
		return nil
	}

	err := s.Close()
	if err != nil {
		return err
	}

	mctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(string) {})
	if err != nil {
		return err
	}

	s.waitq = make(chan bool)
	s.context = mctx
	s.closed = false

	notify.System("Starting audio session %s -> %s", s.capture.name, s.playback.name)

	go s.capture.start(s.context.Context, s.buffer, s.errorq, s.closeq, s.waitq)
	go s.playback.start(s.context.Context, s.buffer, s.errorq, s.closeq, s.waitq)

	return nil
}

func (s *Session) SetCapture(name string) error {
	if name == DeviceDisabled {
		return s.Close()
	}
	defer s.Restart()

	c, err := newCapture(name)
	if err != nil {
		return err
	}
	s.capture = c

	return nil
}

func (s *Session) SetPlayback(name string) error {
	if name == DeviceDisabled {
		return s.Close()
	}
	defer s.Restart()

	p, err := newPlayback(name)
	if err != nil {
		return err
	}
	s.playback = p

	return nil
}

func isDevice(d Device, name string) bool {
	if name == DeviceDefault {
		return d.IsDefault()
	}
	return strings.Contains(d.Name(), name)
}

func free(ctx *malgo.AllocatedContext) error {
	err := ctx.Uninit()
	if err != nil {
		return err
	}

	ctx.Free()

	return nil
}
