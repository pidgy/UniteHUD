package wapi

import (
	"fmt"
	"syscall"
)

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd183375.aspx
type BitmapInfo struct {
	BmiHeader BitmapInfoHeader
	BmiColors *RGBQuad
} //

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd183376.aspx
type BitmapInfoHeader struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

// https://learn.microsoft.com/en-us/previous-versions/dd162805(v=vs.85)
type Point struct {
	X, Y int32
}

// Windows RECT structure
type Rect struct {
	Left, Top, Right, Bottom int32
}

type Rect2 struct {
	Left, Top, Right, Bottom int
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd162938.aspx
type RGBQuad struct {
	RgbBlue     byte
	RgbGreen    byte
	RgbRed      byte
	RgbReserved byte
}

// https://learn.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-windowinfo
type WindowInfo struct {
	/*
	  DWORD cbSize;
	  RECT  rcWindow;
	  RECT  rcClient;
	  DWORD dwStyle;
	  DWORD dwExStyle;
	  DWORD dwWindowStatus;
	  UINT  cxWindowBorders;
	  UINT  cyWindowBorders;
	  ATOM  atomWindowType;
	  WORD  wCreatorVersion;
	*/
	Size           uint32 // DWORD.
	Window         Rect   // RECT.
	Client         Rect   // RECT.
	Style          uint32 // DWORD.
	ExStyle        uint32 // DWORD.
	Status         uint32 // DWORD.
	BordersX       uint   // UINT.
	BordersY       uint   // UINT.
	Type           uint16 // ATOM.
	CreatorVersion uint16 // WORD.
}

// https://learn.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-windowplacement
type WindowPlacement struct {
	Len         uint
	Flags       uint
	ShowCommand uint
	Min         Point
	Max         Point
	Normal      Rect
	Device      Rect
}

// https://learn.microsoft.com/en-us/windows/win32/hidpi/dpi-awareness-context
type SetProcessDpiAwarenessContext int

const (
	Unaware SetProcessDpiAwarenessContext = iota
	SystemAware
	PerMonitorAware
	PerMonitorAwareV2
	UnawareGDIScaled
)

var (
	BitBltRasterOperations = struct {
		SrcCopy,
		CaptureBLT,
		SrcPaint,
		PatCopy,
		PatPaint,
		MergePaint,
		SrcInvert uintptr
	}{
		SrcCopy:    0x00CC0020,
		CaptureBLT: 0x40000000,
		SrcPaint:   0x00EE0086,
		PatCopy:    0x00F00021,
		PatPaint:   0x00FB0A09,
		MergePaint: 0x00BB0226,
		SrcInvert:  0x00660046,
	}
	BitmapInfoHeaderCompression = struct {
		RGB uint32
	}{
		RGB: 0,
	}
	CreateDIBSectionError = struct {
		InvalidParameter uintptr
	}{
		InvalidParameter: 2,
	}
	CreateDIBSectionUsage = struct {
		RGBColors uint
	}{
		RGBColors: 0,
	}
	CreateProcessFlags = struct {
		NoWindow uint32
	}{
		NoWindow: 0x08000000,
	}
	DwmWindowAttributeFlags = struct {
		Cloaked uintptr
	}{
		Cloaked: 0x000E,
	}
	GetDeviceCapsIndex = struct {
		HorzRes,
		VertRes uintptr
	}{
		8,
		10,
	}
	GetWindowFlags = struct {
		Child,
		EnabledPopUp,
		First,
		Next,
		Last,
		Prev uintptr
	}{
		Child:        5,
		EnabledPopUp: 6,
		First:        0,
		Next:         2,
		Last:         1,
		Prev:         3,
	}
	GetWindowLongFlags = struct {
		Style uintptr
	}{
		Style: ^(uintptr(16) - 1), // -16
	}
	SetWindowPosFlags = struct {
		None,
		NoSize,
		Hide,
		Show uintptr
	}{
		None:   0x0000,
		NoSize: 0x0001,
		Hide:   0x0080,
		Show:   0x0040,
	}
	ShowWindowFlags = struct {
		Hide,
		Normal,
		ShowNormal,
		ShowMinimized,
		ShowMaximized,
		Maximize,
		ShowNoActivate,
		Show,
		Minimize,
		ShowMinNoActive,
		ShowNA,
		Restore,
		ShowDefault,
		ForceMinimize uint
	}{
		Hide:            0,
		Normal:          1,
		ShowNormal:      1,
		ShowMinimized:   2,
		ShowMaximized:   3,
		Maximize:        3,
		ShowNoActivate:  4,
		Show:            5,
		Minimize:        6,
		ShowMinNoActive: 7,
		ShowNA:          8,
		Restore:         9,
		ShowDefault:     10,
		ForceMinimize:   11,
	}
	WindowStyleFlags = struct {
		Caption,
		MinimizeBox,
		MaximizeBox,
		Overlapped,
		SysMenu,
		ThickFrame,
		Tiled,
		Visible uint32

		OverlappedWindow uint32
	}{
		Caption:     0x00C00000,
		MinimizeBox: 0x00020000,
		MaximizeBox: 0x00010000,
		Overlapped:  0x00000000,
		SysMenu:     0x00080000,
		ThickFrame:  0x00040000,
		Tiled:       0x00000000,
		Visible:     0x10000000,
	}
)

var (
	psapi32                 = syscall.MustLoadDLL("psapi.dll")
	EnumDeviceDrivers       = psapi32.MustFindProc("EnumDeviceDrivers")
	GetDeviceDriverBaseName = psapi32.MustFindProc("GetDeviceDriverBaseNameW")

	user32                       = syscall.MustLoadDLL("user32.dll")
	EnumWindows                  = user32.MustFindProc("EnumWindows")
	FindWindow                   = user32.MustFindProc("FindWindowW")
	GetClientRect                = user32.MustFindProc("GetClientRect")
	GetDC                        = user32.MustFindProc("GetDC")
	GetDesktopWindow             = user32.MustFindProc("GetDesktopWindow")
	GetSystemMetrics             = user32.MustFindProc("GetSystemMetrics")
	GetTopWindow                 = user32.MustFindProc("GetTopWindow")
	GetWindow                    = user32.MustFindProc("GetWindow")
	GetWindowDC                  = user32.MustFindProc("GetWindowDC")
	GetWindowInfo                = user32.MustFindProc("GetWindowInfo")
	GetWindowLong                = user32.MustFindProc("GetWindowLongW")
	GetWindowPlacement           = user32.MustFindProc("GetWindowPlacement")
	GetWindowRect                = user32.MustFindProc("GetWindowRect")
	GetWindowTextW               = user32.MustFindProc("GetWindowTextW")
	IsWindowVisible              = user32.MustFindProc("IsWindowVisible")
	MoveWindow                   = user32.MustFindProc("MoveWindow")
	ReleaseDC                    = user32.MustFindProc("ReleaseDC")
	SetForegroundWindow          = user32.MustFindProc("SetForegroundWindow")
	SetThreadDpiAwarenessContext = user32.MustFindProc("GetThreadDpiAwarenessContext")
	SetWindowLongPtrW            = user32.MustFindProc("SetWindowLongPtrW")
	SetWindowPlacement           = user32.MustFindProc("SetWindowPlacement")
	SetWindowPos                 = user32.MustFindProc("SetWindowPos")
	ShowWindow                   = user32.MustFindProc("ShowWindow")
	UpdateWindow                 = user32.MustFindProc("UpdateWindow")

	gdi32                  = syscall.MustLoadDLL("gdi32.dll")
	BitBlt                 = gdi32.MustFindProc("BitBlt")
	StretchBlt             = gdi32.MustFindProc("StretchBlt")
	CreateCompatibleBitmap = gdi32.MustFindProc("CreateCompatibleBitmap")
	CreateCompatibleDC     = gdi32.MustFindProc("CreateCompatibleDC")
	CreateDIBSection       = gdi32.MustFindProc("CreateDIBSection")
	DeleteDC               = gdi32.MustFindProc("DeleteDC")
	DeleteObject           = gdi32.MustFindProc("DeleteObject")
	GetDeviceCaps          = gdi32.MustFindProc("GetDeviceCaps")
	GetDIBits              = gdi32.MustFindProc("GetDIBits")
	SelectObject           = gdi32.MustFindProc("SelectObject")
	GetClipBox             = gdi32.MustFindProc("GetClipBox")

	dwmapi                = syscall.MustLoadDLL("dwmapi.dll")
	DwmGetWindowAttribute = dwmapi.MustFindProc("DwmGetWindowAttribute")

	modShcore              = syscall.NewLazyDLL("shcore.dll")
	setProcessDpiAwareness = modShcore.NewProc("SetProcessDpiAwareness")

	modKernel32             = syscall.NewLazyDLL("kernel32.dll")
	GetLastError            = modKernel32.NewProc("GetLastError")
	setThreadExecutionState = modKernel32.NewProc("SetThreadExecutionState")
)

func init() {
	/*
		(WS_OVERLAPPED | WS_CAPTION | WS_SYSMENU | WS_THICKFRAME | WS_MINIMIZEBOX | WS_MAXIMIZEBOX)
	*/
	WindowStyleFlags.OverlappedWindow = WindowStyleFlags.Overlapped |
		WindowStyleFlags.Caption | WindowStyleFlags.SysMenu | WindowStyleFlags.ThickFrame | WindowStyleFlags.MinimizeBox | WindowStyleFlags.MaximizeBox
}

func (p Point) String() string {
	return fmt.Sprintf("(%d,%d)", p.X, p.Y)
}

func (r Rect) String() string {
	return fmt.Sprintf("[%d,%d,%d,%d]", r.Left, r.Top, r.Right, r.Bottom)
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

type ThreadExecutionState int

const (
	ThreadExecutionStateAwayModeRequired ThreadExecutionState = 0x00000040
	ThreadExecutionStateContinuous       ThreadExecutionState = 0x80000000
	ThreadExecutionStateDisplayRequired  ThreadExecutionState = 0x00000002
	ThreadExecutionStateSystemRequired   ThreadExecutionState = 0x00000001
	ThreadExecutionStateUserPresent      ThreadExecutionState = 0x00000004
)

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

func (w *WindowPlacement) String() string {
	return fmt.Sprintf("len: %d, flags: %d, cmd: %d, min: %s, max: %s, normal: %s, device: %s",
		w.Len, w.Flags, w.ShowCommand, w.Min, w.Max, w.Normal, w.Device)
}
