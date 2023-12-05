package monitor

import (
	"fmt"
	"image"
	"reflect"
	"sync"
	"unsafe"

	"github.com/kbinani/screenshot"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/media/video/wapi"
)

var (
	MainResolution = image.Rect(0, 0, 1920, 1080)
	Sources        = []string{}

	displays = map[string]int{}
	bounds   = map[string]image.Rectangle{}

	mutex = &sync.RWMutex{}
)

func Bounds() image.Rectangle {
	mutex.RLock()
	defer mutex.RUnlock()
	b := bounds[config.Current.Video.Capture.Window.Name]
	return b
}

func BoundsOf(d string) image.Rectangle {
	mutex.RLock()
	defer mutex.RUnlock()
	b := bounds[d]
	return b
}

func Capture() (*image.RGBA, error) {
	return CaptureRect(dims())
}

func CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
	b := dims()

	rect.Min.X = b.Min.X + rect.Min.X
	rect.Max.X = b.Min.X + rect.Max.X

	rect.Min.Y = b.Min.Y + rect.Min.Y
	rect.Max.Y = b.Min.Y + rect.Max.Y

	src := getDC(0)
	if src == 0 {
		return nil, fmt.Errorf("Failed to find primary display (%d)", getLastError())
	}
	defer releaseDC(0, src)

	dst := createCompatibleDC(src)
	if dst == 0 {
		return nil, fmt.Errorf("Could not Create Compatible DC (%d)", getLastError())
	}
	defer wapi.DeleteDC.Call(dst) // nolint

	x, y := rect.Dx(), rect.Dy()

	bt := wapi.BitmapInfo{}
	bt.BmiHeader = wapi.BitmapInfoHeader{
		BiSize:        uint32(reflect.TypeOf(bt.BmiHeader).Size()),
		BiWidth:       int32(x),
		BiHeight:      int32(-y),
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: wapi.BitmapInfoHeaderCompression.RGB,
	}

	ptr := uintptr(0)

	m, _, _ := wapi.CreateDIBSection.Call(uintptr(dst), uintptr(unsafe.Pointer(&bt)), uintptr(wapi.CreateDIBSectionUsage.RGBColors), uintptr(unsafe.Pointer(&ptr)), 0, 0)
	if m == 0 {
		return nil, fmt.Errorf("Could not Create DIB Section err:%d.\n", getLastError())
	}
	if m == wapi.CreateDIBSectionError.InvalidParameter {
		return nil, fmt.Errorf("One or more of the input parameters is invalid while calling CreateDIBSection.\n")
	}
	defer deleteObject(m)

	obj := selectObject(dst, m)
	if obj == 0 {
		return nil, fmt.Errorf("error occurred and the selected object is not a region err:%d.\n", getLastError())
	}
	if obj == 0xffffffff { //GDI_ERROR
		return nil, fmt.Errorf("GDI_ERROR while calling SelectObject err:%d.\n", getLastError())
	}
	defer deleteObject(obj)

	//if !bitBlt(mHDC, 0, 0, x, y, hdc, rect.Min.X, rect.Min.Y) {
	//	return nil, fmt.Errorf("BitBlt failed err:%d.\n", getLastError())
	//}

	width := rect.Dx()
	height := rect.Dy()

	var ret uintptr
	switch config.Current.Scale {
	case 1:
		ret, _, _ = wapi.BitBlt.Call(
			uintptr(dst),
			0,
			0,
			uintptr(width),
			uintptr(height),
			uintptr(src),
			uintptr(rect.Min.X),
			uintptr(rect.Min.Y),
			wapi.BitBltRasterOperations.CaptureBLT|wapi.BitBltRasterOperations.SrcCopy,
		)
	default: // Scaled.
		scaledW := int(float64(width) * config.Current.Scale)
		scaledH := int(float64(height) * config.Current.Scale)

		ret, _, _ = wapi.StretchBlt.Call(
			uintptr(dst),
			0,
			0,
			uintptr(scaledW),
			uintptr(scaledH),
			uintptr(src),
			uintptr(rect.Min.X),
			uintptr(rect.Min.Y),
			uintptr(width),
			uintptr(height),
			wapi.BitBltRasterOperations.CaptureBLT|wapi.BitBltRasterOperations.SrcCopy,
		)
	}
	if ret == 0 {
		notify.Error("🖥️  Failed to capture \"%s\"", config.Current.Video.Capture.Window.Name)
		return nil, fmt.Errorf("bitblt returned: %d", ret)
	}

	var slice []byte
	hdrp := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	hdrp.Data = uintptr(ptr)
	hdrp.Len = x * y * 4
	hdrp.Cap = x * y * 4

	imageBytes := make([]byte, len(slice))

	for i := 0; i < len(imageBytes); i += 4 {
		imageBytes[i], imageBytes[i+2], imageBytes[i+1], imageBytes[i+3] = slice[i+2], slice[i], slice[i+1], slice[i+3]
	}

	return &image.RGBA{
		Pix:    imageBytes,
		Stride: 4 * x,
		Rect:   image.Rect(0, 0, x, y),
	}, nil
}

