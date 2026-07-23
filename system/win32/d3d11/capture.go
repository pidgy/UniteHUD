package d3d11

import (
	"fmt"
	"image"
	"time"
	"unsafe"
)

type Desktop struct {
	*device
	*deviceCtx
	*outputDuplication
	descOutput
}

type Window struct {
	*device
	*deviceCtx

	winRTDevice    *unknown
	interopFactory *graphicsCaptureItemInterop
	item           *graphicsCaptureItem
	poolStatics    *captureFramePoolStatics2
	framePool      *captureFramePool
	session        *graphicsCaptureSession
	size           size
}

func NewDesktop(mhwmd uintptr) (*Desktop, error) {
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

	return &Desktop{d, dc, dup, desc}, err
}

func NewWindow(hwnd uintptr) (*Window, error) {
	err := roInitializeApartment()
	if err != nil {
		return nil, fmt.Errorf("winrt init: %w", err)
	}

	if isWindowMinimized(hwnd) {
		restoreWindowForCapture(hwnd)
		defer minimizeWindow(hwnd)
		time.Sleep(150 * time.Millisecond) // let DWM render a real frame
	}

	d, dc, _, err := createDevice(driverHardware, d3d11CreateDeviceBGRASupport)
	if err != nil {
		return nil, fmt.Errorf("create d3d11 device: %w", err)
	}

	winRTDevice, err := createDirect3DDeviceFromD3D11Device(d)
	if err != nil {
		return nil, fmt.Errorf("bridge to winrt device: %w", err)
	}

	interopFactoryRaw, err := activationFactory(
		"Windows.Graphics.Capture.GraphicsCaptureItem",
		&iidGraphicsCaptureItemInterop,
	)
	if err != nil {
		return nil, fmt.Errorf("get IGraphicsCaptureItemInterop factory: %w", err)
	}
	interopFactory := (*graphicsCaptureItemInterop)(interopFactoryRaw)

	item, err := interopFactory.createForWindow(hwnd)
	if err != nil {
		return nil, fmt.Errorf("create capture item for window: %w", err)
	}

	size, err := item.size()
	if err != nil {
		return nil, fmt.Errorf("get capture item size: %w", err)
	}
	if size.width <= 0 || size.height <= 0 {
		return nil, fmt.Errorf("capture item reported empty size (%dx%d)", size.width, size.height)
	}

	poolStaticsRaw, err := activationFactory(
		"Windows.Graphics.Capture.Direct3D11CaptureFramePool",
		&iidCaptureFramePoolStatics2,
	)
	if err != nil {
		return nil, fmt.Errorf("get IDirect3D11CaptureFramePoolStatics2 factory: %w", err)
	}
	poolStatics := (*captureFramePoolStatics2)(poolStaticsRaw)

	framePool, err := poolStatics.createFreeThreaded(winRTDevice, pixelFormatB8G8R8A8UIntNormalized, 2, size)
	if err != nil {
		return nil, fmt.Errorf("create frame pool: %w", err)
	}

	session, err := framePool.createCaptureSession(item)
	if err != nil {
		return nil, fmt.Errorf("create capture session: %w", err)
	}

	if err := session.startCapture(); err != nil {
		return nil, fmt.Errorf("start capture: %w", err)
	}

	w := &Window{
		device:         d,
		deviceCtx:      dc,
		winRTDevice:    winRTDevice,
		interopFactory: interopFactory,
		item:           item,
		poolStatics:    poolStatics,
		framePool:      framePool,
		session:        session,
		size:           size,
	}

	return w, nil
}

func (d *Desktop) Capture(area image.Rectangle) (*image.RGBA, error) {
	resource, info, err := d.acquireNextFrame()
	if err != nil {
		return nil, err
	}
	defer d.releaseFrame()

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

	staging, err := d.texture2D(&desc)
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

	d.copySubresourceRegion(
		staging.resource(),
		0, 0, 0, 0,
		desktop.resource(),
		0,
		&box,
	)

	mapped, err := d.maps(staging.resource(), 0, mapRead, 0)
	if err != nil {
		return nil, err
	}
	defer d.unmap(staging.resource(), 0)

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

func (d *Desktop) Close() {
	d.outputDuplication.release()
	d.deviceCtx.release()
	d.device.release()
}

func (w *Window) Capture() (*image.RGBA, error) {
	var err error

	var frame *captureFrame
	deadline := time.Now().Add(2 * time.Second)
	for frame == nil && time.Now().Before(deadline) {
		frame, err = w.framePool.tryGetNextFrame()
		if err != nil {
			return nil, fmt.Errorf("try get next frame: %w", err)
		}
		if frame == nil {
			time.Sleep(10 * time.Millisecond)
		}
	}
	if frame == nil {
		return nil, fmt.Errorf("timed out waiting for a captured frame")
	}
	defer frame.release()

	surface, err := frame.surface()
	if err != nil {
		return nil, fmt.Errorf("get frame surface: %w", err)
	}
	defer surface.release()

	dxgiAccessRaw, err := queryInterface(unsafe.Pointer(surface), surface.vtbl.QueryInterface, &iidInterfaceAccess)
	if err != nil {
		return nil, fmt.Errorf("QI IDirect3DDxgiInterfaceAccess: %w", err)
	}
	dxgiAccess := (*interfaceAccess)(unsafe.Pointer(dxgiAccessRaw))
	defer dxgiAccess.release()

	tex, err := dxgiAccess.getInterfaceTexture2D(&iidTexture2D)
	if err != nil {
		return nil, fmt.Errorf("get ID3D11Texture2D from surface: %w", err)
	}
	defer tex.release()

	desc, err := tex.desc()
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

	staging, err := w.device.texture2D(&desc)
	if err != nil {
		return nil, err
	}
	defer staging.release()

	w.deviceCtx.copyResource(staging.resource(), tex.resource())

	mapped, err := w.deviceCtx.maps(staging.resource(), 0, mapRead, 0)
	if err != nil {
		return nil, err
	}
	defer w.deviceCtx.unmap(staging.resource(), 0)

	src2 := unsafe.Slice(mapped.PData, mapped.RowPitch*uint32(w.size.height))
	img := image.NewRGBA(image.Rect(0, 0, int(w.size.width), int(w.size.height)))

	for y := range w.size.height {
		copy(
			img.Pix[uint32(y)*uint32(img.Stride):uint32(y)*uint32(img.Stride)+uint32(w.size.width)*4],
			src2[uint32(y)*mapped.RowPitch:uint32(y)*mapped.RowPitch+uint32(w.size.width)*4],
		)
	}

	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+2] = img.Pix[i+2], img.Pix[i]
	}

	return img, nil
}

func (w *Window) Close() {
	w.deviceCtx.release()
	w.device.release()

	w.winRTDevice.release()
	w.interopFactory.release()
	w.item.release()
	w.poolStatics.release()
	w.framePool.release()
	w.session.release()
}
