package win32

/*
#cgo LDFLAGS: -L. -lstdc++ -lstrmiids -lole32 -loleaut32
#cgo CXXFLAGS: -std=c++14 -I.

#include "win32.h"
*/
import "C"

import (
	"strings"
)

// Get the friendly device name of a Video Capture Device using opencv-style indexes 0-9.
func VideoCaptureDeviceName(index int) string {
	len := C.int(0)
	v := C.GetVideoCaptureDeviceName(C.int(index), &len)
	n := C.GoStringN(v, len)
	return strings.ReplaceAll(n, "\x00", "")
}

func VideoCaptureDevicePath(index int) string {
	len := C.int(0)
	v := C.GetVideoCaptureDevicePath(C.int(index), &len)
	n := C.GoStringN(v, len)
	return strings.ReplaceAll(n, "\x00", "")
}

func VideoCaptureDeviceDescription(index int) string {
	len := C.int(0)
	v := C.GetVideoCaptureDeviceDescription(C.int(index), &len)
	n := C.GoStringN(v, len)
	return strings.ReplaceAll(n, "\x00", "")
}

func VideoCaptureDeviceWaveInID(index int) string {
	len := C.int(0)
	v := C.GetVideoCaptureDeviceWaveInID(C.int(index), &len)
	n := C.GoStringN(v, len)
	return strings.ReplaceAll(n, "\x00", "")
}
