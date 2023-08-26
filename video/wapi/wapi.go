package wapi

import (
	"syscall"
)

type (
	Handle  uintptr
	HWND    Handle
	HGDIOBJ Handle
	HDC     Handle
	HBITMAP Handle
)

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd183375.aspx
type BitmapInfo struct {
	BmiHeader BitmapInfoHeader
	BmiColors *RGBQuad
}

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

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd162938.aspx
type RGBQuad struct {
	RgbBlue     byte
	RgbGreen    byte
	RgbRed      byte
	RgbReserved byte
}

// https://learn.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-windowplacement
type WindowPlacement struct {
	Len    uint
	Flags  uint
	Cmd    uint
	Min    Point
	Max    Point
	Normal Rect
	Device Rect
}

var (
	GetDeviceCapsIndex = struct {
		HorzRes,
		VertRes uintptr
	}{
		8,
		10,
	}
	BitmapInfoHeaderCompression = struct {
		RGB uint32
	}{
		RGB: 0,
	}
	CreateDIBSectionError = struct {
		InvalidParameter HBITMAP
	}{
		InvalidParameter: 2,
	}
	CreateDIBSectionUsage = struct {
		RGBColors uint
	}{
		RGBColors: 0,
	}
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
	SetWindowPosFlags = struct {
		None, NoSize, ShowWindow uintptr
	}{
		None:       0x0000,
		NoSize:     0x0001,
		ShowWindow: 0x0040,
	}
	CreateProcessFlags = struct {
		NoWindow uint32
	}{
		NoWindow: 0x08000000,
	}
)

var (
	psapi32                 = syscall.MustLoadDLL("psapi.dll")
	EnumDeviceDrivers       = psapi32.MustFindProc("EnumDeviceDrivers")
	GetDeviceDriverBaseName = psapi32.MustFindProc("GetDeviceDriverBaseNameW")

	user32                       = syscall.MustLoadDLL("user32.dll")
	EnumWindows                  = user32.MustFindProc("EnumWindows")
	GetWindowTextW               = user32.MustFindProc("GetWindowTextW")
	IsWindowVisible              = user32.MustFindProc("IsWindowVisible")
	UpdateWindow                 = user32.MustFindProc("UpdateWindow")
	GetWindowLong                = user32.MustFindProc("GetWindowLongW")
	SetWindowLongPtrW            = user32.MustFindProc("SetWindowLongPtrW")
	GetTopWindow                 = user32.MustFindProc("GetTopWindow")
	FindWindow                   = user32.MustFindProc("FindWindowW")
	MoveWindow                   = user32.MustFindProc("MoveWindow")
	GetClientRect                = user32.MustFindProc("GetClientRect")
	GetDC                        = user32.MustFindProc("GetDC")
	GetWindowDC                  = user32.MustFindProc("GetWindowDC")
	ReleaseDC                    = user32.MustFindProc("ReleaseDC")
	SetThreadDpiAwarenessContext = user32.MustFindProc("GetThreadDpiAwarenessContext")
	SetForegroundWindow          = user32.MustFindProc("SetForegroundWindow")
	SetWindowPlacement           = user32.MustFindProc("SetWindowPlacement")
	GetWindowPlacement           = user32.MustFindProc("GetWindowPlacement")
	GetWindowRect                = user32.MustFindProc("GetWindowRect")
	SetWindowPos                 = user32.MustFindProc("SetWindowPos")
	ShowWindow                   = user32.MustFindProc("ShowWindow")
	GetSystemMetrics             = user32.MustFindProc("GetSystemMetrics")

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

	modShcore              = syscall.NewLazyDLL("shcore.dll")
	SetProcessDpiAwareness = modShcore.NewProc("SetProcessDpiAwareness")

	modKernel32  = syscall.NewLazyDLL("kernel32.dll")
	GetLastError = modKernel32.NewProc("GetLastError")
)
