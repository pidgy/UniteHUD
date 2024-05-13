package win32

/*
#cgo LDFLAGS: -L. -lstdc++ -lstrmiids -lole32 -loleaut32
#cgo CXXFLAGS: -std=c++14 -I.

#include <stdlib.h>

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

// VideoCaptureDevice represents a connected device capable of capturing video.
type VideoCaptureDevice struct {
	Name        string
	Description string
	Path        string
	ID          string

	WaveInID int64

	Index int
}

// NewVideoCaptureDevice returns a VideoCaptureDevice based on an index or nil and an error.
func NewVideoCaptureDevice(index int) (*VideoCaptureDevice, error) {
	d := C.VideoCaptureDevice{}

	hr := C.GetVideoCaptureDevice(C.int(index), &d)
	if hr != 0 {
		return nil, errors.Errorf("failed to find device information: %d", hr)
	}
	defer free(unsafe.Pointer(d.name), unsafe.Pointer(d.path))

	name, err := utf16to8([]byte(C.GoBytes(unsafe.Pointer(d.name), d.namelen)))
	if err != nil {
		return nil, errors.Wrap(err, "video capture device name")
	}

	path, err := utf16to8(C.GoBytes(unsafe.Pointer(d.path), d.pathlen))
	if err != nil {
		return nil, errors.Wrap(err, "video capture device path")
	}

	return &VideoCaptureDevice{
		Name:     name,
		Path:     path,
		ID:       fmt.Sprintf("%s:%s", name, path),
		WaveInID: int64(d.waveinid),

		Index: index,
	}, nil
}

func free(p ...unsafe.Pointer) {
	for _, u := range p {
		C.free(u)
	}
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
