package proc

import (
	"syscall"
)

type (
	HANDLE  uintptr
	HWND    HANDLE
	HGDIOBJ HANDLE
	HDC     HANDLE
	HBITMAP HANDLE
)

const (
	HorzRes          = 8
	VertRes          = 10
	InvalidParameter = 2
	BIRGBCompression = 0
	DIBRGBColors     = 0
	SrcCopy          = 0x00CC0020
	CaptureBLT       = 0x40000000
	SrcPaint         = 0x00EE0086
	PatCopy          = 0x00F00021
	PatPaint         = 0x00FB0A09
	MergePaint       = 0x00BB0226
	SrcInvert        = 0x00660046
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
	SetWindowLongA               = user32.MustFindProc("SetWindowLongA")
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

	gdi32                  = syscall.MustLoadDLL("gdi32.dll")
	BitBlt                 = gdi32.MustFindProc("BitBlt")
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

// Windows RECT structure
type WindowsRect struct {
	Left, Top, Right, Bottom int32
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd183375.aspx
type WindowsBitmapInfo struct {
	BmiHeader WindowsBitmapInfoHeader
	BmiColors *WindowsRGBQuad
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd183376.aspx
type WindowsBitmapInfoHeader struct {
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

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd162938.aspx
type WindowsRGBQuad struct {
	RgbBlue     byte
	RgbGreen    byte
	RgbRed      byte
	RgbReserved byte
}

// https://learn.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-windowplacement
type WindowsPlacement struct {
	Len    uint
	Flags  uint
	Cmd    uint
	Min    WindowsPoint
	Max    WindowsPoint
	Normal WindowsRect
	Device WindowsRect
}

// https://learn.microsoft.com/en-us/previous-versions/dd162805(v=vs.85)
type WindowsPoint struct {
	X, Y int32
}
