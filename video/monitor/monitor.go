package monitor

import (
	"fmt"
	"image"
	"reflect"
	"sync"
	"unsafe"

	"github.com/kbinani/screenshot"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/video/wapi"
)

var (
	Sources = []string{}

	displays = map[string]int{}
	bounds   = map[string]image.Rectangle{}

	mainResolution = image.Rectangle{}

	mutex = &sync.RWMutex{}
)

func Bounds() image.Rectangle {
	mutex.RLock()
	defer mutex.RUnlock()
	b := bounds[config.Current.Window]
	return b
}

func BoundsOf(d string) image.Rectangle {
	mutex.RLock()
	defer mutex.RUnlock()
	b := bounds[d]
	return b
}

func Open() {
	sourcesTmp := []string{}
	displaysTmp := map[string]int{}
	boundsTmp := map[string]image.Rectangle{}

	leftDisplays := 0
	rightDisplays := 0
	topDisplays := 0
	bottomDisplays := 0

	m, err := mainDisplayRect()
	if err != nil {
		notify.SystemWarn("Failed to locate %s bounds", config.MainDisplay)
	}

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
			notify.Error("Failed to locate display #%d [%s] relative to %s [%s]", i, r, config.MainDisplay, m)
			continue
		}

		displaysTmp[name] = i
		boundsTmp[name] = r
		sourcesTmp = append(sourcesTmp, name)
	}
	set(sourcesTmp, displaysTmp, boundsTmp)
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
	defer wapi.DeleteDC.Call(uintptr(dst))

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

	ptr := unsafe.Pointer(uintptr(0))

	mhBmp := createDIBSection(dst, &bt, wapi.CreateDIBSectionUsage.RGBColors, &ptr, 0, 0)
	if mhBmp == 0 {
		return nil, fmt.Errorf("Could not Create DIB Section err:%d.\n", getLastError())
	}
	if mhBmp == wapi.CreateDIBSectionError.InvalidParameter {
		return nil, fmt.Errorf("One or more of the input parameters is invalid while calling CreateDIBSection.\n")
	}
	defer deleteObject(wapi.HGDIOBJ(mhBmp))

	obj := selectObject(dst, wapi.HGDIOBJ(mhBmp))
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
		notify.Error("Failed to capture \"%s\"", config.Current.Window)
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

	_, ok := displays[config.Current.Window]
	return ok
}

func MainResolution() image.Rectangle {
	if mainResolution.Max.Eq(image.Pt(0, 0)) {
		cx := uintptr(0)
		cy := uintptr(1)
		x, _, _ := wapi.GetSystemMetrics.Call(cx)
		y, _, _ := wapi.GetSystemMetrics.Call(cy)
		mainResolution = image.Rectangle{Max: image.Pt(int(x), int(y))}
	}

	return mainResolution
}

func bitBlt(dst wapi.HDC, dstx, dsty, dstw, dsth int, src wapi.HDC, srcx, srcy int) bool {

	var ret uintptr
	switch config.Current.Scale {
	case 1:
		ret, _, _ = wapi.BitBlt.Call(
			uintptr(dst),
			0,
			0,
			uintptr(dstw),
			uintptr(dsth),
			uintptr(src),
			uintptr(dstx),
			uintptr(dsty),
			wapi.BitBltRasterOperations.CaptureBLT|wapi.BitBltRasterOperations.SrcCopy,
		)
	default: // Scaled.
		scaledW := int(float64(dstw) * config.Current.Scale)
		scaledH := int(float64(dsth) * config.Current.Scale)

		ret, _, _ = wapi.StretchBlt.Call(
			uintptr(dst),
			0,
			0,
			uintptr(scaledW),
			uintptr(scaledH),
			uintptr(src),
			uintptr(dstx),
			uintptr(dsty),
			uintptr(srcx),
			uintptr(srcy),
			wapi.BitBltRasterOperations.CaptureBLT|wapi.BitBltRasterOperations.SrcCopy,
		)
	}
	return ret != 0
}

func createCompatibleDC(hdc wapi.HDC) wapi.HDC {
	ret, _, _ := wapi.CreateCompatibleDC.Call(
		uintptr(hdc))

	if ret == 0 {
		panic("Create compatible DC failed")
	}

	return wapi.HDC(ret)
}

func createDIBSection(hdc wapi.HDC, pbmi *wapi.BitmapInfo, iUsage uint, ppvBits *unsafe.Pointer, hSection wapi.Handle, dwOffset uint) wapi.HBITMAP {
	ret, _, _ := wapi.CreateDIBSection.Call(
		uintptr(hdc),
		uintptr(unsafe.Pointer(pbmi)),
		uintptr(iUsage),
		uintptr(unsafe.Pointer(ppvBits)),
		uintptr(hSection),
		uintptr(dwOffset))

	return wapi.HBITMAP(ret)
}

func deleteDC(hdc wapi.HDC) bool {
	ret, _, _ := wapi.DeleteDC.Call(uintptr(hdc))
	return ret != 0
}

func deleteObject(hObject wapi.HGDIOBJ) bool {
	ret, _, _ := wapi.DeleteObject.Call(uintptr(hObject))
	return ret != 0
}

func dims() image.Rectangle {
	mutex.RLock()
	defer mutex.RUnlock()

	b := bounds[config.Current.Window]
	return b
}

func display(name string, count int) string {
	if count <= 1 {
		return name
	}
	return fmt.Sprintf("%s %d", name, count)
}

func getDC(hwnd wapi.HWND) wapi.HDC {
	ret, _, _ := wapi.GetDC.Call(uintptr(hwnd))
	return wapi.HDC(ret)
}

func getLastError() uint32 {
	ret, _, _ := wapi.GetLastError.Call()
	return uint32(ret)
}

func releaseDC(hwnd wapi.HWND, hdc wapi.HDC) bool {
	ret, _, _ := wapi.ReleaseDC.Call(uintptr(hwnd), uintptr(hdc))
	return ret != 0
}

func mainDisplayRect() (image.Rectangle, error) {
	hdc := getDC(0)
	if hdc == 0 {
		return image.Rectangle{}, fmt.Errorf("Could not Get primary display err:%d\n", getLastError())
	}
	defer releaseDC(0, hdc)

	x, _, _ := wapi.GetDeviceCaps.Call(uintptr(hdc), uintptr(wapi.GetDeviceCapsIndex.HorzRes))
	y, _, _ := wapi.GetDeviceCaps.Call(uintptr(hdc), uintptr(wapi.GetDeviceCapsIndex.VertRes))

	return image.Rect(0, 0, int(x), int(y)), nil
}

func selectObject(hdc wapi.HDC, hgdiobj wapi.HGDIOBJ) wapi.HGDIOBJ {
	ret, _, _ := wapi.SelectObject.Call(
		uintptr(hdc),
		uintptr(hgdiobj))

	if ret == 0 {
		panic("SelectObject failed")
	}

	return wapi.HGDIOBJ(ret)
}

func set(s []string, d map[string]int, b map[string]image.Rectangle) {
	mutex.Lock()
	defer mutex.Unlock()

	Sources = s
	displays = d
	bounds = b
}
