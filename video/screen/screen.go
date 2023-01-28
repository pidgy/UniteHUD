package screen

import (
	"fmt"
	"image"
	"reflect"
	"unsafe"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/video/proc"
)

var (
	Sources = []string{config.MainDisplay}
)

func Capture() (*image.RGBA, error) {
	r, e := monitorRect()
	if e != nil {
		return nil, e
	}
	return CaptureRect(r)
}

func CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
	hDC := getDC(0)
	if hDC == 0 {
		return nil, fmt.Errorf("Failed to find primary display (%d)", getLastError())
	}
	defer releaseDC(0, hDC)

	m_hDC := createCompatibleDC(hDC)
	if m_hDC == 0 {
		return nil, fmt.Errorf("Could not Create Compatible DC (%d)", getLastError())
	}
	defer deleteDC(m_hDC)

	x, y := rect.Dx(), rect.Dy()

	bt := proc.WindowsBitmapInfo{}
	bt.BmiHeader.BiSize = uint32(reflect.TypeOf(bt.BmiHeader).Size())
	bt.BmiHeader.BiWidth = int32(x)
	bt.BmiHeader.BiHeight = int32(-y)
	bt.BmiHeader.BiPlanes = 1
	bt.BmiHeader.BiBitCount = 32
	bt.BmiHeader.BiCompression = proc.BIRGBCompression

	ptr := unsafe.Pointer(uintptr(0))

	mhBmp := createDIBSection(m_hDC, &bt, proc.DIBRGBColors, &ptr, 0, 0)
	if mhBmp == 0 {
		return nil, fmt.Errorf("Could not Create DIB Section err:%d.\n", getLastError())
	}
	if mhBmp == proc.InvalidParameter {
		return nil, fmt.Errorf("One or more of the input parameters is invalid while calling CreateDIBSection.\n")
	}
	defer deleteObject(proc.HGDIOBJ(mhBmp))

	obj := selectObject(m_hDC, proc.HGDIOBJ(mhBmp))
	if obj == 0 {
		return nil, fmt.Errorf("error occurred and the selected object is not a region err:%d.\n", getLastError())
	}
	if obj == 0xffffffff { //GDI_ERROR
		return nil, fmt.Errorf("GDI_ERROR while calling SelectObject err:%d.\n", getLastError())
	}
	defer deleteObject(obj)

	if !bitBlt(m_hDC, 0, 0, x, y, hDC, rect.Min.X, rect.Min.Y, proc.SrcCopy|proc.CaptureBLT) {
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

func releaseDC(hwnd proc.HWND, hDC proc.HDC) bool {
	ret, _, _ := proc.ReleaseDC.Call(uintptr(hwnd), uintptr(hDC))
	return ret != 0
}

func monitorRect() (image.Rectangle, error) {
	hDC := getDC(0)
	if hDC == 0 {
		return image.Rectangle{}, fmt.Errorf("Could not Get primary display err:%d\n", getLastError())
	}
	defer releaseDC(0, hDC)

	x0, y0 := 0, 0
	x1, y1 := getDeviceCaps(hDC, proc.HorzRes), getDeviceCaps(hDC, proc.VertRes)

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
