// Copied from gioui.org/internal/d3d11, added helpers for window capture.
package d3d11

import (
	"syscall"
	"unsafe"
)

const (
	usageDefault = uint32(0)
	usageStaging = uint32(3)
)

type box struct {
	left   uint32
	top    uint32
	front  uint32
	right  uint32
	bottom uint32
	back   uint32
}

type captureFrame struct {
	vtbl *struct {
		inspectableVTbl
		getSurface            uintptr
		getSystemRelativeTime uintptr
		getContentSize        uintptr
	}
}

type captureFramePool struct {
	vtbl *struct {
		inspectableVTbl
		recreate             uintptr
		tryGetNextFrame      uintptr
		addFrameArrived      uintptr
		removeFrameArrived   uintptr
		createCaptureSession uintptr
		getDispatcherQueue   uintptr
	}
}

type captureFramePoolStatics2 struct {
	vtbl *struct {
		inspectableVTbl
		createFreeThreaded uintptr
	}
}

type descTexture2D struct {
	width      uint32
	height     uint32
	mipLevels  uint32
	arraySize  uint32
	format     uint32
	descSample struct {
		count   uint32
		quality uint32
	}
	usage          uint32
	bindFlags      uint32
	cpuAccessFlags uint32
	miscFlags      uint32
}

type device struct {
	vtbl *struct {
		unknownVTbl
		createBuffer                         uintptr
		createTexture1D                      uintptr
		createTexture2D                      uintptr
		createTexture3D                      uintptr
		createShaderResourceView             uintptr
		createUnorderedAccessView            uintptr
		createRenderTargetView               uintptr
		createDepthStencilView               uintptr
		createInputLayout                    uintptr
		createVertexShader                   uintptr
		createGeometryShader                 uintptr
		createGeometryShaderWithStreamOutput uintptr
		createPixelShader                    uintptr
		createHullShader                     uintptr
		createDomainShader                   uintptr
		createComputeShader                  uintptr
		createClassLinkage                   uintptr
		createBlendState                     uintptr
		createDepthStencilState              uintptr
		createRasterizerState                uintptr
		createSamplerState                   uintptr
		createQuery                          uintptr
		createPredicate                      uintptr
		createCounter                        uintptr
		createDeferredContext                uintptr
		openSharedResource                   uintptr
		checkFormatSupport                   uintptr
		checkMultisampleQualityLevels        uintptr
		checkCounterInfo                     uintptr
		checkCounter                         uintptr
		checkFeatureSupport                  uintptr
		getPrivateData                       uintptr
		setPrivateData                       uintptr
		setPrivateDataInterface              uintptr
		getFeatureLevel                      uintptr
		getCreationFlags                     uintptr
		getDeviceRemovedReason               uintptr
		getImmediateContext                  uintptr
		setExceptionMode                     uintptr
		getExceptionMode                     uintptr
	}
}

