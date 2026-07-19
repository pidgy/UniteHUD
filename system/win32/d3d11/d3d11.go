// Copied from gioui.org/internal/d3d11, added helpers for window capture.
package d3d11

import (
	"fmt"
	"image"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Capture struct {
	*device
	*deviceCtx
	*outputDuplication
	descOutput
}

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

type point struct {
	x, y int32
}

type rect struct {
	min, max point // Right, Bottom
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

var (
	d3d11             = windows.NewLazySystemDLL("D3D11.dll")
	d3d11CreateDevice = d3d11.NewProc("D3D11CreateDevice")

	iidTexture2D = guid{0x6f15aaf2, 0xd208, 0x4e89, 0x9a, 0xb4, 0x48, 0x95, 0x35, 0xd3, 0x4f, 0x9c}
)

const (
	sdkVersion            = 7
	driverHardware uint32 = 1

	cpuAccessRead = 0x20000
	mapRead       = 1
)

func NewCapture(mhwmd uintptr) (*Capture, error) {
	d, dc, _, err := createDevice(driverHardware, 0)
	if err != nil {
		return nil, err
	}

	dxgi, err := d.idxgi()
	if err != nil {
		return nil, err
	}
	defer dxgi.release()

	adapter, err := dxgi.adapter()
	if err != nil {
		return nil, err
	}
	defer adapter.release()

	var output *output = nil

	for i := uintptr(0); output == nil; i++ {
		output, err = adapter.enumOutputs(i)
		if err != nil {
			if errNotFound(err) {
				break
			}
			return nil, err
		}

		desc, err := output.desc()
		if err != nil {
			return nil, err
		}

		if desc.monitor != mhwmd {
			output.release()
			output = nil
		}

	}
	if output == nil {
		return nil, fmt.Errorf("failed to find output")
	}
	defer output.release()

	desc, err := output.desc()
	if err != nil {
		return nil, err
	}

	output1, err := output.output1()
	if err != nil {
		return nil, err
	}
	defer output1.release()

	dup, err := output1.duplicateOutput(d)
	if err != nil {
		return nil, err
	}
	if dup == nil {
		return nil, fmt.Errorf("nil duplicate")
	}

	_, _, err = dup.acquireNextFrame()
	if err != nil {
		return nil, err
	}
	defer dup.releaseFrame()

	return &Capture{d, dc, dup, desc}, err
}

func (c *Capture) CaptureWindow(area image.Rectangle) (*image.RGBA, error) {
	resource, info, err := c.acquireNextFrame()
	if err != nil {
		return nil, err
	}
	defer c.releaseFrame()

	if info.lastPresentTime == 0 {
		return nil, fmt.Errorf("frame not present")
	}

	desktop, err := resource.texture2D()
	if err != nil {
		return nil, err
	}
	defer desktop.release()

	desc, err := desktop.desc()
	if err != nil {
		return nil, err
	}

	desc.usage = usageStaging
	desc.cpuAccessFlags = cpuAccessRead
	desc.bindFlags = 0
	desc.mipLevels = 1
	desc.arraySize = 1
	desc.miscFlags = 0
	desc.descSample.count = 1

	staging, err := c.texture2D(&desc)
	if err != nil {
		return nil, err
	}
	defer staging.release()

	box := box{
		left:   uint32(area.Min.X),
		top:    uint32(area.Min.Y),
		right:  uint32(area.Max.X),
		bottom: uint32(area.Max.Y),
		front:  0,
		back:   1,
	}

	/*
		box := box{
			Left:   uint32(int32(area.Min.X) + c.desc.DesktopCoordinates.min.x),
			Top:    uint32(int32(area.Min.Y) + c.desc.DesktopCoordinates.min.y),
			Right:  uint32(int32(area.Max.X) + c.desc.DesktopCoordinates.max.x),
			Bottom: uint32(int32(area.Max.Y) + c.desc.DesktopCoordinates.max.y),
			Front:  0,
			Back:   1,
		}
	*/

	c.copySubresourceRegion(
		staging.resource(),
		0, 0, 0, 0,
		desktop.resource(),
		0,
		&box,
	)

	mapped, err := c.maps(staging.resource(), 0, mapRead, 0)
	if err != nil {
		return nil, err
	}
	defer c.unmap(staging.resource(), 0)

	w := uint32(area.Max.X - area.Min.X)
	h := uint32(area.Max.Y - area.Min.Y)

	src := unsafe.Slice(mapped.PData, mapped.RowPitch*h)
	img := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))

	for y := range h {
		copy(
			img.Pix[y*uint32(img.Stride):y*uint32(img.Stride)+w*4],
			src[y*mapped.RowPitch:y*mapped.RowPitch+w*4],
		)
	}
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+2] = img.Pix[i+2], img.Pix[i]
	}

	return img, nil
}

func (c *Capture) Close() {
	c.outputDuplication.release()
	c.deviceCtx.release()
	c.device.release()
}

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
		sdkVersion,                        // SDKVersion
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

func (t *texture2D) release() { release(unsafe.Pointer(t), t.vtbl.Release) }

func (t *texture2D) resource() *resource { return (*resource)(unsafe.Pointer(t)) }
