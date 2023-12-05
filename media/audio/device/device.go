package device

import (
	"io"

	"github.com/gen2brain/malgo"
)

type Type string

const (
	Input  Type = "input"
	Output Type = "output"
)

const (
	Default  = "Default"
	Disabled = "Disabled"
)

type Device interface {
	Active() bool
	Close()
	IsDefault() bool
	IsDisabled() bool
	Name() string
	Start(malgo.Context, io.ReadWriter) error
	Type() Type
}

func Is(d Device, name string) bool {
	if name == Default {
		return d.IsDefault()
	}
	return d.Name() == name
}

func Free(ctx *malgo.AllocatedContext) error {
	err := ctx.Uninit()
	if err != nil {
		return err
	}

	ctx.Free()

	return nil
}

func String(d Device) string {
	return d.Name()
}
