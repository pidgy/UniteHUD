package device

import (
	"io"

	"github.com/gen2brain/malgo"
)

// Type identifies whether an audio device is input or output.
type Type string

const (
	// Input identifies a capture device.
	Input  Type = "input"
	// Output identifies a playback device.
	Output Type = "output"
)

const (
	// Default is the label for the system default device.
	Default  = "Default"
	// Disabled is the label for a disabled device.
	Disabled = "Disabled"
)

// Device describes shared behavior for audio devices.
type Device interface {
	Active() bool
	Close()
	IsDefault() bool
	IsDisabled() bool
	Name() string
	Start(malgo.Context, io.ReadWriter) error
	Type() Type
	String() string
}

// Free releases the audio context.
func Free(ctx *malgo.AllocatedContext) error {
	err := ctx.Uninit()
	if err != nil {
		return err
	}

	ctx.Free()

	return nil
}

// Is reports whether the device matches the given name or default.
func Is(d Device, name string) bool {
	if name == Default {
		return d.IsDefault()
	}
	return d.Name() == name
}

// String formats the device name safely.
func String(d Device) string {
	if d == nil {
		return "Invalid Device"
	}
	return d.Name()
}
