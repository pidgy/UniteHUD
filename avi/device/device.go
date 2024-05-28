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
	Description string

	Path Path

	WaveInID int32
}

// AudioCaptureDevice returns an audio input Device based on a given index.
func AudioCaptureDevice(index int) (*Device, error) {
	return device(index, C.DeviceTypeAudioCapture)
}

// AudioCaptureDeviceName returns the FriendlyName property of an audio input Device.
func AudioCaptureDeviceName(index int) (string, error) {
	return name(index, C.DeviceTypeAudioCapture)
}

// AudioCaptureDevicePath returns the DevicePath property of an audio input Device.
func AudioCaptureDevicePath(index int) (string, error) {
	return path(index, C.DeviceTypeAudioCapture)
}

// VideoCaptureDevice returns a video input Device based on a given index.
func VideoCaptureDevice(index int) (*Device, error) {
	return device(index, C.DeviceTypeVideoCapture)
}

// VideoCaptureDeviceName returns the FriendlyName property of a video input Device.
func VideoCaptureDeviceName(index int) (string, error) {
	return name(index, C.DeviceTypeVideoCapture)
}

// VideoCaptureDeviceName returns the DevicePath property of a video input Device.
func VideoCaptureDevicePath(index int) (string, error) {
	return path(index, C.DeviceTypeVideoCapture)
}

func (p *Path) String() string {
	return p.raw
}

// HasPath returns whether or not a valid DevicePath property was associated with d.
func (d *Device) HasPath() bool {
	return d.Path.raw != ""
}

// ID returns the Device's ID, a concatenation the Name and Path properties respectively.
func (d *Device) ID() string {
	return fmt.Sprintf("%s:%s", d.Name, d.Path)
}

func device(index int, t C.DeviceType) (*Device, error) {
	d := C.Device{}
	r := C.DeviceInit(&d, C.int(index), t)
	if r != 0 {
		return nil, errors.Errorf("error code: %d", r)
	}
	defer C.DeviceFree(&d)

	return &Device{
		Index: index,

		Name:        C.GoString(d.Name),
		Path:        NewPath(C.GoString(d.Path)),
		Description: C.GoString(d.Description),

		WaveInID: int32(d.WaveInID),
	}, nil
}

func name(index int, t C.DeviceType) (string, error) {
	name := C.DeviceName(C.int(index), t)
	if name == nil {
		return "", errors.Errorf("empty value")
	}
	defer C.free(unsafe.Pointer(name))

	return C.GoString(name), nil
}

func path(index int, t C.DeviceType) (string, error) {
	path := C.DevicePath(C.int(index), t)
	if path == nil {
		return "", errors.Errorf("empty value")
	}
	defer C.free(unsafe.Pointer(path))

	return C.GoString(path), nil
}
