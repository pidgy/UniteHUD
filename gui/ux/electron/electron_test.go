package electron

import (
	"fmt"
	"image"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/system/save"
	"github.com/pidgy/unitehud/system/wapi"
)

func TestNewWindow(t *testing.T) {
	for i := 0; i < 30; i++ {
		time.Sleep(time.Second)

		now := time.Now()

		w, err := wapi.NewWindow("Calculator")
		if err != nil {
			fmt.Printf("failed to find window: %v\n", err)
			continue
		}

		if !w.Visible() {
			fmt.Printf("window is not visible\n")
			continue
		}

		t, err := w.Title()
		if err != nil {
			fmt.Printf("failed to find window title: %v\n", err)
			continue
		}

		r, err := w.Dimensions()
		if err != nil {
			fmt.Printf("failed to find window dimensions: %v\n", err)
			continue
		}

		img, err := CaptureRect(w, r)
		if err != nil {
			fmt.Printf("failed to capture window: %v\n", err)
			continue
		}

		err = save.PNG(img, "capture.png")
		if err != nil {
			fmt.Printf("failed to save window: %v\n", err)
			continue
		}

		fmt.Printf("found window: %v: %s (%s) (took: %dms)\n", w.HWND(), t, img.Rect, time.Since(now).Milliseconds())
	}
}

func CaptureRect(w wapi.Window, rect image.Rectangle) (*image.RGBA, error) {
	handle := w.HWND()

	// Get the device context for screenshotting.
	src, _, err := wapi.GetDC.Call(uintptr(handle))
	if src == 0 {
		return nil, fmt.Errorf("failed to prepare screen capture: %s", err)
	}
	defer wapi.ReleaseDC.Call(0, src)

	// Grab a compatible DC for drawing.
	dst, _, err := wapi.CreateCompatibleDC.Call(src)
	if dst == 0 {
		return nil, fmt.Errorf("failed to create DC for drawing: %s", err)
	}
	defer wapi.DeleteDC.Call(dst)

	// Determine the width/height of our capture.
	width := rect.Dx()
	height := rect.Dy()

	// Get the bitmap we're going to draw onto.
	var bitmapInfo wapi.BitmapInfo
	bitmapInfo.BmiHeader = wapi.BitmapInfoHeader{
		BiSize:        uint32(reflect.TypeOf(bitmapInfo.BmiHeader).Size()),
		BiWidth:       int32(width),
		BiHeight:      -int32(height), // Negative value will flip image vertically.
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: wapi.BitmapInfoHeaderCompression.RGB,
	}

	bitmapData := uintptr(0)
	bitmap, _, err := wapi.CreateDIBSection.Call(
		dst,
		uintptr(unsafe.Pointer(&bitmapInfo)),
		0,
		uintptr(unsafe.Pointer(&bitmapData)),
		0, 0,
	)
	if bitmap == 0 {
		return nil, fmt.Errorf("Failed to create bitmap for window")
	}

	defer wapi.DeleteObject.Call(bitmap)

	// Select the object and paint it.
	wapi.SelectObject.Call(dst, bitmap)

	r, _, _ := wapi.BitBlt.Call(
		dst,
		0,
		0,
		uintptr(width),
		uintptr(height),
		src,
		uintptr(rect.Min.X),
		uintptr(rect.Min.Y),
		wapi.BitBltRasterOperations.CaptureBLT|wapi.BitBltRasterOperations.SrcCopy,
	)
	if r == 0 {
		notify.Error("Window: <ini:f:capture> window")
		return nil, fmt.Errorf("bitblt returned: %d", r)
	}

	// Convert the bitmap to an image.Image. We first start by directly
	// creating a slice. This is unsafe but we know the underlying structure
	// directly.
	var slice []byte
	sliceHdr := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	sliceHdr.Data = uintptr(bitmapData)
	sliceHdr.Len = width * height * 4
	sliceHdr.Cap = sliceHdr.Len

	// Using the raw data, grab the RGBA data and transform it into an image.RGBA
	imageBytes := make([]byte, len(slice))
	for i := 0; i < len(imageBytes); i += 4 {
		imageBytes[i], imageBytes[i+2], imageBytes[i+1], imageBytes[i+3] =
			slice[i+2], slice[i], slice[i+1], slice[i+3]
	}

	return &image.RGBA{
		Pix:    imageBytes,
		Stride: 4 * width,
		Rect: image.Rect(
			0,
			0,
			width,
			height,
		),
	}, nil
}
