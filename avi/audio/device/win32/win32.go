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
	Index int

	Name        string
	ID          string
	GUID        string
	Format      string
	Association string
	JackSubType string
	Description string

	Flow Flow
}

// NewAudioCaptureDevice returns an AudioDevice capable of capturing audio, based on an index.
func NewAudioCaptureDevice(index int) (*AudioDevice, error) {
	d := C.AudioDevice{}
	defer free(&d)

	r := C.NewAudioCaptureDevice(&d, C.int(index))
	if r < 0 {
		return nil, fmt.Errorf("failed to find video capture device: %d", r)
	}

	return &AudioDevice{
		Index: index,

		Name:        C.GoString(d.name),
		ID:          C.GoString(d.id),
		GUID:        C.GoString(d.guid),
		Format:      C.GoString(d.format),
		Association: C.GoString(d.association),
		JackSubType: C.GoString(d.jacksubtype),
		Description: C.GoString(d.description),

		Flow: FlowCapture,
	}, nil
}

// NewAudioCaptureRenderDevice returns an AudioDevice capable of both capturing and rendering audio, based on an index.
func NewAudioCaptureRenderDevice(index int) (*AudioDevice, error) {
	d := C.AudioDevice{}
	defer free(&d)

	r := C.NewAudioCaptureRenderDevice(&d, C.int(index))
	if r < 0 {
		return nil, fmt.Errorf("failed to find video capture device: %d", r)
	}

	return &AudioDevice{
		Index: index,

		Name:        C.GoString(d.name),
		ID:          C.GoString(d.id),
		GUID:        C.GoString(d.guid),
		Format:      C.GoString(d.format),
		Association: C.GoString(d.association),
		JackSubType: C.GoString(d.jacksubtype),
		Description: C.GoString(d.description),

		Flow: FlowCaptureRender,
	}, nil
}

// NewAudioRenderDevice returns an AudioDevice capable of rendering audio, based on an index.
func NewAudioRenderDevice(index int) (*AudioDevice, error) {
	d := C.AudioDevice{}
	defer free(&d)

	r := C.NewAudioRenderDevice(&d, C.int(index))
	if r < 0 {
		return nil, fmt.Errorf("failed to find video capture device: %d", r)
	}

	return &AudioDevice{
		Index: index,

		Name:        C.GoString(d.name),
		ID:          C.GoString(d.id),
		GUID:        C.GoString(d.guid),
		Format:      C.GoString(d.format),
		Association: C.GoString(d.association),
		JackSubType: C.GoString(d.jacksubtype),
		Description: C.GoString(d.description),

		Flow: FlowRender,
	}, nil
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
	C.free(unsafe.Pointer(d.name))
	d.name = nil
	C.free(unsafe.Pointer(d.id))
	d.id = nil
	C.free(unsafe.Pointer(d.guid))
	d.guid = nil
	C.free(unsafe.Pointer(d.format))
	d.format = nil
	C.free(unsafe.Pointer(d.association))
	d.association = nil
	C.free(unsafe.Pointer(d.jacksubtype))
	d.jacksubtype = nil
	C.free(unsafe.Pointer(d.description))
	d.description = nil
}
