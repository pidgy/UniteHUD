package win32

/*
#cgo LDFLAGS: -L. -lstdc++ -lstrmiids -lole32 -loleaut32
#cgo CXXFLAGS: -std=c++14 -I.

#include "win32.h"
*/
import "C"
import (
	"bytes"
	"fmt"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"

	"github.com/pkg/errors"
)

type VideoCaptureDevice struct {
	Name     string
	Path     string
	ID       string
	WaveInID string
}

func NewVideoCaptureDevice(index int) (*VideoCaptureDevice, error) {
	d := C.VideoCaptureDevice{}

	hr := C.GetVideoCaptureDevice(C.int(index), &d)
	if hr != 0 {
		return nil, errors.Wrap(errors.Errorf("failed to find device information: %d", hr), fmt.Sprintf("video capture device %d", index))
	}

	defer C.free(unsafe.Pointer(d.name))
	defer C.free(unsafe.Pointer(d.path))
	defer C.free(unsafe.Pointer(d.waveinid))

	name, err := utf16to8([]byte(C.GoBytes(unsafe.Pointer(d.name), d.namelen)))
	if err != nil {
		return nil, errors.Wrap(err, "video capture device name")
	}

	path, err := utf16to8(C.GoBytes(unsafe.Pointer(d.path), d.pathlen))
	if err != nil {
		return nil, errors.Wrap(err, "video capture device path")
	}

	wave, err := utf16to8(C.GoBytes(unsafe.Pointer(d.waveinid), d.waveinidlen))
	if err != nil {
		return nil, errors.Wrap(err, "video capture device wave in id")
	}

	return &VideoCaptureDevice{
		Name:     name,
		Path:     path,
		ID:       fmt.Sprintf("%s:%s", name, path),
		WaveInID: wave,
	}, nil
}

// VideoCaptureDeviceName will fetch the L"FriendlyName" property of a DirectShow device.
func VideoCaptureDeviceName(index int) (string, error) {
	len := C.int(0)
	v := C.GetVideoCaptureDeviceName(C.int(index), &len)
	src := C.GoBytes(unsafe.Pointer(v), len)
	dst := make([]byte, len)
	copy(dst, src)
	u, err := utf16to8(dst)
	return u, errors.Wrap(err, "dshow device name")
}

// VideoCaptureDevicePath will fetch the L"DevicePath" property of a DirectShow device.
func VideoCaptureDevicePath(index int) (string, error) {
	len := C.int(0)
	v := C.GetVideoCaptureDevicePath(C.int(index), &len)
	src := C.GoBytes(unsafe.Pointer(v), len)
	dst := make([]byte, len)
	copy(dst, src)
	u, err := utf16to8(dst)
	return u, errors.Wrap(err, "dshow device path")
}

// VideoCaptureDeviceDescription will fetch the L"Description" property of a DirectShow device.
func VideoCaptureDeviceDescription(index int) (string, error) {
	len := C.int(0)
	v := C.GetVideoCaptureDeviceDescription(C.int(index), &len)
	u, err := utf16to8(C.GoBytes(unsafe.Pointer(v), len))
	return u, errors.Wrap(err, "dshow device description")
}

// VideoCaptureDeviceWaveInID will fetch the L"WaveInId" property of a DirectShow device.
func VideoCaptureDeviceWaveInID(index int) (string, error) {
	len := C.int(0)
	v := C.GetVideoCaptureDeviceWaveInID(C.int(index), &len)
	u, err := utf16to8(C.GoBytes(unsafe.Pointer(v), len))
	return u, errors.Wrap(err, "dshow device wave in id")
}

func utf16to8(b []byte) (string, error) {
	if len(b)%2 != 0 {
		return "", errors.Wrap(fmt.Errorf("must have even length byte slice"), "utf16to8")
	}

	u16s := make([]uint16, 1)

	ret := &bytes.Buffer{}

	b8buf := make([]byte, 4)

	lb := len(b)
	for i := 0; i < lb; i += 2 {
		u16s[0] = uint16(b[i]) + (uint16(b[i+1]) << 8)
		r := utf16.Decode(u16s)
		n := utf8.EncodeRune(b8buf, r[0])
		ret.Write(b8buf[:n])
	}

	return ret.String(), nil
}
