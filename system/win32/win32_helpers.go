package win32

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const. Then remove this comment.

import (
	"fmt"
	"image"
	"math"
	"reflect"
	"syscall"
	"unsafe"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
)

const BitmapInfoHeaderSize = uint32(40)

type (
	Bitmap    uintptr
	BitValues uintptr
	Bytes     uintptr
	Device    uintptr
	Object    uintptr
	Window    uintptr

	Monitor struct {
		HWND  uintptr
		Index int
		Rect  Rect
	}

	Monitors struct {
		Active [4]Monitor
		Count  int
	}

	WindowInfoEx struct {
		Window
		WindowInfo
		Title string
	}

	Windows struct {
		Infos []WindowInfoEx
	}

	sliceHeader struct {
		Data uintptr
		Len  int
		Cap  int
	}
)

const (
	false32 uintptr = iota
	true32

	enumStop     = false32
	enumContinue = true32
)

var (
	enumerateDisplayMonitorsCallback = syscall.NewCallback(func(h uintptr, dc uintptr, r *Rect, d uintptr) uintptr {
		m := (*Monitors)(unsafe.Pointer(d))
		if m.Count == 4 {
			return enumStop
		}
		m.Active[m.Count] = Monitor{0, m.Count, *r}
		m.Count++
		return enumContinue
	})

	enumerateWindowsCallback = syscall.NewCallback(func(h, l uintptr) uintptr {
		ws := (*Windows)(unsafe.Pointer(l))
		if ws == nil {
			return enumStop
		}

		w := Window(h)

		t, err := w.Title()
		if err == nil {
			ws.Infos = append(ws.Infos, WindowInfoEx{Window: w, WindowInfo: w.Info(), Title: t})
		}

		return enumContinue
	})
)

func (b BitValues) Image(size image.Point) *image.RGBA {
	var slice []byte
	hdr := (*sliceHeader)(unsafe.Pointer(&slice))
	hdr.Data = uintptr(b)
	hdr.Len = size.X * size.Y * 4
	hdr.Cap = hdr.Len

	// Using the raw data, grab the RGBA data and transform it into an image.RGBA
	raw := make([]byte, len(slice))
	for i := 0; i < len(raw); i += 4 {
		raw[i], raw[i+2], raw[i+1], raw[i+3] =
			slice[i+2], slice[i], slice[i+1], slice[i+3]
	}

	return &image.RGBA{
		Pix:    raw,
		Stride: 4 * size.X,
		Rect:   image.Rectangle{Max: size},
	}
}

func (b Bitmap) Delete() {
	deleteObject.Call(b.id())
}

func (b Bitmap) id() uintptr {
	return uintptr(b)
}

