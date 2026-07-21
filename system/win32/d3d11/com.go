package d3d11

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type inspectableVTbl struct {
	unknownVTbl
	getIids             uintptr
	getRuntimeClassName uintptr
	getTrustLevel       uintptr
}

type point struct {
	x, y int32
}

type rect struct {
	min, max point // Right, Bottom
}

type size struct {
	width, height int32
}

type unknown struct {
	vtbl *struct {
		unknownVTbl
	}
}

type unknownVTbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
}

const (
	roInitMultiThreaded = 1

	swShowNoActivate = 4
	swMinimize       = 6

	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpNoActivate = 0x0010

	d3d11CreateDeviceBGRASupport = 0x20

	pixelFormatB8G8R8A8UIntNormalized = 87 // DirectXPixelFormat enum value

	d3d11SDKVersion        = 7
	driverHardware  uint32 = 1

	cpuAccessRead = 0x20000
	mapRead       = 1
)

var (
	d3d11                               = windows.MustLoadDLL("D3D11.dll")
	d3d11CreateDevice                   = d3d11.MustFindProc("D3D11CreateDevice")
	d3d11CreateDirect3D11DeviceFromDXGI = d3d11.MustFindProc("CreateDirect3D11DeviceFromDXGIDevice")

	iidTexture2D                  = guid{0x6f15aaf2, 0xd208, 0x4e89, 0x9a, 0xb4, 0x48, 0x95, 0x35, 0xd3, 0x4f, 0x9c}
	iidGraphicsCaptureItemInterop = guid{0x3628e81b, 0x3cac, 0x4c60, 0xb7, 0xf4, 0x23, 0xce, 0x0e, 0x0c, 0x33, 0x56}
	iidGraphicsCaptureItem        = guid{0x79c3f95b, 0x31f7, 0x4ec2, 0xa4, 0x64, 0x63, 0x2e, 0xf5, 0xd3, 0x07, 0x60}
	iidCaptureFramePoolStatics2   = guid{0x589b103f, 0x6bbc, 0x5df5, 0xa9, 0x91, 0x02, 0xe2, 0x8b, 0x3b, 0x66, 0xd5}
	iidInterfaceAccess            = guid{0xa9b3d012, 0x3df2, 0x4ee3, 0xb8, 0xd1, 0x86, 0x95, 0xf4, 0x57, 0xd3, 0xc1}
	iidDirect3DDevice             = guid{0xa37624ab, 0x8d5f, 0x4650, 0x9d, 0x3e, 0x9e, 0xae, 0x3d, 0x9b, 0xc6, 0x70}

	combase                    = windows.MustLoadDLL("combase.dll")
	procRoInitialize           = combase.MustFindProc("RoInitialize")
	procRoGetActivationFactory = combase.MustFindProc("RoGetActivationFactory")
	procWindowsCreateString    = combase.MustFindProc("WindowsCreateString")
	procWindowsDeleteString    = combase.MustFindProc("WindowsDeleteString")

	user32           = windows.MustLoadDLL("user32.dll")
	procIsIconic     = user32.MustFindProc("IsIconic")
	procShowWindow   = user32.MustFindProc("ShowWindow")
	procSetWindowPos = user32.MustFindProc("SetWindowPos")
)

// activationFactory calls RoGetActivationFactory for the given runtime
// class, requesting the interface identified by iid directly (this is what
// winrt::get_activation_factory<Class, Interface>() does under the hood).
func activationFactory(className string, iid *guid) (unsafe.Pointer, error) {
	hstr, err := createHString(className)
	if err != nil {
		return nil, err
	}
	defer deleteHString(hstr)

	var factory unsafe.Pointer
	r, _, _ := procRoGetActivationFactory.Call(
		hstr,
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&factory)),
	)
	if r != 0 {
		return nil, errDXGI{name: "RoGetActivationFactory(" + className + ")", code: uint32(r)}
	}
	return factory, nil
}

// createDirect3DDeviceFromD3D11Device bridges a raw D3D11 device to the
// WinRT IDirect3DDevice the capture APIs expect.
func createDirect3DDeviceFromD3D11Device(d *device) (*unknown, error) {
	dxgiDev, err := d.idxgi()
	if err != nil {
		return nil, err
	}
	defer dxgiDev.release()

	var inspectable *unknown
	r, _, _ := d3d11CreateDirect3D11DeviceFromDXGI.Call(
		uintptr(unsafe.Pointer(dxgiDev)),
		uintptr(unsafe.Pointer(&inspectable)),
	)
	if r != 0 {
		return nil, errDXGI{name: "CreateDirect3D11DeviceFromDXGIDevice", code: uint32(r)}
	}

	winrtDevice, err := queryInterface(unsafe.Pointer(inspectable), inspectable.vtbl.QueryInterface, &iidDirect3DDevice)
	if err != nil {
		inspectable.release()
		return nil, err
	}
	inspectable.release()
	return (*unknown)(unsafe.Pointer(winrtDevice)), nil
}

func createHString(s string) (uintptr, error) {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		return 0, err
	}
	var hstr uintptr
	r, _, _ := procWindowsCreateString.Call(
		uintptr(unsafe.Pointer(p)),
		uintptr(len(s)),
		uintptr(unsafe.Pointer(&hstr)),
	)
	if r != 0 {
		return 0, errDXGI{name: "WindowsCreateString", code: uint32(r)}
	}
	return hstr, nil
}

func deleteHString(h uintptr) {
	if h != 0 {
		procWindowsDeleteString.Call(h)
	}
}

func isWindowMinimized(hwnd uintptr) bool {
	r, _, _ := procIsIconic.Call(hwnd)
	return r != 0
}

func minimizeWindow(hwnd uintptr) {
	procShowWindow.Call(hwnd, uintptr(swMinimize))
}

func queryInterface(obj unsafe.Pointer, queryInterfaceMethod uintptr, guid *guid) (*unknown, error) {
	var ref *unknown
	r, _, _ := syscall.SyscallN(
		queryInterfaceMethod,
		uintptr(obj),
		uintptr(unsafe.Pointer(guid)),
		uintptr(unsafe.Pointer(&ref)),
	)
	if r != 0 {
		return nil, errDXGI{name: "D3D11::QueryInterface", code: uint32(r)}
	}
	return ref, nil
}

func release(obj unsafe.Pointer, releaseMethod uintptr) {
	syscall.SyscallN(
		releaseMethod,
		uintptr(obj),
		0,
		0,
	)
}

func restoreWindowForCapture(hwnd uintptr) {
	procShowWindow.Call(hwnd, uintptr(swShowNoActivate))
	procSetWindowPos.Call(hwnd, 1 /*HWND_BOTTOM*/, 0, 0, 0, 0,
		uintptr(swpNoMove|swpNoSize|swpNoActivate))
}

func roInitializeApartment() error {
	r, _, _ := procRoInitialize.Call(uintptr(roInitMultiThreaded))
	// S_OK (0) and S_FALSE (1, already initialized) are both fine.
	if r != 0 && r != 1 {
		return errDXGI{name: "RoInitialize", code: uint32(r)}
	}
	return nil
}

func (u *unknown) release() { release(unsafe.Pointer(u), u.vtbl.Release) }
