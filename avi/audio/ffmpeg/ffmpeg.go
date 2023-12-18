package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/gen2brain/malgo"
	"github.com/pidgy/unitehud/avi/audio/device"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pkg/errors"
)

type Device struct {
	name   string
	active bool

	cmd  *exec.Cmd
	errq chan error

	ctx    context.Context
	cancel context.CancelFunc
}

func New(name string) *Device {
	return &Device{name: name, errq: make(chan error, 1024)}
}

func (d *Device) Active() bool { return d.active }

func (d *Device) Close() {
	if d.cmd == nil {
		return
	}

	notify.System("Audio Input: Closing %s (ffmpeg)", d.name)

	d.cancel()
}

func (d *Device) Is(name string) bool { return device.Is(d, name) }
func (d *Device) IsDefault() bool     { return false }
func (d *Device) IsDisabled() bool    { return d == nil || d.name == device.Disabled }
func (d *Device) Name() string        { return d.name }

func (d *Device) Start(ctx malgo.Context, w io.ReadWriter) error {
	if d.Active() {
		return errors.Wrap(errors.New("already active"), d.name)
	}

	// d.cmd = exec.Command(
	// 	"ffmpeg",
	// 	strings.Split(
	// 		fmt.Sprintf(`-f dshow -i audio="%s" -vn -sn -f mp3 pipe:1`, d.name),
	// 		" ",
	// 	)...,
	// )

	d.ctx, d.cancel = context.WithCancel(context.Background())

	d.cmd = exec.CommandContext(
		d.ctx,
		"ffmpeg",
		strings.Split(
			fmt.Sprintf(`-f dshow -i audio="%s" -vn -sn -f pcm_s32le pipe:1 -v quiet`, d.name),
			" ",
		)...,
	)
	// d.cmd.Stdout = w
	d.cmd.Stdout = bytes.NewBuffer(make([]byte, 4096))

	d.errq <- errors.Wrapf(d.cmd.Start(), "run: %s", d)

	d.active = true

	defer notify.Debug("Audio Input: Started %s", d)

	go func() {
		buf := make([]byte, 2646)

		for d.active {
			_, err := d.cmd.Stdout.Write(buf)
			if err != nil {
				d.errq <- err
				return
			}

			_, err = w.Write(buf)
			if err != nil {
				d.errq <- err
				return
			}
		}
	}()

	d.errq <- errors.Wrapf(d.cmd.Wait(), "wait: %s", d)

	close(d.errq)

	time.Sleep(time.Second)

	for {
		err, ok := <-d.errq
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
}

func (d *Device) String() string { return fmt.Sprintf("%s (ffmpeg)", device.String(d)) }

func (d *Device) Type() device.Type { return device.Input }
