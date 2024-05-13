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

// AudioCaptureDevice represents a connected device capable of capturing audio.
type AudioCaptureDevice struct {
	Index int

	Name        string
	ID          string
	GUID        string
	Format      string
	Association string
	JackSubType string
	Description string
}

func (a *AudioCaptureDevice) String() string {
	return fmt.Sprintf("%s (%s)", a.Description, a.Name)
}

// NewAudioCaptureDevice returns an AudioCaptureDevice based on an index or nil and an error.
func NewAudioCaptureDevice(index int) (*AudioCaptureDevice, error) {
	d := C.AudioCaptureDevice{}
	defer free(&d)

	r := C.NewAudioCaptureDevice(&d, C.int(index))
	if r < 0 {
		return nil, fmt.Errorf("failed to find video capture device: %d", r)
	}

	return &AudioCaptureDevice{
		Index: index,

		Name:        C.GoString(d.name),
		ID:          C.GoString(d.id),
		GUID:        C.GoString(d.guid),
		Format:      C.GoString(d.format),
		Association: C.GoString(d.association),
		JackSubType: C.GoString(d.jacksubtype),
		Description: C.GoString(d.description),
	}, nil
}

func free(d *C.AudioCaptureDevice) {
	C.free(unsafe.Pointer(d.name))
	d.name = nil

	C.free(unsafe.Pointer(d.id))
	d.id = nil
}
