package wapi

import (
	"fmt"
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

	// Callback function passed to (EnumerateWindows). Enumeration stops on first error returned.
	EnumerateWindowsCallback func(w Window) (stop bool)
)

func (b Bitmap) Delete() {
	DeleteObject.Call(b.id())
}

func (b Bitmap) id() uintptr {
	return uintptr(b)
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
	for i := uintptr(0); i < uintptr(length); i++ {
		slice[i] = *(*byte)(unsafe.Pointer(data + i))
	}
	return slice
}

func (d Device) Compatible() (Device, error) {
	dst, _, err := CreateCompatibleDC.Call(d.id())
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
	ReleaseDC.Call(0, d.id())
}

func (d Device) Delete() {
	DeleteDC.Call(d.id())
}

func (d Device) id() uintptr {
	return uintptr(d)
}

func EnumerateWindows(callback EnumerateWindowsCallback) error {
	_, _, err := EnumWindows.Call(syscall.NewCallback(func(h, l uintptr) uintptr {
		if callback(Window(h)) {
			return 0 // Stop.
		}
		return 1
	}), 0, 0)
	if err != syscall.Errno(0) {
		return err
	}

	return nil
}

func GetMonitorInfo() (MonitorInfo, error) {
	mi := MonitorInfo{
		cbSize: uint32(unsafe.Sizeof(MonitorInfo{})),
	}

	v, _, err := MonitorFromWindow.Call(uintptr(0), MonitorFromWindowOptions.DefaultToPrimary)
	if v == 0 {
		return mi, err
	}

	v, _, err = GetMonitorInfoW.Call(v, uintptr(unsafe.Pointer(&mi)))
	if v == 0 {
		return mi, err
	}

	return mi, nil
}

func MoveWindowNoSize(hwnd uintptr, pos image.Point) {
	MoveWindow.Call(hwnd, uintptr(pos.X), uintptr(pos.Y), 0, 0, uintptr(1))
}

func NewWindow(title string) (Window, error) {
	argv, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return 0, err
	}

	r, _, _ := FindWindow.Call(0, uintptr(unsafe.Pointer(argv)))
	if r == 0 {
		return 0, fmt.Errorf("failed to find window with title: %s", title)
	}

	return Window(r), nil
}

func (o Object) Delete() {
	DeleteObject.Call(o.id())
}

func (o Object) id() uintptr {
	return uintptr(o)
}

func ObjectSelect(hwnd1, hwnd2 uintptr) {
	SelectObject.Call(hwnd1, hwnd2)
}

func (p Point) String() string {
	return fmt.Sprintf("(%d,%d)", p.X, p.Y)
}

func (r Rectangle) Eq(r2 Rectangle) bool {
	return r.Bottom == r2.Bottom && r.Left == r2.Left && r.Right == r2.Right && r.Top == r2.Top
}

func (r Rectangle) String() string {
	return fmt.Sprintf("[L:%d,T:%d,R:%d,B:%d]", r.Left, r.Top, r.Right, r.Bottom)
}

// ? ShowWindowMinimizedRestore: ShowWindowFlags.ShowMinimized not working.
func ShowWindowMinimizedRestore(hwnd uintptr) {
	ShowWindow.Call(hwnd, ShowWindowFlags.ShowMinimized|ShowWindowFlags.Restore)
}

func ShowWindowHide(hwnd uintptr) {
	ShowWindow.Call(hwnd, ShowWindowFlags.Hide)
}

func ShowWindowRestore(hwnd uintptr) {
	ShowWindow.Call(hwnd, ShowWindowFlags.Restore)
}

func SetWindowAlwaysOnTop(hwnd uintptr) {
	go SetWindowPos.Call(hwnd, uintptr(HWNDInsertAfterFlags.TopMost), 0, 0, 0, 0, SetWindowPosFlags.NoMove|SetWindowPosFlags.NoSize)
}

func SetWindowNotAlwaysOnTop(hwnd uintptr) {
	go SetWindowPos.Call(hwnd, uintptr(HWNDInsertAfterFlags.NoTopMost), 0, 0, 0, 0, SetWindowPosFlags.NoMove|SetWindowPosFlags.NoSize)
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

func (w Window) Device() (Device, error) {
	r, _, err := GetDC.Call(w.HWND())
	if r == 0 {
		return 0, err
	}
	return Device(r), nil
}

func (w Window) Dimensions() (image.Rectangle, error) {
	rect, err := w.Rect()
	if err != nil {
		return image.Rectangle{}, err
	}

	return image.Rect(0, 0, int(rect.Right), int(rect.Bottom)), nil
}

func (w Window) Select(b Bitmap) error {
	r, _, err := SelectObject.Call(w.HWND(), b.id())
	if r == 0 {
		return err
	}
	return nil
}

func (w Window) HWND() uintptr {
	return uintptr(w)
}

// InfoStatus will return the WindowInfoStatus field from GetWindowInfo, See: WindowInfoStatus.
func (w Window) InfoStatus() WindowInfoStatus {
	info := WindowInfo{}
	r, _, _ := GetWindowInfo.Call(w.HWND(), uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		return WindowInfoStatusUnknown
	}
	return info.Status
}

func (w Window) Rect() (Rectangle, error) {
	r := Rectangle{}
	_, _, err := GetClientRect.Call(w.HWND(), uintptr(unsafe.Pointer(&r)))
	if err != nil {
		if err != syscall.Errno(0) {
			return r, err
		}
	}
	return r, nil
}

func (w Window) Title() (string, error) {
	var str *uint16
	b := make([]uint16, 200)
	maxCount := uint32(200)
	str = &b[0]

	r, _, _ := GetWindowTextW.Call(w.HWND(), uintptr(unsafe.Pointer(str)), uintptr(maxCount))
	if r == 0 {
		return "", fmt.Errorf("invalid title or handle: %d", w.HWND())
	}

	return syscall.UTF16ToString(b), nil
}

// Visible will determine if a window is "technically visible", see InfoStatus for "actually visible".
func (w Window) Visible() bool {
	f, _, _ := IsWindowVisible.Call(w.HWND())
	return f == 1
}

func (w *WindowPlacement) String() string {
	return fmt.Sprintf("len: %d, flags: %d, cmd: %d, min: %s, max: %s, normal: %s, device: %s", w.Len, w.Flags, w.ShowCommand, w.Min, w.Max, w.Normal, w.Device)
}

// https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-systemparametersinfoa?redirectedfrom=MSDN
func WorkArea() (Rectangle, error) {
	var r Rectangle

	v, _, err := SystemParametersInfoA.Call(SystemParametersInfoOptions.GetWorkArea, 0, uintptr(unsafe.Pointer(&r)), 0)
	if v == 0 {
		return r, err
	}

	return r, nil
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
