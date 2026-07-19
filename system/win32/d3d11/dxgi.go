package d3d11

import (
	"fmt"
	"syscall"
	"unsafe"
)

type adapter struct {
	vtbl *struct {
		unknownVTbl
		setPrivateData          uintptr
		setPrivateDataInterface uintptr
		getPrivateData          uintptr
		getParent               uintptr
		enumOutputs             uintptr
		getDesc                 uintptr
		checkInterfaceSupport   uintptr
		getDesc1                uintptr
	}
}

type descOutput struct {
	deviceName         [32]uint16
	desktopCoordinates rect
	attachedToDesktop  uint32
	rotation           uint32
	monitor            uintptr
}

type deviceDXGI struct {
	vtbl *struct {
		unknownVTbl
		setPrivateData          uintptr
		setPrivateDataInterface uintptr
		getPrivateData          uintptr
		getParent               uintptr
		getAdapter              uintptr
		createSurface           uintptr
		queryResourceResidency  uintptr
		setGPUThreadPriority    uintptr
		getGPUThreadPriority    uintptr
	}
}

type errDXGI struct {
	name string
	code uint32
}

type outduplPointerPosition struct {
	position point
	visible  uint32
}

type outduplFrameInfo struct {
	lastPresentTime           int64
	lastMouseUpdateTime       int64
	accumulatedFrames         uint32
	rectsCoalesced            uint32
	protectedContentMaskedOut uint32
	pointerPosition           outduplPointerPosition
	totalMetadataBufferSize   uint32
	pointerShapeBufferSize    uint32
}

type output struct {
	vtbl *struct {
		unknownVTbl
		setPrivateData          uintptr
		setPrivateDataInterface uintptr
		getPrivateData          uintptr
		getParent               uintptr

		getDesc                     uintptr
		getDisplayModeList          uintptr
		findClosestMatchingMode     uintptr
		WaitForVBlank               uintptr
		takeOwnership               uintptr
		releaseOwnership            uintptr
		getGammaControlCapabilities uintptr
		setGammaControl             uintptr
		getGammaControl             uintptr
		setDisplaySurface           uintptr
		getDisplaySurfaceData       uintptr
		getFrameStatistics          uintptr
	}
}

type output1 struct {
	vtbl *struct {
		unknownVTbl
		setPrivateData          uintptr
		setPrivateDataInterface uintptr
		getPrivateData          uintptr
		getParent               uintptr

		getDesc                     uintptr
		getDisplayModeList          uintptr
		findClosestMatchingMode     uintptr
		waitForVBlank               uintptr
		takeOwnership               uintptr
		releaseOwnership            uintptr
		getGammaControlCapabilities uintptr
		setGammaControl             uintptr
		getGammaControl             uintptr
		setDisplaySurface           uintptr
		getDisplaySurfaceData       uintptr
		getFrameStatistics          uintptr

		getDisplayModeList1      uintptr
		findClosestMatchingMode1 uintptr
		getDisplaySurfaceData1   uintptr
		duplicateOutput          uintptr
	}
}

type outputDuplication struct {
	vtbl *struct {
		unknownVTbl
		setPrivateData          uintptr
		setPrivateDataInterface uintptr
		getPrivateData          uintptr
		getParent               uintptr
		getDesc                 uintptr
		acquireNextFrame        uintptr
		getFrameDirtyRects      uintptr
		getFrameMoveRects       uintptr
		getFramePointerShape    uintptr
		mapDesktopSurface       uintptr
		unMapDesktopSurface     uintptr
		releaseFrame            uintptr
	}
}

type resource struct {
	vtbl *struct {
		unknownVTbl
	}
}

var (
	iidDevice  = guid{0x54ec77fa, 0x1377, 0x44e6, 0x8c, 0x32, 0x88, 0xfd, 0x5f, 0x44, 0xc8, 0x4c}
	iidOutput1 = guid{0x00cddea8, 0x939b, 0x4b83, 0xa3, 0x40, 0xa6, 0x85, 0x22, 0x66, 0x66, 0xcc}

	// dxgi = windows.NewLazySystemDLL("dxgi.dll")
)

func (a *adapter) enumOutputs(i uintptr) (*output, error) {
	var output *output
	r, _, _ := syscall.SyscallN(
		a.vtbl.enumOutputs,
		uintptr(unsafe.Pointer(a)),
		i,
		uintptr(unsafe.Pointer(&output)),
	)
	if r != 0 {
		return nil, errDXGI{name: "IDXGIAdapter::EnumOutputs", code: uint32(r)}
	}

	return output, nil
}

func (a *adapter) release() { release(unsafe.Pointer(a), a.vtbl.Release) }

func (d *deviceDXGI) adapter() (*adapter, error) {
	var adapter *adapter
	r, _, _ := syscall.SyscallN(
		d.vtbl.getAdapter,
		uintptr(unsafe.Pointer(d)),
		uintptr(unsafe.Pointer(&adapter)),
		0,
	)
	if r != 0 {
		return nil, errDXGI{name: "IDXGIDevice::GetAdapter", code: uint32(r)}
	}
	return adapter, nil
}

func (d *deviceDXGI) release() { release(unsafe.Pointer(d), d.vtbl.Release) }

