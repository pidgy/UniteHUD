package win32

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const. Then remove this comment.

import (
	"syscall"
)

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd183375.aspx
type (
	BitmapInfo struct {
		BmiHeader BitmapInfoHeader
		BmiColors *RGBQuad
	}

	// http://msdn.microsoft.com/en-us/library/windows/desktop/dd183376.aspx
	BitmapInfoHeader struct {
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

	// https://learn.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-monitorinfo
	MonitorInfo struct {
		cbSize   uint32
		Monitor  Rect
		WorkArea Rect
		Flags    uint32
	}

	// https://learn.microsoft.com/en-us/previous-versions/dd162805(v=vs.85)
	Point struct {
		X, Y int32
	}

	// Windows RECT structure.
	Rect struct {
		Left   int32 // Min.X
		Top    int32 // Min.Y
		Right  int32 // Max.X
		Bottom int32 // Max.Y
	}

	// http://msdn.microsoft.com/en-us/library/windows/desktop/dd162938.aspx
	RGBQuad struct {
		RgbBlue     byte
		RgbGreen    byte
		RgbRed      byte
		RgbReserved byte
	}

	WindowStyle uint32

	// https://learn.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-windowinfo
	WindowInfo struct {
		Size           uint32       // DWORD.
		Window         Rect         // RECT.
		Client         Rect         // RECT.
		Style          WindowStyle  // DWORD.
		ExStyle        uint32       // DWORD.
		Status         WindowStatus // DWORD.
		BordersX       uint         // UINT.
		BordersY       uint         // UINT.
		Type           uint16       // ATOM.
		CreatorVersion uint16       // WORD.
	}

	// https://learn.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-windowinfo
	WindowStatus uint32

	// https://learn.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-windowplacement
	WindowPlacement struct {
		Len         uint
		Flags       uint
		ShowCommand uint
		Min         Point
		Max         Point
		Normal      Rect
		Device      Rect
	}

	WindowPos struct {
		HWND            syscall.Handle
		HWNDInsertAfter syscall.Handle
		x               int32
		y               int32
		cx              int32
		cy              int32
		flags           uint32
	}
)

const (
	WindowStatusNotVisible WindowStatus = iota
	WindowStatusVisible
	WindowStatusUnknown
)

var (
	bitBltRasterOperations = struct {
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
	}{0}

	CreateDIBSectionError = struct {
		InvalidParameter uintptr
	}{2}

	CreateDIBSectionUsage = struct {
		RGBColors uint
	}{0}

	CreateProcessFlags = struct {
		NoWindow uint32
	}{0x08000000}

	CreateToolhelp32SnapshotFlags = struct {
		SnapProcess uint32
	}{
		SnapProcess: 0x00000002,
	}

	DwmWindowAttributeFlags = struct {
		Cloaked                uintptr
		UseImmersiveDarkMode10 uintptr // Windows 10.
		UseImmersiveDarkMode11 uintptr // Windows 11.
	}{
		Cloaked:                0x000E,
		UseImmersiveDarkMode10: 19,
		UseImmersiveDarkMode11: 20,
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

	HWNDInsertAfterFlags = struct {
		Bottom,
		NoTopMost,
		Top,
		TopMost int32
	}{
		Bottom:    1,
		NoTopMost: -2,
		Top:       0,
		TopMost:   -1,
	}

	MonitorFromWindowOptions = struct {
		DefaultToPrimary uintptr
	}{
		DefaultToPrimary: 1,
	}

	SetWindowPosFlags = struct {
		None,
		NoSize,
		Hide,
		Show,
		NoMove,
		ShowWindow uintptr
	}{
		None:       0x0000,
		NoSize:     0x0001,
		Hide:       0x0080,
		Show:       0x0040,
		NoMove:     0x0002,
		ShowWindow: 0x0040,
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
		ForceMinimize uintptr
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

	// WindowStyles = struct {
	// 	Caption,
	// 	MinimizeBox,
	// 	MaximizeBox,
	// 	Overlapped,
	// 	SysMenu,
	// 	ThickFrame,
	// 	Tiled,
	// 	Visible uintptr

	// 	OverlappedWindow uintptr
	// }{
	// 	Caption:     0x00C00000,
	// 	MinimizeBox: 0x00020000,
	// 	MaximizeBox: 0x00010000,
	// 	Overlapped:  0x00000000,
	// 	SysMenu:     0x00080000,
	// 	ThickFrame:  0x00040000,
	// 	Tiled:       0x00000000,
	// 	Visible:     0x10000000,
	// }

	WindowStyles = struct {
		Border,
		Caption,
		Child,
		ChildWindow,
		MinimizeBox,
		MaximizeBox,
		Overlapped,
		SysMenu,
		ThickFrame,
		Tiled,
		Visible,
		Maximize,
		OverlappedWindow WindowStyle
	}{
		Border:      0x00800000,
		Caption:     0x00C00000,
		Child:       0x40000000,
		MinimizeBox: 0x00020000,
		MaximizeBox: 0x00010000,
		Overlapped:  0x00000000,
		SysMenu:     0x00080000,
		ThickFrame:  0x00040000,
		Tiled:       0x00000000,
		Visible:     0x10000000,
		Maximize:    0x01000000,
	}

	SystemParametersInfoOptions = struct {
		GetWorkArea uintptr
	}{
		GetWorkArea: 0x0030,
	}

	// https://learn.microsoft.com/en-us/windows/win32/api/wingdi/nf-wingdi-setstretchbltmode.
	SetStretchBltMode = struct {
		// Performs a Boolean AND operation using the color values for the eliminated and existing pixels. If the bitmap is a monochrome bitmap, this mode preserves black pixels at the expense of white pixels.
		BlackOnWhite,
		// Performs a Boolean OR operation using the color values for the eliminated and existing pixels. If the bitmap is a monochrome bitmap, this mode preserves white pixels at the expense of black pixels.
		WhiteOnBlack,
		// Deletes the pixels. This mode deletes all eliminated lines of pixels without trying to preserve their information.
		ColorOnColor,
		// Maps pixels from the source rectangle into blocks of pixels in the destination rectangle. The average color over the destination block of pixels approximates the color of the source pixels.
		// After setting the HALFTONE stretching mode, an application must call the SetBrushOrgEx function to set the brush origin. If it fails to do so, brush misalignment occurs.
		HalfTone uintptr
	}{
		1, 2, 3, 4,
	}

	printWindowFlags = struct {
		ClientOnly        uintptr
		RenderFullContent uintptr
	}{
		ClientOnly:        1,
		RenderFullContent: 2,
	}

	getDIBitsUsage = struct {
		RGBColors,
		PALColors,
		PALIndicies uintptr
	}{0, 1, 2}

	enumDeviceDrivers       = psapi32.MustFindProc("EnumDeviceDrivers")
	getDeviceDriverBaseName = psapi32.MustFindProc("GetDeviceDriverBaseNameW")

	getDC                        = user32.MustFindProc("GetDC")
	getMonitorInfoW              = user32.MustFindProc("GetMonitorInfoW")
	getWindowRect                = user32.MustFindProc("GetWindowRect")
	getWindowTextW               = user32.MustFindProc("GetWindowTextW")
	isWindowVisible              = user32.MustFindProc("IsWindowVisible")
	moveWindow                   = user32.MustFindProc("MoveWindow")
	printWindow                  = user32.MustFindProc("PrintWindow")
	rReleaseDC                   = user32.MustFindProc("ReleaseDC")
	setForegroundWindow          = user32.MustFindProc("SetForegroundWindow")
	setThreadDpiAwarenessContext = user32.MustFindProc("GetThreadDpiAwarenessContext")
	setWindowLongPtrA            = user32.MustFindProc("SetWindowLongPtrA")
	setWindowLongPtrW            = user32.MustFindProc("SetWindowLongPtrW")
	setWindowPlacement           = user32.MustFindProc("SetWindowPlacement")
	setWindowPos                 = user32.MustFindProc("SetWindowPos")
	enumWindows                  = user32.MustFindProc("EnumWindows") // https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-enumwindows
	enumDisplayMonitors          = user32.MustFindProc("EnumDisplayMonitors")
	findWindow                   = user32.MustFindProc("FindWindowW")
	getClientRect                = user32.MustFindProc("GetClientRect")
	getWindowInfo                = user32.MustFindProc("GetWindowInfo")
	monitorFromWindow            = user32.MustFindProc("MonitorFromWindow")
	showWindow                   = user32.MustFindProc("ShowWindow")
	systemParametersInfoA        = user32.MustFindProc("SystemParametersInfoA")

	createCompatibleBitmap = gdi32.MustFindProc("CreateCompatibleBitmap")
	createCompatibleDC     = gdi32.MustFindProc("CreateCompatibleDC")
	createDIBSection       = gdi32.MustFindProc("CreateDIBSection")
	deleteDC               = gdi32.MustFindProc("DeleteDC")
	deleteObject           = gdi32.MustFindProc("DeleteObject")
	getClipBox             = gdi32.MustFindProc("GetClipBox")
	getDIBits              = gdi32.MustFindProc("GetDIBits")
	getDeviceCaps          = gdi32.MustFindProc("GetDeviceCaps")
	selectObject           = gdi32.MustFindProc("SelectObject")
	bitBlt                 = gdi32.MustFindProc("BitBlt")
	stretchBlt             = gdi32.MustFindProc("StretchBlt")
	setStretchBltMode      = gdi32.MustFindProc("SetStretchBltMode")

	dwmGetWindowAttribute = dwmapi.MustFindProc("DwmGetWindowAttribute")
	dwmSetWindowAttribute = dwmapi.MustFindProc("DwmSetWindowAttribute")

	getLastError            = modKernel32.NewProc("GetLastError")
	setThreadExecutionState = modKernel32.NewProc("SetThreadExecutionState")

	transparentBlt = msimg32.MustFindProc("TransparentBlt")

	dwmapi      = syscall.MustLoadDLL("dwmapi.dll")
	gdi32       = syscall.MustLoadDLL("gdi32.dll")
	modShcore   = syscall.NewLazyDLL("shcore.dll")
	modKernel32 = syscall.NewLazyDLL("kernel32.dll")
	psapi32     = syscall.MustLoadDLL("psapi.dll")
	user32      = syscall.MustLoadDLL("user32.dll")
	msimg32     = syscall.MustLoadDLL("msimg32.dll")

	setProcessDpiAwareness = modShcore.NewProc("SetProcessDpiAwareness")
)

func init() {
	/*
		(WS_OVERLAPPED | WS_CAPTION | WS_SYSMENU | WS_THICKFRAME | WS_MINIMIZEBOX | WS_MAXIMIZEBOX)
	*/
	WindowStyles.OverlappedWindow = WindowStyles.Overlapped |
		WindowStyles.Caption | WindowStyles.SysMenu | WindowStyles.ThickFrame | WindowStyles.MinimizeBox | WindowStyles.MaximizeBox
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

type ThreadExecutionState int

const (
	ThreadExecutionStateAwayModeRequired ThreadExecutionState = 0x00000040
	ThreadExecutionStateContinuous       ThreadExecutionState = 0x80000000
	ThreadExecutionStateDisplayRequired  ThreadExecutionState = 0x00000002
	ThreadExecutionStateSystemRequired   ThreadExecutionState = 0x00000001
	ThreadExecutionStateUserPresent      ThreadExecutionState = 0x00000004
)
