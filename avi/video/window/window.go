package window

import (
	"fmt"
	"image"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/pkg/errors"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/system/wapi"
)

var (
	Sources = []string{}
	lock    = &sync.Mutex{}
)

// Capture captures the desired area from a Window and returns an image.
func Capture() (*image.RGBA, error) {
	w, err := wapi.NewWindow(config.Current.Video.Capture.Window.Name)
	if err != nil {
		return nil, err
	}

	rect, err := w.Dimensions()
	if err != nil {
		return nil, errors.Wrap(err, "dimensions")
	}

	return CaptureRect(w, rect)
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

	bitmapData := unsafe.Pointer(uintptr(0))
	bitmap, _, err := wapi.CreateDIBSection.Call(
		dst,
		uintptr(unsafe.Pointer(&bitmapInfo)),
		0,
		uintptr(unsafe.Pointer(&bitmapData)),
		0, 0,
	)
	if bitmap == 0 {
		return nil, fmt.Errorf("Failed to create bitmap for \"%s\" window", config.Current.Video.Capture.Window.Name)
	}

	defer wapi.DeleteObject.Call(bitmap)

	// Select the object and paint it.
	wapi.SelectObject.Call(dst, bitmap)

	var ret uintptr
	switch config.Current.Scale {
	case 1:
		ret, _, _ = wapi.BitBlt.Call(
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
	default: // Scaled.
		ret, _, _ = wapi.StretchBlt.Call(
			dst,
			0,
			0,
			uintptr(int(float64(width)*config.Current.Scale)),
			uintptr(int(float64(height)*config.Current.Scale)),
			src,
			uintptr(rect.Min.X),
			uintptr(rect.Min.Y),
			uintptr(width),
			uintptr(height),
			wapi.BitBltRasterOperations.CaptureBLT|wapi.BitBltRasterOperations.SrcCopy,
		)
	}
	if ret == 0 {
		notify.Error("Window: Failed to capture \"%s\" window", config.Current.Video.Capture.Window.Name)
		return nil, fmt.Errorf("bitblt returned: %d", ret)
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

func IsOpen() bool {
	return !Lost()
}

func Lost() bool {
	return config.Current.Video.Capture.Window.Lost != ""
}

func Open() error {
	go sync.OnceFunc(func() {
		for ; ; time.Sleep(time.Second * 5) {
			windows, err := list()
			if err != nil {
				notify.Warn("[Window] Failed to list windows (%v)", err)
				continue
			}

			Sources = windows
		}
	})

	windows, err := list()
	if err != nil {
		return err
	}

	Sources = windows

	for _, win := range windows {
		if win == config.Current.Video.Capture.Window.Name {
			config.Current.Video.Capture.Window.Lost = ""
			return nil
		}
	}

	notify.Error("[Window] <ini:failed:find> \"%s\"", config.Current.Video.Capture.Window.Name)

	config.Current.Video.Capture.Window.Lost = config.Current.Video.Capture.Window.Name
	config.Current.Video.Capture.Window.Name = config.MainDisplay

	return nil
}

// var reattachAttempts = 0

// func Reattach() error {
// 	if !Lost() {
// 		return nil
// 	}

// 	max := 5
// 	windows, err := list()
// 	if err != nil {
// 		return err
// 	}

// 	for _, win := range windows {
// 		if win == config.Current.Video.Capture.Window.Lost {
// 			config.Current.Video.Capture.Window.Name = win

// 			notify.Announce("[Window] Found \"%s\" window", config.Current.Video.Capture.Window.Name)
// 			config.Current.Video.Capture.Window.Lost = ""
// 			reattachAttempts = 0

// 			return nil
// 		}
// 	}

// 	reattachAttempts++
// 	if reattachAttempts == max {
// 		config.Current.Video.Capture.Window.Name = config.MainDisplay
// 		config.Current.Video.Capture.Window.Lost = ""
// 		reattachAttempts = 0
// 	}

// 	return nil
// }

func list() ([]string, error) {
	lock.Lock()
	defer lock.Unlock()

	Sources = []string{}

	err := wapi.EnumerateWindows(func(w wapi.Window) (stop bool) {
		// Ignore windows that are visible to a user.
		if w.InfoStatus() == wapi.WindowInfoStatusVisible {
			return false // Don't stop.
		}

		// Ignore windows without valid titles or handles.
		name, err := w.Title()
		if err != nil {
			return false // Don't stop.
		}

		// Ignore the projector window to prevent recursive painting.
		if name == config.ProjectorWindow {
			return false // Don't stop.
		}

		Sources = append(Sources, name)

		return false // Don't stop.
	})
	if err != nil {
		return nil, err
	}

	return Sources, nil
}
