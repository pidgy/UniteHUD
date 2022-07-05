package window

import "syscall"

const (
	// GetDeviceCaps constants from Wingdi.h
	deviceCaps_HORZRES    = 8
	deviceCaps_VERTRES    = 10
	deviceCaps_LOGPIXELSX = 88
	deviceCaps_LOGPIXELSY = 90

	// BitBlt constants
	bitBlt_SRCCOPY = 0x00CC0020
)

var (
	user32              = syscall.MustLoadDLL("user32.dll")
	procEnumWindows     = user32.MustFindProc("EnumWindows")
	procGetWindowTextW  = user32.MustFindProc("GetWindowTextW")
	procIsWindowVisible = user32.MustFindProc("IsWindowVisible")

	modUser32         = syscall.NewLazyDLL("User32.dll")
	procFindWindow    = modUser32.NewProc("FindWindowW")
	procMoveWindow    = modUser32.NewProc("MoveWindow")
	procGetClientRect = modUser32.NewProc("GetClientRect")
	procGetDC         = modUser32.NewProc("GetDC")
	procReleaseDC     = modUser32.NewProc("ReleaseDC")

	modGdi32                   = syscall.NewLazyDLL("Gdi32.dll")
	procBitBlt                 = modGdi32.NewProc("BitBlt")
	procCreateCompatibleBitmap = modGdi32.NewProc("CreateCompatibleBitmap")
	procCreateCompatibleDC     = modGdi32.NewProc("CreateCompatibleDC")
	procCreateDIBSection       = modGdi32.NewProc("CreateDIBSection")
	procDeleteDC               = modGdi32.NewProc("DeleteDC")
	procDeleteObject           = modGdi32.NewProc("DeleteObject")
	procGetDeviceCaps          = modGdi32.NewProc("GetDeviceCaps")
	procSelectObject           = modGdi32.NewProc("SelectObject")

	modShcore                  = syscall.NewLazyDLL("Shcore.dll")
	procSetProcessDpiAwareness = modShcore.NewProc("SetProcessDpiAwareness")

	modKernel32      = syscall.NewLazyDLL("kernel32.dll")
	procGetLastError = modKernel32.NewProc("GetLastError")
)

// Windows RECT structure
type windowsRect struct {
	Left, Top, Right, Bottom int32
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd183375.aspx
type windowsBitmapInfo struct {
	BmiHeader windowsBitmapInfoHeader
	BmiColors *windowsRGBQuad
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd183376.aspx
type windowsBitmapInfoHeader struct {
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
type windowsRGBQuad struct {
	RgbBlue     byte
	RgbGreen    byte
	RgbRed      byte
	RgbReserved byte
}