type deviceCtx struct {
	vtbl *struct {
		unknownVTbl
		getDevice                                 uintptr
		getPrivateData                            uintptr
		setPrivateData                            uintptr
		setPrivateDataInterface                   uintptr
		vsSetConstantBuffers                      uintptr
		psSetShaderResources                      uintptr
		psSetShader                               uintptr
		psSetSamplers                             uintptr
		vsSetShader                               uintptr
		drawIndexed                               uintptr
		draw                                      uintptr
		maps                                      uintptr
		unmap                                     uintptr
		psSetConstantBuffers                      uintptr
		iaSetInputLayout                          uintptr
		iaSetVertexBuffers                        uintptr
		iaSetIndexBuffer                          uintptr
		drawIndexedInstanced                      uintptr
		drawInstanced                             uintptr
		gsSetConstantBuffers                      uintptr
		gsSetShader                               uintptr
		iaSetPrimitiveTopology                    uintptr
		vsSetShaderResources                      uintptr
		vsSetSamplers                             uintptr
		begin                                     uintptr
		end                                       uintptr
		getData                                   uintptr
		setPredication                            uintptr
		gsSetShaderResources                      uintptr
		gsSetSamplers                             uintptr
		omSetRenderTargets                        uintptr
		omSetRenderTargetsAndUnorderedAccessViews uintptr
		omSetBlendState                           uintptr
		omSetDepthStencilState                    uintptr
		soSetTargets                              uintptr
		drawAuto                                  uintptr
		drawIndexedInstancedIndirect              uintptr
		drawInstancedIndirect                     uintptr
		dispatch                                  uintptr
		dispatchIndirect                          uintptr
		rsSetState                                uintptr
		rsSetViewports                            uintptr
		rsSetScissorRects                         uintptr
		copySubresourceRegion                     uintptr
		copyResource                              uintptr
		updateSubresource                         uintptr
		copyStructureCount                        uintptr
		clearRenderTargetView                     uintptr
		clearUnorderedAccessViewUint              uintptr
		clearUnorderedAccessViewFloat             uintptr
		clearDepthStencilView                     uintptr
		generateMips                              uintptr
		setResourceMinLOD                         uintptr
		getResourceMinLOD                         uintptr
		resolveSubresource                        uintptr
		executeCommandList                        uintptr
		hsSetShaderResources                      uintptr
		hsSetShader                               uintptr
		hsSetSamplers                             uintptr
		hsSetConstantBuffers                      uintptr
		dsSetShaderResources                      uintptr
		dsSetShader                               uintptr
		dsSetSamplers                             uintptr
		dsSetConstantBuffers                      uintptr
		csSetShaderResources                      uintptr
		csSetUnorderedAccessViews                 uintptr
		csSetShader                               uintptr
		csSetSamplers                             uintptr
		csSetConstantBuffers                      uintptr
		vsGetConstantBuffers                      uintptr
		psGetShaderResources                      uintptr
		psGetShader                               uintptr
		psGetSamplers                             uintptr
		vsGetShader                               uintptr
		psGetConstantBuffers                      uintptr
		iaGetInputLayout                          uintptr
		iaGetVertexBuffers                        uintptr
		iaGetIndexBuffer                          uintptr
		gsGetConstantBuffers                      uintptr
		gsGetShader                               uintptr
		iaGetPrimitiveTopology                    uintptr
		vsGetShaderResources                      uintptr
		vsGetSamplers                             uintptr
		getPredication                            uintptr
		gsGetShaderResources                      uintptr
		gsGetSamplers                             uintptr
		omGetRenderTargets                        uintptr
		omGetRenderTargetsAndUnorderedAccessViews uintptr
		omGetBlendState                           uintptr
		omGetDepthStencilState                    uintptr
		soGetTargets                              uintptr
		rsGetState                                uintptr
		rsGetViewports                            uintptr
		rsGetScissorRects                         uintptr
		hsGetShaderResources                      uintptr
		hsGetShader                               uintptr
		hsGetSamplers                             uintptr
		hsGetConstantBuffers                      uintptr
		dsGetShaderResources                      uintptr
		dsGetShader                               uintptr
		dsGetSamplers                             uintptr
		dsGetConstantBuffers                      uintptr
		csGetShaderResources                      uintptr
		csGetUnorderedAccessViews                 uintptr
		csGetShader                               uintptr
		csGetSamplers                             uintptr
		csGetConstantBuffers                      uintptr
		clearState                                uintptr
		flush                                     uintptr
		getType                                   uintptr
		getContextFlags                           uintptr
		finishCommandList                         uintptr
	}
}

type framePool struct {
	vtbl *struct {
		// IUnknown.
		unknownVTbl

		// IInspectable.
		getIIds             uintptr
		getRuntimeClassName uintptr
		getTrustLevel       uintptr

		// IGraphicsCaptureItem.
		// DisplayName   uintptr
		// Size          uintptr
		// add_Closed    uintptr
		// remove_Closed uintptr

		recreate             uintptr
		tryGetNextFrame      uintptr
		add_FrameArrived     uintptr
		remove_FrameArrived  uintptr
		createCaptureSession uintptr
		get_DispatcherQueue  uintptr
	}
}

