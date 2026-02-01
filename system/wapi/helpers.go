package wapi

import (
	"fmt"
	"image"
	"reflect"
	"syscall"
	"unsafe"
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
		Handle uintptr
		Index  int
		Rect   *Rect
	}

	Monitors struct {
		Active []Monitor
	}

	WindowInfoEx struct {
		Window
		WindowInfo
		Title string
	}

	Windows struct {
		Infos []WindowInfoEx
	}
)

var (
	enumerateDisplayMonitorsCallback = syscall.NewCallback(func(h uintptr, dc uintptr, r *Rect, d uintptr) uintptr {
		m := (*Monitors)(unsafe.Pointer(d))
		m.Active = append(m.Active, Monitor{h, len(m.Active), r})
		return 1
	})

	enumerateWindowsCallback = syscall.NewCallback(func(h, l uintptr) uintptr {
		ws := (*Windows)(unsafe.Pointer(l))
		if ws == nil {
			return 0 // Stop.
		}

		w := Window(h)

		t, err := w.Title()
		if err == nil {
			ws.Infos = append(ws.Infos, WindowInfoEx{Window: w, WindowInfo: w.Info(), Title: t})
		}

		return 1
	})
)

func (b BitValues) Image(size image.Point) *image.RGBA {
	var slice []byte
	sliceHdr := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	sliceHdr.Data = uintptr(b)
	sliceHdr.Len = size.X * size.Y * 4
	sliceHdr.Cap = sliceHdr.Len

	// Using the raw data, grab the RGBA data and transform it into an image.RGBA
	imageBytes := make([]byte, len(slice))
	for i := 0; i < len(imageBytes); i += 4 {
		imageBytes[i], imageBytes[i+2], imageBytes[i+1], imageBytes[i+3] =
			slice[i+2], slice[i], slice[i+1], slice[i+3]
	}

	return &image.RGBA{
		Pix:    imageBytes,
		Stride: 4 * size.X,
		Rect:   image.Rectangle{Max: size},
	}
}

func (b Bitmap) Delete() {
	DeleteObject.Call(b.id())
}