func (e errDXGI) Error() string {
	switch e.code {
	case 0x887A002B:
		return e.name + "::DXGIERROR_ACCESS_DENIED"
	case 0x887A0026:
		return e.name + "::DXGIERROR_ACCESS_LOST"
	case 0x887A0036:
		return e.name + "::DXGIERROR_ALREADY_EXISTS"
	case 0x887A002A:
		return e.name + "::DXGIERROR_CANNOT_PROTECT_CONTENT"
	case 0x887A0006:
		return e.name + "::DXGIERROR_DEVICE_HUNG"
	case 0x887A0005:
		return e.name + "::DXGIERROR_DEVICE_REMOVED"
	case 0x887A0007:
		return e.name + "::DXGIERROR_DEVICE_RESET"
	case 0x887A0020:
		return e.name + "::DXGIERROR_DRIVER_INTERNAL_ERROR"
	case 0x887A000B:
		return e.name + "::DXGIERROR_FRAME_STATISTICS_DISJOINT"
	case 0x887A000C:
		return e.name + "::DXGIERROR_GRAPHICS_VIDPN_SOURCE_IN_USE"
	case 0x887A0001:
		return e.name + "::DXGIERROR_INVALID_CALL"
	case 0x887A0003:
		return e.name + "::DXGIERROR_MORE_DATA"
	case 0x887A002C:
		return e.name + "::DXGIERROR_NAME_ALREADY_EXISTS"
	case 0x887A0021:
		return e.name + "::DXGIERROR_NONEXCLUSIVE"
	case 0x887A0022:
		return e.name + "::DXGIERROR_NOT_CURRENTLY_AVAILABLE"
	case 0x887A0002:
		return e.name + "::DXGIERROR_NOT_FOUND"
	case 0x887A0023:
		return e.name + "::DXGIERROR_REMOTE_CLIENT_DISCONNECTED"
	case 0x887A0024:
		return e.name + "::DXGIERROR_REMOTE_OUTOFMEMORY"
	case 0x887A0029:
		return e.name + "::DXGIERROR_RESTRICT_TO_OUTPUT_STALE"
	case 0x887A002D:
		return e.name + "::DXGIERROR_SDK_COMPONENT_MISSING"
	case 0x887A0028:
		return e.name + "::DXGIERROR_SESSION_DISCONNECTED"
	case 0x887A0004:
		return e.name + "::DXGIERROR_UNSUPPORTED"
	case 0x887A0027:
		return e.name + "::DXGIERROR_WAIT_TIMEOUT"
	case 0x887A000A:
		return e.name + "::DXGIERROR_WAS_STILL_DRAWING"
	default:
		return fmt.Sprintf("::DXGIERROR_UNKNOWN_0x%08X", uint32(e.code))
	}
}

func errNotFound(err error) bool {
	e, ok := err.(errDXGI)
	if !ok {
		return false
	}
	return e.code == 0x887A0002
}

func (o *output) desc() (descOutput, error) {
	desc := descOutput{}
	r, _, _ := syscall.SyscallN(
		o.vtbl.getDesc,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&desc)),
	)
	if r != 0 {
		return descOutput{}, errDXGI{name: "IDXGIOutput1::GetDesc", code: uint32(r)}
	}

	return desc, nil
}

func (o *output) release() { release(unsafe.Pointer(o), o.vtbl.Release) }

func (o *output1) duplicateOutput(d *device) (*outputDuplication, error) {
	var d2 *outputDuplication
	r, _, _ := syscall.SyscallN(
		o.vtbl.duplicateOutput,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(d)),
		uintptr(unsafe.Pointer(&d2)),
	)
	if r != 0 {
		return nil, errDXGI{name: "IDXGIOutput1::DuplicateOutput", code: uint32(r)}
	}
	return d2, nil
}

func (o *outputDuplication) acquireNextFrame() (*resource, outduplFrameInfo, error) {
	var info outduplFrameInfo
	var resource *resource
	r, _, _ := syscall.SyscallN(
		o.vtbl.acquireNextFrame,
		uintptr(unsafe.Pointer(o)),
		uintptr(100), // Timeout in ms.
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Pointer(&resource)),
	)
	if r != 0 {
		return nil, info, errDXGI{name: "IDXGIOutputDuplication::AcquireNextFrame", code: uint32(r)}
	}
	return resource, info, nil
}

func (o *outputDuplication) release() { release(unsafe.Pointer(o), o.vtbl.Release) }

func (o *outputDuplication) releaseFrame() {
	syscall.SyscallN(
		o.vtbl.releaseFrame,
		uintptr(unsafe.Pointer(o)),
	)
}

func (o *output) output1() (*output1, error) {
	o1, err := queryInterface(unsafe.Pointer(o), o.vtbl.QueryInterface, &iidOutput1)
	if err != nil {
		return nil, err
	}
	return (*output1)(unsafe.Pointer(o1)), nil
}

func (o *output1) release() { release(unsafe.Pointer(o), o.vtbl.Release) }

func (r *resource) texture2D() (*texture2D, error) {
	t, err := queryInterface(unsafe.Pointer(r), r.vtbl.QueryInterface, &iidTexture2D)
	return (*texture2D)(unsafe.Pointer(t)), err
}
