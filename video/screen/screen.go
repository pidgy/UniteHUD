package screen

import (
	"fmt"
	"image"
	"reflect"
	"sync"
	"unsafe"

	"github.com/kbinani/screenshot"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/video/proc"
)

var (
	Sources = []string{}

	displays = map[string]int{}
	bounds   = map[string]image.Rectangle{}

	mutex = &sync.RWMutex{}
)

func Bounds() image.Rectangle {
	return bounds[config.Current.Window]
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

	mutex.Lock()
	Sources = sourcesTmp
	displays = displaysTmp
	bounds = boundsTmp
	mutex.Unlock()
}

func Capture() (*image.RGBA, error) {
	mutex.RLock()
	r := bounds[config.Current.Window]
	mutex.RUnlock()

	return CaptureRect2(r)
}

func CaptureRect(r image.Rectangle) (*image.RGBA, error) {
	mutex.RLock()
	b := bounds[config.Current.Window]
	mutex.RUnlock()

	r.Min.X = b.Min.X + r.Min.X
	r.Max.X = b.Min.X + r.Max.X

	r.Min.Y = b.Min.Y + r.Min.Y
	r.Max.Y = b.Min.Y + r.Max.Y

	return CaptureRect2(r)
}

func CaptureRect2(rect image.Rectangle) (*image.RGBA, error) {
	hdc := getDC(0)
	if hdc == 0 {
		return nil, fmt.Errorf("Failed to find primary display (%d)", getLastError())
	}
	defer releaseDC(0, hdc)

	mHDC := createCompatibleDC(hdc)
	if mHDC == 0 {
		return nil, fmt.Errorf("Could not Create Compatible DC (%d)", getLastError())
	}
	defer proc.DeleteDC.Call(uintptr(mHDC))

	x, y := rect.Dx(), rect.Dy()

	bt := proc.WindowsBitmapInfo{}
	bt.BmiHeader.BiSize = uint32(reflect.TypeOf(bt.BmiHeader).Size())
	bt.BmiHeader.BiWidth = int32(x)
	bt.BmiHeader.BiHeight = int32(-y)
	bt.BmiHeader.BiPlanes = 1
	bt.BmiHeader.BiBitCount = 32
	bt.BmiHeader.BiCompression = proc.BIRGBCompression

	ptr := unsafe.Pointer(uintptr(0))

	mhBmp := createDIBSection(mHDC, &bt, proc.DIBRGBColors, &ptr, 0, 0)
	if mhBmp == 0 {
		return nil, fmt.Errorf("Could not Create DIB Section err:%d.\n", getLastError())
	}
	if mhBmp == proc.InvalidParameter {
		return nil, fmt.Errorf("One or more of the input parameters is invalid while calling CreateDIBSection.\n")
	}
	defer deleteObject(proc.HGDIOBJ(mhBmp))

	obj := selectObject(mHDC, proc.HGDIOBJ(mhBmp))
	if obj == 0 {
		return nil, fmt.Errorf("error occurred and the selected object is not a region err:%d.\n", getLastError())
	}
	if obj == 0xffffffff { //GDI_ERROR
		return nil, fmt.Errorf("GDI_ERROR while calling SelectObject err:%d.\n", getLastError())
	}
	defer deleteObject(obj)

	if !bitBlt(mHDC, 0, 0, x, y, hdc, rect.Min.X, rect.Min.Y, proc.SrcCopy|proc.CaptureBLT) {
		return nil, fmt.Errorf("BitBlt failed err:%d.\n", getLastError())
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

func bitBlt(hdcDest proc.HDC, nXDest, nYDest, nWidth, nHeight int, hdcSrc proc.HDC, nXSrc, nYSrc int, dwRop uint) bool {
	ret, _, _ := proc.BitBlt.Call(
		uintptr(hdcDest),
		uintptr(nXDest),
		uintptr(nYDest),
		uintptr(nWidth),
		uintptr(nHeight),
		uintptr(hdcSrc),
		uintptr(nXSrc),
		uintptr(nYSrc),
		uintptr(dwRop))

	return ret != 0
}

func createCompatibleDC(hdc proc.HDC) proc.HDC {
	ret, _, _ := proc.CreateCompatibleDC.Call(
		uintptr(hdc))

	if ret == 0 {
		panic("Create compatible DC failed")
	}

	return proc.HDC(ret)
}

func createDIBSection(hdc proc.HDC, pbmi *proc.WindowsBitmapInfo, iUsage uint, ppvBits *unsafe.Pointer, hSection proc.HANDLE, dwOffset uint) proc.HBITMAP {
	ret, _, _ := proc.CreateDIBSection.Call(
		uintptr(hdc),
		uintptr(unsafe.Pointer(pbmi)),
		uintptr(iUsage),
		uintptr(unsafe.Pointer(ppvBits)),
		uintptr(hSection),
		uintptr(dwOffset))

	return proc.HBITMAP(ret)
}

func deleteDC(hdc proc.HDC) bool {
	ret, _, _ := proc.DeleteDC.Call(uintptr(hdc))
	return ret != 0
}

func deleteObject(hObject proc.HGDIOBJ) bool {
	ret, _, _ := proc.DeleteObject.Call(uintptr(hObject))
	return ret != 0
}

func display(name string, count int) string {
	if count <= 1 {
		return name
	}
	return fmt.Sprintf("%s %d", name, count)
}

func getDC(hwnd proc.HWND) proc.HDC {
	ret, _, _ := proc.GetDC.Call(uintptr(hwnd))
	return proc.HDC(ret)
}

func getDeviceCaps(hdc proc.HDC, index int) int {
	ret, _, _ := proc.GetDeviceCaps.Call(
		uintptr(hdc),
		uintptr(index))

	return int(ret)
}

func getLastError() uint32 {
	ret, _, _ := proc.GetLastError.Call()
	return uint32(ret)
}

func releaseDC(hwnd proc.HWND, hdc proc.HDC) bool {
	ret, _, _ := proc.ReleaseDC.Call(uintptr(hwnd), uintptr(hdc))
	return ret != 0
}

func mainDisplayRect() (image.Rectangle, error) {
	hdc := getDC(0)
	if hdc == 0 {
		return image.Rectangle{}, fmt.Errorf("Could not Get primary display err:%d\n", getLastError())
	}
	defer releaseDC(0, hdc)

	x0, y0 := 0, 0
	x1, y1 := getDeviceCaps(hdc, proc.HorzRes), getDeviceCaps(hdc, proc.VertRes)

	/*
		switch config.Current.Window {
		case config.LeftDisplay:
			x0, y0 = -100, -100
			x1, y1 = x1-100, y1-100
		case config.RightDisplay:
			x0, y0 = x1, y1
			x1, y1 = x1*2, y1*2
		}
	*/

	return image.Rect(x0, y0, x1, y1), nil
}

func selectObject(hdc proc.HDC, hgdiobj proc.HGDIOBJ) proc.HGDIOBJ {
	ret, _, _ := proc.SelectObject.Call(
		uintptr(hdc),
		uintptr(hgdiobj))

	if ret == 0 {
		panic("SelectObject failed")
	}

	return proc.HGDIOBJ(ret)
}
