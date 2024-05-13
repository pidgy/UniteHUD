package win32

/*
#cgo LDFLAGS: -L. -lstdc++ -lstrmiids -lole32 -loleaut32 -lMfplat -lMf
#cgo CXXFLAGS: -std=c++14 -I.

#include "win32.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type Flow int

const (
	FlowCapture Flow = iota
	FlowRender
	FlowCaptureRender
)

// AudioDevice represents a connected device capable of capturing audio.
type AudioDevice struct {
	Flow

	Name        string
	ID          string
	GUID        string
	Format      string
	Association string
	JackSubType string
	Description string

	Index int
}

// NewAudioCaptureDevice returns an AudioDevice capable of capturing audio, based on an index.
func NewAudioCaptureDevice(index int) (*AudioDevice, error) {
	return newAudioDevice(index, FlowCapture, func(ad *C.AudioDevice, i C.int) C.int { return C.NewAudioCaptureDevice(ad, i) })
}

// NewAudioCaptureRenderDevice returns an AudioDevice capable of both capturing and rendering audio, based on an index.
func NewAudioCaptureRenderDevice(index int) (*AudioDevice, error) {
	return newAudioDevice(index, FlowCaptureRender, func(ad *C.AudioDevice, i C.int) C.int { return C.NewAudioCaptureRenderDevice(ad, i) })
}

// NewAudioRenderDevice returns an AudioDevice capable of rendering audio, based on an index.
func NewAudioRenderDevice(index int) (*AudioDevice, error) {
	return newAudioDevice(index, FlowRender, func(ad *C.AudioDevice, i C.int) C.int { return C.NewAudioCaptureRenderDevice(ad, i) })
}

func (a *AudioDevice) String() string {
	return fmt.Sprintf("%s (%s)", a.Description, a.Name)
}

func (f Flow) String() string {
	switch f {
	case FlowCapture:
		return "Capture"
	case FlowRender:
		return "Render"
	case FlowCaptureRender:
		return "Capture/Render"
	default:
		return "Invalid"
	}
}

func free(d *C.AudioDevice) {
	fchar := func(c *C.char) *C.char {
		C.free(unsafe.Pointer(c))
		return nil
	}
	d.name = fchar(d.name)
	d.id = fchar(d.id)
	d.guid = fchar(d.guid)
	d.format = fchar(d.format)
	d.association = fchar(d.association)
	d.jacksubtype = fchar(d.jacksubtype)
	d.description = fchar(d.description)
}

func newAudioDevice(index int, f Flow, fn func(*C.AudioDevice, C.int) C.int) (*AudioDevice, error) {
	d := C.AudioDevice{}
	defer free(&d)

	r := fn(&d, C.int(index))
	if r != 0 {
		return nil, fmt.Errorf("failed to find video capture device: %d", r)
	}

	return &AudioDevice{
		Flow: f,

		Name:        C.GoString(d.name),
		ID:          C.GoString(d.id),
		GUID:        C.GoString(d.guid),
		Format:      C.GoString(d.format),
		Association: C.GoString(d.association),
		JackSubType: C.GoString(d.jacksubtype),
		Description: C.GoString(d.description),

		Index: index,
	}, nil
}