func IsDisplay() bool {
	mutex.RLock()
	defer mutex.RUnlock()

	_, ok := displays[config.Current.Video.Capture.Window.Name]
	return ok
}

func Open() {
	sourcesTmp := []string{}
	displaysTmp := map[string]int{}
	boundsTmp := map[string]image.Rectangle{}

	leftDisplays := 0
	rightDisplays := 0
	topDisplays := 0
	bottomDisplays := 0

	m := MainResolution

	for i := 0; i < screenshot.NumActiveDisplays(); i++ {
		name := ""

		r := screenshot.GetDisplayBounds(i)
		switch {
		case r.Eq(m):
			name = config.MainDisplay
		case r.Min.X < m.Min.X:
			leftDisplays++
			name = display("Left Display", leftDisplays)
		case r.Min.X > m.Min.X:
			rightDisplays++
			name = display("Right Display", rightDisplays)
		case r.Min.Y < m.Min.Y:
			topDisplays++
			name = display("Top Display", topDisplays)
		case r.Min.Y > m.Min.Y:
			bottomDisplays++
			name = display("Bottom Display", bottomDisplays)
		default:
			notify.Error("🖥️  Failed to locate display #%d [%s] relative to %s [%s]", i, r, config.MainDisplay, m)
			continue
		}

		displaysTmp[name] = i
		boundsTmp[name] = r
		sourcesTmp = append(sourcesTmp, name)
	}
	set(sourcesTmp, displaysTmp, boundsTmp)
}

func createCompatibleDC(hdc uintptr) uintptr {
	ret, _, _ := wapi.CreateCompatibleDC.Call(uintptr(hdc))
	return ret
}

func deleteObject(hObject uintptr) bool {
	ret, _, _ := wapi.DeleteObject.Call(hObject)
	return ret != 0
}

func dims() image.Rectangle {
	mutex.RLock()
	defer mutex.RUnlock()

	b := bounds[config.Current.Video.Capture.Window.Name]
	return b
}

func display(name string, count int) string {
	if count <= 1 {
		return name
	}
	return fmt.Sprintf("%s %d", name, count)
}

func getDC(hwnd uintptr) uintptr {
	ret, _, _ := wapi.GetDC.Call(uintptr(hwnd))
	return ret
}

func getLastError() uint32 {
	ret, _, _ := wapi.GetLastError.Call()
	return uint32(ret)
}

func releaseDC(hwnd uintptr, hdc uintptr) bool {
	ret, _, _ := wapi.ReleaseDC.Call(uintptr(hwnd), uintptr(hdc))
	return ret != 0
}

func selectObject(hdc, hgdiobj uintptr) uintptr {
	ret, _, _ := wapi.SelectObject.Call(uintptr(hdc), uintptr(hgdiobj))
	return ret
}

func set(s []string, d map[string]int, b map[string]image.Rectangle) {
	mutex.Lock()
	defer mutex.Unlock()

	Sources = s
	displays = d
	bounds = b
}