type guid struct {
	Data1   uint32
	Data2   uint16
	Data3   uint16
	Data4_0 uint8
	Data4_1 uint8
	Data4_2 uint8
	Data4_3 uint8
	Data4_4 uint8
	Data4_5 uint8
	Data4_6 uint8
	Data4_7 uint8
}

type mappedSubresource struct {
	PData      *byte
	RowPitch   uint32
	DepthPitch uint32
}

type texture2D struct {
	vtbl *struct {
		unknownVTbl

		// id3d11DeviceChild.
		getDevice               uintptr
		getPrivateData          uintptr
		setPrivateData          uintptr
		setPrivateDataInterface uintptr

		// id3d11Resource.
		getType             uintptr
		setEvictionPriority uintptr
		getEvictionPriority uintptr

		// id3d11Texture2D.
		getDesc2 uintptr
		getDesc  uintptr
	}
}

func (f *captureFrame) surface() (*unknown, error) {
	var surf *unknown
	r, _, _ := syscall.SyscallN(
		f.vtbl.getSurface,
		uintptr(unsafe.Pointer(f)),
		uintptr(unsafe.Pointer(&surf)),
	)
	if r != 0 {
		return nil, errDXGI{name: "IDirect3D11CaptureFrame::get_Surface", code: uint32(r)}
	}
	return surf, nil
}

func (f *captureFrame) release() { release(unsafe.Pointer(f), f.vtbl.Release) }

func (p *captureFramePool) tryGetNextFrame() (*captureFrame, error) {
	var frame *captureFrame
	r, _, _ := syscall.SyscallN(
		p.vtbl.tryGetNextFrame,
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&frame)),
	)
	if r != 0 {
		return nil, errDXGI{name: "IDirect3D11CaptureFramePool::TryGetNextFrame", code: uint32(r)}
	}
	return frame, nil // frame may legitimately be nil if none is ready yet
}

func (p *captureFramePool) createCaptureSession(item *graphicsCaptureItem) (*graphicsCaptureSession, error) {
	var session *graphicsCaptureSession
	r, _, _ := syscall.SyscallN(
		p.vtbl.createCaptureSession,
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(item)),
		uintptr(unsafe.Pointer(&session)),
	)
	if r != 0 {
		return nil, errDXGI{name: "IDirect3D11CaptureFramePool::CreateCaptureSession", code: uint32(r)}
	}
	return session, nil
}

func (p *captureFramePool) release() { release(unsafe.Pointer(p), p.vtbl.Release) }

func (s *captureFramePoolStatics2) createFreeThreaded(device *unknown, pixelFormat int32, numberOfBuffers int32, size size) (*captureFramePool, error) {
	var pool *captureFramePool
	r, _, _ := syscall.SyscallN(
		s.vtbl.createFreeThreaded,
		uintptr(unsafe.Pointer(s)),
		uintptr(unsafe.Pointer(device)),
		uintptr(pixelFormat),
		uintptr(numberOfBuffers),
		size.pack(),
		uintptr(unsafe.Pointer(&pool)),
	)
	if r != 0 {
		return nil, errDXGI{name: "IDirect3D11CaptureFramePoolStatics2::CreateFreeThreaded", code: uint32(r)}
	}
	return pool, nil
}

func (s *captureFramePoolStatics2) release() { release(unsafe.Pointer(s), s.vtbl.Release) }

func createDevice(driverType uint32, flags uint32) (*device, *deviceCtx, uint32, error) {
	var (
		dev     *device
		ctx     *deviceCtx
		featLvl uint32
	)
	r, _, _ := d3d11CreateDevice.Call(
		0,                                 // pAdapter
		uintptr(driverType),               // driverType
		0,                                 // Software
		uintptr(flags),                    // Flags
		0,                                 // pFeatureLevels
		0,                                 // FeatureLevels
		d3d11SDKVersion,                   // SDKVersion
		uintptr(unsafe.Pointer(&dev)),     // ppDevice
		uintptr(unsafe.Pointer(&featLvl)), // pFeatureLevel
		uintptr(unsafe.Pointer(&ctx)),     // ppImmediateContext
	)
	if r != 0 {
		return nil, nil, 0, errDXGI{name: "D3D11::CreateDevice", code: uint32(r)}
	}
	return dev, ctx, featLvl, nil
}

