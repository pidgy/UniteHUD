package device

/*
#cgo LDFLAGS: -L. -lstdc++ -lstrmiids -lole32 -loleaut32
#cgo CXXFLAGS: -std=c++14 -I.

#include "device.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/pkg/errors"
)

// Device represents a Windows-compatible device.
type Device struct {
	Index int

	Name        string
	Path        string
	Description string

	WaveInID int32
}

// NewAudioCaptureDevice returns an audio input Device based on a given index.
func NewAudioCaptureDevice(index int) (*Device, error) {
	return device(index, C.DeviceTypeAudioCapture)
}

// NewVideoCaptureDevice returns a video input Device based on a given index.
func NewVideoCaptureDevice(index int) (*Device, error) {
	return device(index, C.DeviceTypeVideoCapture)
}

// AudioCaptureDeviceName returns the FriendlyName property of an audio input Device.
func AudioCaptureDeviceName(index int) (string, error) {
	return name(index, C.DeviceTypeAudioCapture)
}

// VideoCaptureDeviceName returns the FriendlyName property of a video input Device.
func VideoCaptureDeviceName(index int) (string, error) {
	return name(index, C.DeviceTypeVideoCapture)
}

// ID returns the Device's ID, a concatenation the Name and Path properties respectively.
func (v *Device) ID() string {
	return fmt.Sprintf("%s:%s", v.Name, v.Path)
}

func device(index int, t C.DeviceType) (*Device, error) {
	d := C.Device{}
	r := C.DeviceInit(&d, C.int(index), t)
	if r != 0 {
		return nil, errors.Errorf("failed to find device information: %d", r)
	}
	defer C.DeviceFree(&d)

	return &Device{
		Index: index,

		Name:        C.GoString(d.Name),
		Path:        C.GoString(d.Path),
		Description: C.GoString(d.Description),

		WaveInID: int32(d.WaveInID),
	}, nil
}

func name(index int, t C.DeviceType) (string, error) {
	name := C.DeviceName(C.int(index), t)
	if name == nil {
		return "", fmt.Errorf("%d: failed to find device name", index)
	}
	defer C.free(unsafe.Pointer(name))

	return C.GoString(name), nil
}
