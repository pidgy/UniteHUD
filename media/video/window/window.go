package window

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const. Then remove this comment.

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/media/video/monitor"
	"github.com/pidgy/unitehud/system/wapi"
)

var (
	Sources       = []wapi.WindowInfoEx{}
	ErrFailedFind = fmt.Errorf("failed to find window")

	once = sync.OnceFunc(func() {
		for ; ; time.Sleep(time.Second * 5) {
			err := list()
			if err != nil {
				notify.Error("<ini:f:find> active windows (%v)", err)
			}
		}
	})

	fullscreen = image.Rect(-1, -1, -1, -1)
)

func Active() (w wapi.Window) {
	// return slice.Map(Sources, func(i wapi.WindowInfoEx) wapi.Window {
	// 	if i.Title == config.Current.Video.Capture.Window.Name {
	// 		return i.Window
	// 	}
	// 	return wapi.Window(0)
	// })[0]
	_ = slices.ContainsFunc(Sources, func(w2 wapi.WindowInfoEx) bool {
		if w2.Title == config.Current.Video.Capture.Window.Name {
			w = w2.Window
		}
		return w != wapi.Window(0)
	})
	return
}

// Capture captures the desired area from a Window and returns an image.
func Capture() (*image.RGBA, error) {
	return CaptureRect(fullscreen)
}

func CaptureRect(r image.Rectangle) (*image.RGBA, error) {
	a := Active()

	if r.Eq(fullscreen) {
		w, err := a.Rect()
		if err != nil {
			return nil, err
		}
		r = w.Image()

		notify.Debug("[Window] Dimensions: %s", r.Size())
	}

	return Active().Capture(r, config.Current.Scale)
}

/*
func CaptureRect(r image.Rectangle) (*image.RGBA, error) {
	// Get the device context for screenshotting.
	src, _, err := wapi.GetDC.Call(Active().HWND())
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
	width := r.Dx()
	height := r.Dy()

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
			uintptr(r.Min.X),
			uintptr(r.Min.Y),
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
			uintptr(r.Min.X),
			uintptr(r.Min.Y),
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
*/

func IsActive() bool {
	return Active() != wapi.Window(0)
}

func Open() error {
	go once()

	err := list()
	if err != nil {
		return err
	}

	if config.Current.Video.Capture.Window.Name == config.NoDeviceName {
		return nil
	}

	if !slices.ContainsFunc(Sources, func(w wapi.WindowInfoEx) bool { return w.Title == config.Current.Video.Capture.Window.Name }) {
		notify.Error("[Window] <ini:f:find> \"%s\"", config.Current.Video.Capture.Window.Name)
		return ErrFailedFind
	}

	return nil
}

func Resolution() image.Point {
	return monitor.DefaultResolution.Size()
}

func list() error {
	windows, err := wapi.GetAllWindows()
	if err != nil {
		return err
	}

	s := []wapi.WindowInfoEx{}

	for _, w := range windows.Infos {
		if w.WindowInfo.HasVisibleStyle() && !strings.Contains(w.Title, exe.Title) {
			s = append(s, w)
		}
	}

	Sources = s

	return nil
}