func (b Bitmap) id() uintptr {
	return uintptr(b)
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

func (d Device) BitBlt(src Device, size, origin image.Point) error {
	r, _, err := bitBlt.Call(
		d.id(),
		0,
		0,
		uintptr(size.X),
		uintptr(size.Y),
		src.id(),
		uintptr(origin.X),
		uintptr(origin.Y),
		BitBltRasterOperations.CaptureBLT|BitBltRasterOperations.SrcCopy,
	)
	if r == 0 {
		return err
	}
	return nil
}

func (d Device) Copy(src Device, size image.Point, rect image.Rectangle, scale float64) error {
	if scale != 1 {
		size = image.Pt(int(float64(size.X)*scale), int(float64(size.Y)*scale))
	}

	r, _, err := bitBlt.Call(
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

func (d Device) CreateCompatible() (Device, error) {
	dst, _, _ := CreateCompatibleDC.Call(d.id())
	if dst == 0 {
		return 0, fmt.Errorf("window is not visible")
	}
	return Device(dst), nil
}

func (d Device) CreateDIBSection(size image.Point) (bitmap Bitmap, bitvals BitValues, err error) {
	b := BitmapInfo{}
	b.BmiHeader = BitmapInfoHeader{
		BiSize:        uint32(reflect.TypeOf(b.BmiHeader).Size()),
		BiWidth:       int32(size.X),
		BiHeight:      -int32(size.Y), // Negative value will flip image vertically.
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: BitmapInfoHeaderCompression.RGB,
	}

	r, _, _ := CreateDIBSection.Call(uintptr(d), uintptr(unsafe.Pointer(&b)), 0, uintptr(unsafe.Pointer(&bitvals)))
	if r == 0 {
		return 0, 0, fmt.Errorf("failed to create DIB section")
	}
	return Bitmap(r), bitvals, nil
}

func (d Device) Select(b Bitmap) (Object, error) {
	r, _, err := SelectObject.Call(d.id(), b.id())
	if r == 0 && !syscall.Errno(0).Is(err) {
		return 0, err
	}
	return Object(r), nil
}

func (d Device) StretchBlt(src Device, size, origin image.Point, scale float64) error {
	r, _, err := stretchBlt.Call(
		d.id(),
		0,
		0,
		uintptr(int(float64(size.X)*scale)),
		uintptr(int(float64(size.Y)*scale)),
		src.id(),
		uintptr(origin.X),
		uintptr(origin.Y),
		uintptr(size.X),
		uintptr(size.Y),
		BitBltRasterOperations.CaptureBLT|BitBltRasterOperations.SrcCopy,
	)
	if r == 0 {
		return err
	}
	return nil
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

func GetLastError() error {
	ret, _, _ := getLastError.Call()
	return fmt.Errorf(syscall.Errno(ret).Error())
	// return uint32(ret)
}

func GetMonitorHandleFromIndex(index int) (hwnd uintptr, err error) {
	ms, err := GetAllMonitors()
	if err != nil {
		return 0, err
	}
	for i, m := range ms.Active {
		if i == index {
			return m.Handle, nil
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

func (m Monitor) String() string {
	return fmt.Sprintf("Monitor=[Handle: %d, Index: %d, Rect: %s", m.Handle, m.Index, m.Rect)
}

func (mi MonitorInfo) String() string {
	return fmt.Sprintf("MonitorInfo=[Rect: %s, Work Area: %s, Flags: %d]", mi.Monitor, mi.WorkArea, mi.Flags)
}

func MoveWindowNoSize(hwnd uintptr, pos image.Point) {
	MoveWindow.Call(hwnd, uintptr(pos.X), uintptr(pos.Y), 0, 0, uintptr(1))
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
	DeleteObject.Call(o.id())
}

func (o Object) id() uintptr {
	return uintptr(o)
}

func (p Point) String() string {
	return fmt.Sprintf("(%d,%d)", p.X, p.Y)
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

func (w Window) Device() (Device, error) {
	r, _, err := getDC.Call(w.HWND())
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

func (w Window) Select(b Bitmap) error {
	r, _, err := SelectObject.Call(w.HWND(), b.id())
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

func (w Window) Capture(rect image.Rectangle, scale float64) (*image.RGBA, error) {
	src, err := w.Device()
	if err != nil {
		return nil, fmt.Errorf("%s: invalid device: %v", w)
	}

	dst, err := src.CreateCompatible()
	if err != nil {
		return nil, fmt.Errorf("%s: incompatible: %v", w, GetLastError())
	}
	defer dst.Delete()

	size := rect.Size()

	info := BitmapInfo{
		BmiHeader: BitmapInfoHeader{
			BiSize:        BitmapInfoHeaderSize,
			BiWidth:       int32(size.X),
			BiHeight:      -int32(size.Y),
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: BitmapInfoHeaderCompression.RGB,
		},
	}

	bitmap, raw, err := info.CreateRGBSection(&dst)
	if err != nil {
		return nil, fmt.Errorf("%s: RGB section %v", w, err)
	}
	defer bitmap.Delete()

	obj, err := dst.Select(bitmap)
	if err != nil {
		return nil, fmt.Errorf("%s: bitmap Select %v", w, err)
	}
	defer obj.Delete()

	err = dst.Copy(src, size, rect, scale)
	if err != nil {
		return nil, fmt.Errorf("%s: bitmap copy %v", w, err)
	}

	data := raw.Slice(size)

	pix := make([]byte, len(data))

	for i := 0; i < len(pix); i += 4 {
		pix[i], pix[i+2], pix[i+1], pix[i+3] = byte(data[i+2]), byte(data[i]), byte(data[i+1]), byte(data[i+3])
	}

	return &image.RGBA{
		Pix:    pix,
		Stride: 4 * size.X,
		Rect:   image.Rect(0, 0, size.X, size.Y),
	}, nil
}