func (d *device) texture2D(desc *descTexture2D) (*texture2D, error) {
	var tex *texture2D
	r, _, _ := syscall.SyscallN(
		d.vtbl.createTexture2D,
		uintptr(unsafe.Pointer(d)),
		uintptr(unsafe.Pointer(desc)),
		0, // pInitialData
		uintptr(unsafe.Pointer(&tex)),
		0, 0,
	)
	if r != 0 {
		return nil, errDXGI{name: "ID3D11Device::CreateTexture2D", code: uint32(r)}
	}
	return tex, nil
}

func (d *device) idxgi() (*deviceDXGI, error) {
	dx, err := queryInterface(unsafe.Pointer(d), d.vtbl.QueryInterface, &iidDevice)
	return (*deviceDXGI)(unsafe.Pointer(dx)), err
}

func (d *device) release() { release(unsafe.Pointer(d), d.vtbl.Release) }

func (d *deviceCtx) copyResource(dst, src *resource) {
	syscall.SyscallN(
		d.vtbl.copyResource,
		uintptr(unsafe.Pointer(d)),
		uintptr(unsafe.Pointer(dst)),
		uintptr(unsafe.Pointer(src)),
	)
}

func (d *deviceCtx) copySubresourceRegion(dst *resource, dstSubresource, dstX, dstY, dstZ uint32, src *resource, srcSubresource uint32, srcBox *box) {
	syscall.SyscallN(
		d.vtbl.copySubresourceRegion,
		uintptr(unsafe.Pointer(d)),
		uintptr(unsafe.Pointer(dst)),
		uintptr(dstSubresource),
		uintptr(dstX),
		uintptr(dstY),
		uintptr(dstZ),
		uintptr(unsafe.Pointer(src)),
		uintptr(srcSubresource),
		uintptr(unsafe.Pointer(srcBox)),
	)
}

func (d *deviceCtx) maps(resource *resource, subResource, mapType, mapFlags uint32) (mappedSubresource, error) {
	var resMap mappedSubresource
	r, _, _ := syscall.SyscallN(
		d.vtbl.maps,
		uintptr(unsafe.Pointer(d)),
		uintptr(unsafe.Pointer(resource)),
		uintptr(subResource),
		uintptr(mapType),
		uintptr(mapFlags),
		uintptr(unsafe.Pointer(&resMap)),
	)
	if r != 0 {
		return resMap, errDXGI{name: "ID3D11DeviceContext::ContextMap", code: uint32(r)}
	}
	return resMap, nil
}

func (d *deviceCtx) release() { release(unsafe.Pointer(d), d.vtbl.Release) }

func (d *deviceCtx) unmap(resource *resource, subResource uint32) {
	syscall.SyscallN(
		d.vtbl.unmap,
		uintptr(unsafe.Pointer(d)),
		uintptr(unsafe.Pointer(resource)),
		uintptr(subResource),
	)
}

func (t *texture2D) desc() (descTexture2D, error) {
	desc := descTexture2D{}
	r, _, _ := syscall.SyscallN(
		t.vtbl.getDesc,
		uintptr(unsafe.Pointer(t)),
		uintptr(unsafe.Pointer(&desc)),
	)
	if r != 0 {
		return desc, errDXGI{name: "ID3D11Texture2D::GetDesc", code: uint32(r)}
	}

	return desc, nil
}

func (t *texture2D) release()            { release(unsafe.Pointer(t), t.vtbl.Release) }
func (t *texture2D) resource() *resource { return (*resource)(unsafe.Pointer(t)) }
