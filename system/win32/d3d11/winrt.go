package d3d11

import (
	"syscall"
	"unsafe"
)

func (s size) pack() uintptr {
	return uintptr(uint32(s.width)) | uintptr(uint32(s.height))<<32
}

type graphicsCaptureItemInterop struct {
	vtbl *struct {
		unknownVTbl
		createForWindow  uintptr
		createForMonitor uintptr
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

type graphicsCaptureItem struct {
	vtbl *struct {
		inspectableVTbl
		getDisplayName uintptr
		getSize        uintptr
		addClosed      uintptr
		removeClosed   uintptr
	}
}

func (i *graphicsCaptureItem) size() (size, error) {
	var sz size
	r, _, _ := syscall.SyscallN(
		i.vtbl.getSize,
		uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(&sz)),
	)
	if r != 0 {
		return sz, errDXGI{name: "IGraphicsCaptureItem::get_Size", code: uint32(r)}
	}
	return sz, nil
}

func (i *graphicsCaptureItem) release() { release(unsafe.Pointer(i), i.vtbl.Release) }

type graphicsCaptureSession struct {
	vtbl *struct {
		inspectableVTbl
		startCapture uintptr
	}
}

func (s *graphicsCaptureSession) startCapture() error {
	r, _, _ := syscall.SyscallN(s.vtbl.startCapture, uintptr(unsafe.Pointer(s)))
	if r != 0 {
		return errDXGI{name: "IGraphicsCaptureSession::StartCapture", code: uint32(r)}
	}
	return nil
}

func (s *graphicsCaptureSession) release() { release(unsafe.Pointer(s), s.vtbl.Release) }
