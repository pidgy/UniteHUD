package device

/*
#cgo LDFLAGS: -L. -lstdc++ -lstrmiids -lole32 -loleaut32
#cgo CXXFLAGS: -std=c++14 -I.

#include "device.h"
*/
import "C"
import (
	"fmt"
	"strings"
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

// NewAudioCaptureDevice returns an audio input Device based on a given index.
func NewAudioCaptureDevice(index int) (*Device, error) {
	return device(index, C.DeviceTypeAudioCapture)
}

// NewAudioVideoCaptureDevice returns an audio/video input Device based on a given video index.
func NewAudioVideoCaptureDevice(index int) (a, v *Device, err error) {
	v, err = device(index, C.DeviceTypeVideoCapture)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "video capture device %d", index)
	}
	if !v.HasPath() {
		return nil, nil, errors.Errorf("%s has an invalid path", v.Name)
	}

	for i := 0; i < 100; i++ {
		a, err = device(i, C.DeviceTypeAudioCapture)
		if err != nil {
			continue
		}
		if a.Path != v.Path {
			continue
		}

		println("here?", a == nil)
		return a, v, nil
	}

	return nil, nil, errors.Errorf("%s: failed to find audio capture device path", v.Name)
}

// NewVideoCaptureDevice returns a video input Device based on a given index.
func NewVideoCaptureDevice(index int) (*Device, error) {
	return device(index, C.DeviceTypeVideoCapture)
}

// AudioCaptureDeviceName returns the FriendlyName property of an audio input Device.
func AudioCaptureDeviceName(index int) (string, error) {
	return name(index, C.DeviceTypeAudioCapture)
}

// AudioCaptureDevicePath returns the DevicePath property of an audio input Device.
func AudioCaptureDevicePath(index int) (string, error) {
	return path(index, C.DeviceTypeAudioCapture)
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
		return nil, errors.Errorf("%d: failed to find device %d", r, index)
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
		return "", errors.Errorf("%d: failed to find device name", index)
	}
	defer C.free(unsafe.Pointer(name))

	return C.GoString(name), nil
}

func path(index int, t C.DeviceType) (string, error) {
	path := C.DevicePath(C.int(index), t)
	if path == nil {
		return "", errors.Errorf("%d: failed to find device path", index)
	}
	defer C.free(unsafe.Pointer(path))

	return C.GoString(path), nil
}

func compare(stringOne, stringTwo string) float32 {
	stringOne = strings.Replace(stringOne, " ", "", -1)
	stringTwo = strings.Replace(stringTwo, " ", "", -1)

	// if both are empty strings
	if len(stringOne) == 0 && len(stringTwo) == 0 {
		return 1
	}

	// if only one is empty string
	if len(stringOne) == 0 || len(stringTwo) == 0 {
		return 0
	}

	// identical
	if stringOne == stringTwo {
		return 1
	}

	// both are 1-letter strings
	if len(stringOne) == 1 && len(stringTwo) == 1 {
		return 0
	}

	// if either is a 1-letter string
	if len(stringOne) < 2 || len(stringTwo) < 2 {
		return 0
	}

	firstBigrams := make(map[string]int)
	for i := 0; i < len(stringOne)-1; i++ {
		a := fmt.Sprintf("%c", stringOne[i])
		b := fmt.Sprintf("%c", stringOne[i+1])

		bigram := a + b

		var count int

		if value, ok := firstBigrams[bigram]; ok {
			count = value + 1
		} else {
			count = 1
		}

		firstBigrams[bigram] = count
	}

	var intersectionSize float32
	intersectionSize = 0

	for i := 0; i < len(stringTwo)-1; i++ {
		a := fmt.Sprintf("%c", stringTwo[i])
		b := fmt.Sprintf("%c", stringTwo[i+1])

		bigram := a + b

		var count int

		if value, ok := firstBigrams[bigram]; ok {
			count = value
		} else {
			count = 0
		}

		if count > 0 {
			firstBigrams[bigram] = count - 1
			intersectionSize = intersectionSize + 1
		}
	}

	return (2.0 * intersectionSize) / (float32(len(stringOne)) + float32(len(stringTwo)) - 2)
}
