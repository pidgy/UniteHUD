package d3d11

import (
	"syscall"
	"unsafe"
)

type graphicsCaptureItem struct {
	vtbl *struct {
		inspectableVTbl
		getDisplayName uintptr
		getSize        uintptr
		addClosed      uintptr
		removeClosed   uintptr
	}
}

type graphicsCaptureItemInterop struct {
	vtbl *struct {
		unknownVTbl
		createForWindow  uintptr
		createForMonitor uintptr
	}
}

type graphicsCaptureSession struct {
	vtbl *struct {
		inspectableVTbl
		startCapture uintptr
	}
}

func (g *graphicsCaptureItemInterop) createForWindow(hwnd uintptr) (*graphicsCaptureItem, error) {
	var item *graphicsCaptureItem
	r, _, _ := syscall.SyscallN(
		g.vtbl.createForWindow,
		uintptr(unsafe.Pointer(g)),
		hwnd,
		uintptr(unsafe.Pointer(&iidGraphicsCaptureItem)),
		uintptr(unsafe.Pointer(&item)),
	)
	if r != 0 {
		return nil, errDXGI{name: "IGraphicsCaptureItemInterop::CreateForWindow", code: uint32(r)}
	}
	return item, nil
}

func (g *graphicsCaptureItemInterop) release() { release(unsafe.Pointer(g), g.vtbl.Release) }

func (g *graphicsCaptureItem) size() (size, error) {
	var sz size
	r, _, _ := syscall.SyscallN(
		g.vtbl.getSize,
		uintptr(unsafe.Pointer(g)),
		uintptr(unsafe.Pointer(&sz)),
	)
	if r != 0 {
		return sz, errDXGI{name: "IGraphicsCaptureItem::get_Size", code: uint32(r)}
	}
	return sz, nil
}

func (g *graphicsCaptureItem) release() { release(unsafe.Pointer(g), g.vtbl.Release) }

func (g *graphicsCaptureSession) startCapture() error {
	r, _, _ := syscall.SyscallN(g.vtbl.startCapture, uintptr(unsafe.Pointer(g)))
	if r != 0 {
		return errDXGI{name: "IGraphicsCaptureSession::StartCapture", code: uint32(r)}
	}
	return nil
}

func (g *graphicsCaptureSession) release() { release(unsafe.Pointer(g), g.vtbl.Release) }
