package win

/*
#cgo LDFLAGS: -L. -lstdc++ -lstrmiids -lole32 -loleaut32
#cgo CXXFLAGS: -std=c++14 -I.

#include "win.h"
*/
import "C"

import (
	"strings"
)

func VideoCaptureDeviceName(index int) string {
	len := C.int(0)
	v := C.GetVideoCaptureDeviceName(C.int(index), &len)
	n := C.GoStringN(v, len)
	return strings.ReplaceAll(n, "\x00", "")
}
