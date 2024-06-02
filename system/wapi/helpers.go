package wapi

import (
	"image"
	"syscall"
	"unsafe"
)

const BitmapInfoHeaderSize = uint32(40)

type (
	Bitmap uintptr
	Device uintptr
	Object uintptr
	Window uintptr
	Bytes  uintptr
)

func NewWindow(name string) (Window, error) {
	argv, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}

	r, _, err := FindWindow.Call(0, uintptr(unsafe.Pointer(argv)))
	if r == 0 {
		return 0, err
	}

	return Window(r), nil
}

func (b Bitmap) Delete() {
	DeleteObject.Call(b.id())
}

func (b *BitmapInfo) CreateSection(d Device) (bitmap Bitmap, data uintptr, err error) {
	r, _, err := CreateDIBSection.Call(
		uintptr(d),
		uintptr(unsafe.Pointer(b)),
		0,
		uintptr(unsafe.Pointer(&data)),
		0, 0,
	)
	if r == 0 {
		return 0, 0, err
	}
	return Bitmap(r), data, nil
}

func (b *BitmapInfo) CreateRGBSection(d *Device) (bitmap Bitmap, data Bytes, err error) {
	r, _, err := CreateDIBSection.Call(
		uintptr(d.id()),
		uintptr(unsafe.Pointer(b)),
		uintptr(CreateDIBSectionUsage.RGBColors),
		uintptr(unsafe.Pointer(&data)), 0, 0,
	)
	if r == 0 || r == CreateDIBSectionError.InvalidParameter {
		return 0, 0, err
	}

	return Bitmap(r), data, nil
}

func (b Bytes) Slice(size image.Point) []byte {
	data := uintptr(b)

	length := size.X * size.Y * 4
	slice := make([]byte, length)
	for i := 0; i < length; i++ {
		slice[i] = *(*byte)(unsafe.Pointer(data + uintptr(i)))
	}
	return slice
}

func (d Device) Compatible() (Device, error) {
	dst, _, err := CreateCompatibleDC.Call(uintptr(d))
	if dst == 0 {
		return 0, err
	}
	return Device(dst), nil
}

func (d Device) Copy(src Device, size image.Point, rect image.Rectangle, scale float64) error {
	if scale != 1 {
		size = image.Pt(int(float64(size.X)*scale), int(float64(size.Y)*scale))
	}

	r, _, err := BitBlt.Call(
		d.id(),
		0,
		0,
		uintptr(size.X),
		uintptr(size.Y),
		src.id(),
		uintptr(rect.Min.X),
		uintptr(rect.Min.Y),
		BitBltRasterOperations.CaptureBLT|BitBltRasterOperations.SrcCopy,
	)
	if r == 0 {
		return err
	}
	return nil
}

func (d Device) Select(b Bitmap) (Object, error) {
	r, _, err := SelectObject.Call(d.id(), b.id())
	if r == 0 {
		return 0, err
	}
	return Object(r), nil
}

func (d Device) Release() {
	ReleaseDC.Call(0, uintptr(d))
}

func (d Device) Delete() {
	DeleteDC.Call(uintptr(d))
}

func (o Object) Delete() {
	DeleteObject.Call(o.id())
}

func EnumerateWindows(callback func(h uintptr, p uintptr) uintptr) error {
	r, _, err := EnumWindows.Call(syscall.NewCallback(callback), 0, 0)
	if r == 0 {
		return err
	}
	return nil
}

func MoveWindowNoSize(hwnd uintptr, pos image.Point) {
	MoveWindow.Call(hwnd, uintptr(pos.X), uintptr(pos.Y), 0, 0, uintptr(1))
}

func ObjectSelect(hwnd1, hwnd2 uintptr) {
	SelectObject.Call(hwnd1, hwnd2)
}

func ShowWindowMinimizedRestore(hwnd uintptr) {
	ShowWindow.Call(hwnd, ShowWindowFlags.ShowMinimized)
	ShowWindow.Call(hwnd, ShowWindowFlags.Restore)
}

func ShowWindowHide(hwnd uintptr) {
	ShowWindow.Call(hwnd, ShowWindowFlags.Hide)
}

func SetWindowDarkMode(hwnd uintptr) {
	pv10, pv11 := 1, 1
	DwmSetWindowAttribute.Call(hwnd, DwmWindowAttributeFlags.UseImmersiveDarkMode10, uintptr(unsafe.Pointer(&pv10)), uintptr(4))
	DwmSetWindowAttribute.Call(hwnd, DwmWindowAttributeFlags.UseImmersiveDarkMode11, uintptr(unsafe.Pointer(&pv11)), uintptr(4))
}

func SetWindowPosNone(hwnd uintptr, pt image.Point, size image.Point) {
	helpSetWindowPos(hwnd, pt, size, SetWindowPosFlags.None)
}

func SetWindowPosNoSize(hwnd uintptr, pt image.Point) {
	helpSetWindowPos(hwnd, pt, image.Pt(0, 0), SetWindowPosFlags.NoSize)
}

func SetWindowPosNoSizeNoMoveShowWindow(hwnd uintptr) {
	helpSetWindowPos(hwnd, image.Pt(0, 0), image.Pt(0, 0), SetWindowPosFlags.NoSize|SetWindowPosFlags.NoMove|SetWindowPosFlags.ShowWindow)
}

func SetWindowPosHide(hwnd uintptr, pt image.Point, size image.Point) {
	helpSetWindowPos(hwnd, pt, size, SetWindowPosFlags.Hide)
}

func SetWindowPosShow(hwnd uintptr, pt image.Point, size image.Point) {
	helpSetWindowPos(hwnd, pt, size, SetWindowPosFlags.Show)
}

func (b Bitmap) id() uintptr { return uintptr(b) }
func (d Device) id() uintptr { return uintptr(d) }
func (o Object) id() uintptr { return uintptr(o) }
func (w Window) id() uintptr { return uintptr(w) }

func (w Window) Device() (Device, error) {
	r, _, err := GetDC.Call(w.id())
	if r == 0 {
		return 0, err
	}
	return Device(r), nil
}

func (w Window) Select(b Bitmap) error {
	r, _, err := SelectObject.Call(w.id(), b.id())
	if r == 0 {
		return err
	}
	return nil
}

func (w Window) Visible() bool {
	f, _, _ := IsWindowVisible.Call(w.id())
	return f == 1
}

func (w Window) Dimensions() (image.Rectangle, error) {
	var rect Rect
	r, _, err := GetClientRect.Call(w.id(), uintptr(unsafe.Pointer(&rect)))
	if r == 0 {
		return image.Rectangle{}, err
	}
	return image.Rect(0, 0, int(rect.Right), int(rect.Bottom)), nil
}

func (w Window) Title() (string, error) {
	var str *uint16
	b := make([]uint16, 200)
	maxCount := uint32(200)
	str = &b[0]

	r, _, err := GetWindowTextW.Call(w.id(), uintptr(unsafe.Pointer(str)), uintptr(maxCount))
	if r == 0 {
		return "", err
	}

	return syscall.UTF16ToString(b), nil
}

func helpSetWindowPos(hwnd uintptr, pt image.Point, size image.Point, flags uintptr) {
	go SetWindowPos.Call(
		hwnd,
		uintptr(0),
		uintptr(pt.X),
		uintptr(pt.Y),
		uintptr(size.X),
		uintptr(size.Y),
		flags,
	)
}
