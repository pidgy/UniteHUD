package device

/*
#cgo LDFLAGS: -L. -lstdc++ -lstrmiids -lole32 -loleaut32
#cgo CXXFLAGS: -std=c++14 -I.

#include "device.h"
*/
import "C"
// TODO: Add go style comments that reflect the purpose of each type, function, var, and const.

import (
	"fmt"
	"unsafe"
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
	return newDevice(index, C.DeviceTypeAudioCapture)
}

// AudioCaptureDeviceName returns the FriendlyName property of an audio input Device.
func AudioCaptureDeviceName(index int) (string, error) {
	return deviceName(index, C.DeviceTypeAudioCapture)
}

// AudioCaptureDevicePath returns the DevicePath property of an audio input Device.
func AudioCaptureDevicePath(index int) (string, error) {
	return path(index, C.DeviceTypeAudioCapture)
}

func (p Path) String() string {
	return p.raw
}

// VideoCaptureDevice returns a video input Device based on a given index.
func VideoCaptureDevice(index int) (*Device, error) {
	return newDevice(index, C.DeviceTypeVideoCapture)
}

// VideoCaptureDeviceName returns the FriendlyName property of a video input Device.
func VideoCaptureDeviceName(index int) (string, error) {
	return deviceName(index, C.DeviceTypeVideoCapture)
}

// VideoCaptureDeviceName returns the DevicePath property of a video input Device.
func VideoCaptureDevicePath(index int) (string, error) {
	return path(index, C.DeviceTypeVideoCapture)
}

func deviceName(index int, t C.DeviceType) (string, error) {
	name := C.DeviceName(C.int(index), t)
	if name == nil {
		return "", fmt.Errorf("failed to find name for device at index %d", index)
	}
	defer C.free(unsafe.Pointer(name))

	return C.GoString(name), nil
}

func newDevice(index int, t C.DeviceType) (*Device, error) {
	d := C.Device{}
	r := C.DeviceInit(&d, C.int(index), t)
	if r != 0 {
		return nil, fmt.Errorf("error code: %d", r)
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

func path(index int, t C.DeviceType) (string, error) {
	path := C.DevicePath(C.int(index), t)
	if path == nil {
		return "", fmt.Errorf("failed to find path for device at index %d", index)
	}
	defer C.free(unsafe.Pointer(path))

	return C.GoString(path), nil
}