func (b *BitmapInfo) CreateRGBSection(d *Device) (bitmap Bitmap, data Bytes, err error) {
	r, _, err := createDIBSection.Call(
		uintptr(d.HWND()),
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

func (d Device) HWND() uintptr {
	return uintptr(d)
}

func GetAllMonitors() (Monitors, error) {
	m := Monitors{}
	_, _, err := enumDisplayMonitors.Call(0, 0, enumerateDisplayMonitorsCallback, uintptr(unsafe.Pointer(&m)))
	if err != syscall.Errno(0) {
		return m, err
	}
	return m, nil
}

func GetAllWindows() (Windows, error) {
	w := Windows{}
	enumWindows.Call(enumerateWindowsCallback, uintptr(unsafe.Pointer(&w)))
	return w, nil
}

func GetMonitorHandleFromIndex(index int) (hwnd uintptr, err error) {
	ms, err := GetAllMonitors()
	if err != nil {
		return 0, err
	}
	for i, m := range ms.Active {
		if i == index {
			return m.HWND, nil
		}
	}

	return 0, fmt.Errorf("invalid monitor index: %d", index)
}

func GetMonitorIndexFromMonitorInfo(mi MonitorInfo) (int, error) {
	ms, err := GetAllMonitors()
	if err != nil {
		return -1, err
	}
	for i, m := range ms.Active {
		if m.Rect.Eq(mi.Monitor) {
			return i, nil
		}
	}
	return -1, fmt.Errorf("invalid monitor info")
}

func GetMonitorInfo() (MonitorInfo, error) {
	mi := MonitorInfo{
		cbSize: uint32(unsafe.Sizeof(MonitorInfo{})),
	}

	v, _, err := monitorFromWindow.Call(0, MonitorFromWindowOptions.DefaultToPrimary)
	if v == 0 {
		return mi, err
	}

	v, _, err = getMonitorInfoW.Call(v, uintptr(unsafe.Pointer(&mi)))
	if v == 0 {
		return mi, err
	}

	return mi, nil
}

func GetMonitorInfoFromWindow(w Window) (MonitorInfo, error) {
	mi := MonitorInfo{
		cbSize: uint32(unsafe.Sizeof(MonitorInfo{})),
	}

	v, _, err := monitorFromWindow.Call(w.HWND(), MonitorFromWindowOptions.DefaultToPrimary)
	if v == 0 {
		return mi, err
	}

	v, _, err = getMonitorInfoW.Call(v, uintptr(unsafe.Pointer(&mi)))
	if v == 0 {
		return mi, err
	}

	return mi, nil
}

func (w Monitor) Capture(area image.Rectangle, resize image.Point) (*image.RGBA, error) {
	scaling := true
	if resize.Eq(image.Pt(0, 0)) || resize.Eq(area.Size()) {
		scaling = false
		resize = area.Size()
	}

	// Get the device context for screenshotting.
	src, _, err := getDC.Call(w.HWND)
	if src == 0 {
		return nil, fmt.Errorf("failed to prepare monitor capture: %v", err)
	}
	defer rReleaseDC.Call(0, src)

	// Grab a compatible DC for drawing.
	dst, _, err := createCompatibleDC.Call(src)
	if dst == 0 {
		return nil, fmt.Errorf("failed to create dest for monitor: %v", err)
	}
	defer deleteDC.Call(dst)

	// Get the bitmap we're going to draw onto.
	bitmapInfo := BitmapInfo{
		BmiHeader: BitmapInfoHeader{
			BiSize:        uint32(reflect.TypeFor[BitmapInfoHeader]().Size()),
			BiWidth:       int32(resize.X),
			BiHeight:      -int32(resize.Y),
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: BitmapInfoHeaderCompression.RGB,
		},
	}

	bitmapData := unsafe.Pointer(uintptr(0))
	bitmap, _, err := createDIBSection.Call(
		dst,
		uintptr(unsafe.Pointer(&bitmapInfo)),
		0,
		uintptr(unsafe.Pointer(&bitmapData)),
		0,
		0,
	)
	if bitmap == 0 {
		return nil, fmt.Errorf("failed to create bitmap for \"%s\" monitor: %v", config.Current.Video.Capture.Window.Name, err)
	}
	defer deleteObject.Call(bitmap)

	// Select the object and paint it.
	selectObject.Call(dst, bitmap)

	if scaling {
		// res, _, _ := setStretchBltMode.Call(dst, SetStretchBltMode.ColorOnColor)
		// if res == 0 {
		// 	notify.Warn("[Win32] Failed to set StretchBlt mode")
		// }

		res, _, _ := stretchBlt.Call(
			// Dst.
			dst,
			0,
			0,
			uintptr(resize.X),
			uintptr(resize.Y),
			// Src.
			src,
			uintptr(area.Min.X),
			uintptr(area.Min.Y),
			uintptr(area.Max.X),
			uintptr(area.Max.Y),
			bitBltRasterOperations.SrcPaint,
		)
		if res == 0 {
			notify.Error("[Win32] Failed to scale and capture monitor")
			return nil, fmt.Errorf("bitblt returned: %d", res)
		}
	} else {
		res, _, err := bitBlt.Call(
			// Dst.
			dst,
			0,
			0,
			uintptr(resize.X),
			uintptr(resize.Y),
			// Src.
			src,
			uintptr(area.Min.X),
			uintptr(area.Min.Y),
			bitBltRasterOperations.SrcPaint,
		)
		if res == 0 {
			notify.Error("[Win32] Failed to capture monitor (%v)", err)
			return nil, fmt.Errorf("failed to capture window: %v", err)
		}
	}

	// Convert the bitmap to *image.RGBA. We first start by directly creating a slice. This is unsafe but we know the underlying structure directly.
	var slice []byte
	hdr := (*sliceHeader)(unsafe.Pointer(&slice))
	hdr.Data = uintptr(bitmapData)
	hdr.Len = resize.X * resize.Y * 4
	hdr.Cap = hdr.Len

	// Using the raw data, grab the RGBA data and transform it into an image.RGBA
	raw := make([]byte, len(slice))
	for i := 0; i < len(raw); i += 4 {
		raw[i], raw[i+2], raw[i+1], raw[i+3] =
			slice[i+2], slice[i], slice[i+1], slice[i+3]
	}

	return &image.RGBA{
		Pix:    raw,
		Stride: 4 * resize.X,
		Rect: image.Rect(
			0,
			0,
			resize.X,
			resize.Y,
		),
	}, nil
}
func (w Monitor) String() string {
	return fmt.Sprintf("Monitor: Handle: %d, Index: %d, Rect: %s", w.HWND, w.Index, w.Rect)
}

func (mi MonitorInfo) String() string {
	return fmt.Sprintf("Monitor Info: Rect: %s, Work Area: %s, Flags: %d", mi.Monitor, mi.WorkArea, mi.Flags)
}

func MoveWindowNoSize(hwnd uintptr, pos image.Point) {
	moveWindow.Call(hwnd, uintptr(pos.X), uintptr(pos.Y), 0, 0, uintptr(1))
}

func NewWindow(title string) (Window, error) {
	argv, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return 0, err
	}

	r, _, _ := findWindow.Call(0, uintptr(unsafe.Pointer(argv)))
	if r == 0 {
		return 0, fmt.Errorf("failed to find window with title: %s", title)
	}

	return Window(r), nil
}

func (o Object) Delete() {
	deleteObject.Call(o.id())
}

func (o Object) id() uintptr {
	return uintptr(o)
}

func (p Point) String() string {
	return fmt.Sprintf("(%d,%d)", p.X, p.Y)
}

func (r Rect) Empty() bool {
	return r.Left >= r.Right || r.Top >= r.Bottom
}

func (r Rect) Eq(r2 Rect) bool {
	return r.Bottom == r2.Bottom && r.Left == r2.Left && r.Right == r2.Right && r.Top == r2.Top
}

func (r Rect) Image() image.Rectangle {
	return image.Rect(int(r.Left), int(r.Top), int(r.Right), int(r.Bottom))
}

func (r Rect) String() string {
	return fmt.Sprintf("[L:%d,T:%d,R:%d,B:%d]", r.Left, r.Top, r.Right, r.Bottom)
}

// SetProcessDpiAwareness ensures that Windows API calls will tell us the scale factor for our
// screen so that our screenshot works on hi-res displays.
func SetProcessDPIAwareness(ctx SetProcessDpiAwarenessContext) error {
	_, _, err := setProcessDpiAwareness.Call(uintptr(ctx))
	if err != syscall.Errno(0) {
		return err
	}
	return nil
}

func SetThreadExecutionState(states ...ThreadExecutionState) error {
	t := ThreadExecutionState(0)
	for _, state := range states {
		t |= state
	}
	_, _, err := setThreadExecutionState.Call(uintptr(t))
	if err != syscall.Errno(0) {
		return err
	}
	return nil
}

func SetWindowAlwaysOnTop(hwnd uintptr) {
	go setWindowPos.Call(hwnd, uintptr(HWNDInsertAfterFlags.TopMost), 0, 0, 0, 0, SetWindowPosFlags.NoMove|SetWindowPosFlags.NoSize)
}

func SetWindowNotAlwaysOnTop(hwnd uintptr) {
	go setWindowPos.Call(hwnd, uintptr(HWNDInsertAfterFlags.NoTopMost), 0, 0, 0, 0, SetWindowPosFlags.NoMove|SetWindowPosFlags.NoSize)
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

// ? ShowWindowMinimizedRestore: ShowWindowFlags.ShowMinimized not working.
func ShowWindowMinimizedRestore(hwnd uintptr) {
	showWindow.Call(hwnd, ShowWindowFlags.ShowMinimized|ShowWindowFlags.Restore)
}

func ShowWindowHide(hwnd uintptr) {
	showWindow.Call(hwnd, ShowWindowFlags.Hide)
}

func ShowWindowRestore(hwnd uintptr) {
	showWindow.Call(hwnd, ShowWindowFlags.Restore)
}

/*
func (w Window) Capture2(area image.Rectangle, resize image.Point) (*image.RGBA, error) {
	scaling := true
	if resize.Eq(image.Pt(0, 0)) {
		scaling = true
		resize = area.Size()
	}

	// Get the device context for screenshotting.
	src, _, err := getDC.Call(w.HWND())
	if src == 0 {
		return nil, fmt.Errorf("failed to prepare screen capture: %v", err)
	}
	defer ReleaseDC.Call(0, src)

	// Grab a compatible DC for drawing.
	dst, _, err := CreateCompatibleDC.Call(src)
	if dst == 0 {
		return nil, fmt.Errorf("failed to create DC for drawing: %v", err)
	}
	defer DeleteDC.Call(dst)

	// Get the bitmap we're going to draw onto.
	bitmapInfo := BitmapInfo{
		BmiHeader: BitmapInfoHeader{
			BiSize:        uint32(reflect.TypeFor[BitmapInfoHeader]().Size()),
			BiWidth:       int32(resize.X),
			BiHeight:      -int32(resize.Y),
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: BitmapInfoHeaderCompression.RGB,
		},
	}

	bitmapData := unsafe.Pointer(uintptr(0))
	bitmap, _, err := CreateDIBSection.Call(
		dst,
		uintptr(unsafe.Pointer(&bitmapInfo)),
		0,
		uintptr(unsafe.Pointer(&bitmapData)),
		0,
		0,
	)
	if bitmap == 0 {
		return nil, fmt.Errorf("failed to create bitmap for \"%s\" window: %v", config.Current.Video.Capture.Window.Name, err)
	}
	defer DeleteObject.Call(bitmap)

	// Select the object and paint it.
	SelectObject.Call(dst, bitmap)

	if scaling {
				// res, _, _ := setStretchBltMode.Call(dst, SetStretchBltMode.ColorOnColor)
		// if res == 0 {
		// 	notify.Warn("[Win32] Failed to set StretchBlt mode")
		// }

		res, _, _ := stretchBlt.Call(

		res, _, _ = stretchBlt.Call(
			// Dst.
			dst,
			0,
			0,
			uintptr(resize.X),
			uintptr(resize.Y),
			// Src.
			src,
			uintptr(area.Min.X),
			uintptr(area.Min.Y),
			uintptr(area.Max.X),
			uintptr(area.Max.Y),
			BitBltRasterOperations.SrcPaint,
		)
		if res == 0 {
			notify.Error("[Win32] Failed to scale and capture \"%s\" window", config.Current.Video.Capture.Window.Name)
			return nil, fmt.Errorf("bitblt returned: %d", res)
		}
	} else {
		res, _, err := bitBlt.Call(
			// Dst.
			dst,
			0,
			0,
			uintptr(resize.X),
			uintptr(resize.Y),
			// Src.
			src,
			uintptr(area.Min.X),
			uintptr(area.Min.Y),
			BitBltRasterOperations.SrcPaint,
		)
		if res == 0 {
			notify.Error("[Win32] Failed to capture \"%s\" window (%v)", config.Current.Video.Capture.Window.Name, err)
		return nil, fmt.Errorf("failed to capture window: %v", err)
		}
	}

	// Convert the bitmap to an image.Image. We first start by directly
	// creating a slice. This is unsafe but we know the underlying structure
	// directly.
	var slice []byte
	hdr := (*sliceHeader)(unsafe.Pointer(&slice))
	hdr.Data = uintptr(bitmapData)
	hdr.Len = resize.X * resize.Y * 4
	hdr.Cap = hdr.Len

	// Using the raw data, grab the RGBA data and transform it into an image.RGBA
	raw := make([]byte, len(slice))
	for i := 0; i < len(raw); i += 4 {
		raw[i], raw[i+2], raw[i+1], raw[i+3] =
			slice[i+2], slice[i], slice[i+1], slice[i+3]
	}

	return &image.RGBA{
		Pix:    raw,
		Stride: 4 * resize.X,
		Rect:   image.Rectangle{Max: resize},
	}, nil
*/

func (w Window) capture(area image.Rectangle) (*image.RGBA, error) {
	// Get the device context for screenshotting.
	src, _, err := getDC.Call(w.HWND())
	if src == 0 {
		return nil, fmt.Errorf("failed to prepare window capture: %v", err)
	}
	defer rReleaseDC.Call(0, src)

	// Grab a compatible DC for drawing.
	dst, _, err := createCompatibleDC.Call(src)
	if dst == 0 {
		return nil, fmt.Errorf("failed to create dest for window: %v", err)
	}
	defer deleteDC.Call(dst)

	// Get the bitmap we're going to draw onto.
	bitmapInfo := BitmapInfo{
		BmiHeader: BitmapInfoHeader{
			BiSize:        uint32(reflect.TypeFor[BitmapInfoHeader]().Size()),
			BiWidth:       int32(area.Dx()),
			BiHeight:      -int32(area.Dy()),
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: BitmapInfoHeaderCompression.RGB,
		},
	}

	bitmapData := unsafe.Pointer(uintptr(0))
	bitmap, _, err := createDIBSection.Call(
		dst,
		uintptr(unsafe.Pointer(&bitmapInfo)),
		0,
		uintptr(unsafe.Pointer(&bitmapData)),
		0,
		0,
	)
	if bitmap == 0 {
		return nil, fmt.Errorf("failed to create bitmap for window: %v", err)
	}
	defer deleteObject.Call(bitmap)

	// Select the object and paint it.
	selectObject.Call(dst, bitmap)

	res, _, err := transparentBlt.Call(
		// Dst.
		dst,
		0,
		0,
		uintptr(area.Dx()),
		uintptr(area.Dy()),
		// Src.
		src,
		uintptr(area.Min.X),
		uintptr(area.Min.Y),
		uintptr(area.Max.X),
		uintptr(area.Max.Y),
		232323,
	)
	if res == 0 {
		notify.Error("[Win32] Failed to capture \"%s\" (%v)", config.Current.Video.Capture.Window.Name, err)
		return nil, fmt.Errorf("failed to capture window: %v", err)
	}

	// res, _, err := bitBlt.Call(
	// 	// Dst.
	// 	dst,
	// 	0,
	// 	0,
	// 	uintptr(area.Dx()),
	// 	uintptr(area.Dy()),
	// 	// Src.
	// 	src,
	// 	uintptr(area.Min.X),
	// 	uintptr(area.Min.Y),
	// 	bitBltRasterOperations.SrcPaint,
	// )
	// if res == 0 {
	// 	notify.Error("[Win32] Failed to capture \"%s\" (%v)", config.Current.Video.Capture.Window.Name, err)
	// 	return nil, fmt.Errorf("failed to capture window: %v", err)
	// }

	// Convert the bitmap to an image.Image. We first start by directly
	// creating a slice. This is unsafe but we know the underlying structure
	// directly.
	var slice []byte
	hdr := (*sliceHeader)(unsafe.Pointer(&slice))
	hdr.Data = uintptr(bitmapData)
	hdr.Len = area.Dx() * area.Dy() * 4
	hdr.Cap = hdr.Len

	// Using the raw data, grab the RGBA data and transform it into an image.RGBA
	raw := make([]byte, len(slice))
	for i := 0; i < len(raw); i += 4 {
		raw[i], raw[i+2], raw[i+1], raw[i+3] =
			slice[i+2], slice[i], slice[i+1], slice[i+3]
	}

	return &image.RGBA{
		Pix:    raw,
		Stride: 4 * area.Dx(),
		Rect: image.Rect(
			0,
			0,
			area.Dx(),
			area.Dy(),
		),
	}, nil
}

func (w Window) captureResize(area image.Rectangle, resize image.Point) (*image.RGBA, error) {
	// Get the device context for screenshotting.
	src, _, err := getDC.Call(w.HWND())
	if src == 0 {
		return nil, fmt.Errorf("failed to prepare window capture: %v", err)
	}
	defer rReleaseDC.Call(0, src)

	// ------- Part 1: Scaling -------

	// Scale the original image to a 16:9 format.
	scale := image.Pt(area.Dx(), 0)
	if math.Abs(float64(resize.Y-area.Dy())) > math.Abs(float64(area.Dy())*16.0/9.0+0.5-float64(area.Dx())) {
		scale.Y = int(float64(area.Dy())*16.0/9.0 + 0.5)
	} else {
		scale.Y = int(float64(area.Dx())*9.0/16.0 + 0.5)
	}

	// Grab a compatible DC for drawing.
	dst1, _, err := createCompatibleDC.Call(src)
	if dst1 == 0 {
		return nil, fmt.Errorf("failed to create dest for scaled window: %v", err)
	}
	defer deleteDC.Call(dst1)

	// Get the bitmap we're going to draw onto.
	bitmapInfo1 := BitmapInfo{
		BmiHeader: BitmapInfoHeader{
			BiSize:         uint32(reflect.TypeFor[BitmapInfoHeader]().Size()),
			BiWidth:        int32(scale.X),
			BiHeight:       -int32(scale.Y),
			BiPlanes:       1,
			BiBitCount:     32,
			BiCompression:  BitmapInfoHeaderCompression.RGB,
			BiClrUsed:      1,
			BiClrImportant: 1,
		},
	}

	bitmapData1 := unsafe.Pointer(uintptr(0))
	bitmap1, _, err := createDIBSection.Call(
		dst1,
		uintptr(unsafe.Pointer(&bitmapInfo1)),
		0,
		uintptr(unsafe.Pointer(&bitmapData1)),
		0,
		0,
	)
	if bitmap1 == 0 {
		return nil, fmt.Errorf("failed to create bitmap for scaled window: %v", err)
	}
	defer deleteObject.Call(bitmap1)

	// Select the object and paint it.
	selectObject.Call(dst1, bitmap1)

	// res, _, _ := setStretchBltMode.Call(dst, SetStretchBltMode.ColorOnColor)
	// if res == 0 {
	// 	notify.Warn("[Win32] Failed to set StretchBlt mode")
	// }

	res, _, _ := stretchBlt.Call(
		// Dst.
		dst1,
		0,
		0,
		uintptr(scale.X),
		uintptr(scale.Y),
		// Src.
		src,
		uintptr(area.Min.X),
		uintptr(area.Min.Y),
		uintptr(area.Max.X),
		uintptr(area.Max.Y),
		bitBltRasterOperations.SrcPaint,
	)
	if res == 0 {
		notify.Error("[Win32] Failed to scale and capture \"%s\"", config.Current.Video.Capture.Window.Name)
		return nil, fmt.Errorf("failed to scale window")
	}

	// ------- Part 2: Resizing -------

	// Grab a compatible DC for drawing.
	dst2, _, err := createCompatibleDC.Call(src)
	if dst2 == 0 {
		return nil, fmt.Errorf("failed to create dest for resized window: %v", err)
	}
	defer deleteDC.Call(dst2)

	// Get the bitmap we're going to draw onto.
	bitmapInfo2 := BitmapInfo{
		BmiHeader: BitmapInfoHeader{
			BiSize:         uint32(reflect.TypeFor[BitmapInfoHeader]().Size()),
			BiWidth:        int32(resize.X),
			BiHeight:       -int32(resize.Y),
			BiPlanes:       1,
			BiBitCount:     32,
			BiCompression:  BitmapInfoHeaderCompression.RGB,
			BiClrUsed:      1,
			BiClrImportant: 1,
		},
	}

	bitmapData2 := unsafe.Pointer(uintptr(0))
	bitmap2, _, err := createDIBSection.Call(
		dst2,
		uintptr(unsafe.Pointer(&bitmapInfo2)),
		0,
		uintptr(unsafe.Pointer(&bitmapData2)),
		0,
		0,
	)
	if bitmap2 == 0 {
		return nil, fmt.Errorf("failed to create bitmap for \"%s\" window: %v", config.Current.Video.Capture.Window.Name, err)
	}
	defer deleteObject.Call(bitmap2)

	// Select the object and paint it.
	selectObject.Call(dst2, bitmap2)

	// res, _, _ := setStretchBltMode.Call(dst, SetStretchBltMode.ColorOnColor)
	// if res == 0 {
	// 	notify.Warn("[Win32] Failed to set StretchBlt mode")
	// }

	res, _, _ = stretchBlt.Call(
		// Dst.
		dst2,
		0,
		0,
		uintptr(resize.X),
		uintptr(resize.Y),
		// Src.
		dst1,
		uintptr(0),
		uintptr(0),
		uintptr(scale.X),
		uintptr(scale.Y),
		bitBltRasterOperations.SrcPaint,
	)
	if res == 0 {
		notify.Error("[Win32] Failed to scale and capture \"%s\" window", config.Current.Video.Capture.Window.Name)
		return nil, fmt.Errorf("failed to resize window")
	}

	// ------------ Done scaling ------------.

	// Convert the bitmap to an image.Image. We first start by directly
	// creating a slice. This is unsafe but we know the underlying structure
	// directly.
	var slice []byte
	hdr := (*sliceHeader)(unsafe.Pointer(&slice))
	hdr.Data = uintptr(bitmapData2)
	hdr.Len = resize.X * resize.Y * 4
	hdr.Cap = hdr.Len

	// Using the raw data, grab the RGBA data and transform it into an image.RGBA
	raw := make([]byte, len(slice))
	for i := 0; i < len(raw); i += 4 {
		raw[i], raw[i+2], raw[i+1], raw[i+3] =
			slice[i+2], slice[i], slice[i+1], slice[i+3]
	}

	return &image.RGBA{
		Pix:    raw,
		Stride: 4 * resize.X,
		Rect: image.Rect(
			0,
			0,
			resize.X,
			resize.Y,
		),
	}, nil
}

func (w Window) Capture(area image.Rectangle, resize image.Point) (*image.RGBA, error) {
	if resize.Eq(image.Pt(0, 0)) || resize.Eq(area.Size()) {
		return w.capture(area)
	}
	return w.captureResize(area, resize)
}

func (w Window) Dimensions() (image.Point, error) {
	rect, err := w.Rect()
	if err != nil {
		return image.Pt(-1, -1), err
	}

	return image.Pt(int(rect.Right), int(rect.Bottom)), nil
}

func (w Window) HWND() uintptr {
	return uintptr(w)
}

// Info will return a WindowInfo struct from GetWindowInfo, See:WindowInfo.
func (w Window) Info() WindowInfo {
	wi := WindowInfo{
		Size: uint32(unsafe.Sizeof(WindowInfo{})),
	}
	r, _, _ := getWindowInfo.Call(w.HWND(), uintptr(unsafe.Pointer(&wi)))
	if r == 0 {
		return WindowInfo{Status: WindowStatusUnknown}
	}
	return wi
}

func (w Window) Rect() (Rect, error) {
	r := Rect{}
	_, _, err := getClientRect.Call(w.HWND(), uintptr(unsafe.Pointer(&r)))
	if err != nil {
		if err != syscall.Errno(0) {
			return r, err
		}
	}
	return r, nil
}

func (w Window) RectComplete() (Rect, error) {
	r := Rect{}
	res, _, err := getWindowRect.Call(w.HWND(), uintptr(unsafe.Pointer(&r)))
	if res == 0 {
		return r, err
	}
	return r, nil
}

func (w Window) Select(b Bitmap) error {
	res, _, err := selectObject.Call(w.HWND(), b.id())
	if res == 0 {
		return err
	}
	return nil
}

func (w Window) StretchBlt(src Device, size, origin image.Point, scale float64) error {
	r, _, err := stretchBlt.Call(
		w.HWND(),
		0,
		0,
		uintptr(int(float64(size.X)*scale)),
		uintptr(int(float64(size.Y)*scale)),
		src.HWND(),
		uintptr(origin.X),
		uintptr(origin.Y),
		uintptr(size.X),
		uintptr(size.Y),
		bitBltRasterOperations.SrcPaint,
	)
	if r == 0 {
		return err
	}
	return nil
}

func (w Window) String() string {
	t, err := w.Title()
	if err != nil {
		t = "Window"
	}
	return fmt.Sprintf("%s (%d)", t, w.HWND())
}

func (w Window) Title() (string, error) {
	var str *uint16
	b := make([]uint16, 200)
	maxCount := uint32(200)
	str = &b[0]

	r, _, _ := getWindowTextW.Call(w.HWND(), uintptr(unsafe.Pointer(str)), uintptr(maxCount))
	if r == 0 {
		return "", fmt.Errorf("invalid title or handle: %d", w.HWND())
	}

	return syscall.UTF16ToString(b), nil
}

// Visible will determine if a window is "technically visible", see InfoStatus for "actually visible".
func (w Window) Visible() bool {
	f, _, _ := isWindowVisible.Call(w.HWND())
	return f == 1
}

func (i WindowInfo) HasVisibleStyle() bool {
	return i.Style&WindowStyles.Visible == WindowStyles.Visible
}

func (s WindowStatus) Visible() bool {
	return s == WindowStatusVisible
}

func (w *WindowPlacement) String() string {
	return fmt.Sprintf("len: %d, flags: %d, cmd: %d, min: %s, max: %s, normal: %s, device: %s", w.Len, w.Flags, w.ShowCommand, w.Min, w.Max, w.Normal, w.Device)
}

func (s WindowStyle) Maximized() bool {
	return s&WindowStyles.Maximize == WindowStyles.Maximize
}

func (s WindowStyle) OverlappedWindow() bool {
	return s&WindowStyles.OverlappedWindow == WindowStyles.OverlappedWindow
}

func (s WindowStyle) Visible() bool {
	return s&WindowStyles.Visible == WindowStyles.Visible
}

// https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-systemparametersinfoa?redirectedfrom=MSDN
func WorkArea() (Rect, error) {
	var r Rect

	v, _, err := systemParametersInfoA.Call(SystemParametersInfoOptions.GetWorkArea, 0, uintptr(unsafe.Pointer(&r)), 0)
	if v == 0 {
		return r, err
	}

	return r, nil
}

func helpSetWindowPos(hwnd uintptr, pt image.Point, size image.Point, flags uintptr) {
	go setWindowPos.Call(
		hwnd,
		uintptr(0),
		uintptr(pt.X),
		uintptr(pt.Y),
		uintptr(size.X),
		uintptr(size.Y),
		flags,
	)
}
