//go:build windows
// +build windows

package window

import (
	"fmt"
	"image"
	"reflect"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/nfnt/resize"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/video/monitor"
	"github.com/pidgy/unitehud/video/wapi"
)

var (
	Sources = []string{}

	handles = []syscall.Handle{}
	lock    = &sync.Mutex{}

	callback = syscall.NewCallback(func(h syscall.Handle, p uintptr) uintptr {
		found, _, _ := syscall.Syscall(wapi.IsWindowVisible.Addr(), 1, uintptr(h), uintptr(0), uintptr(0))
		if found == 0 {
			return 1
		}

		name, err := getWindowText(h) //, &b[0], int32(len(b)))
		if err != nil {
			// notify.Error("Failed to find a window title (%v)", err)
			return 1
		}

		if name == config.ProjectorWindow {
			return 1
		}

		Sources = append(Sources, name)
		handles = append(handles, h)

		return 1
	})
)

func init() {
	// We need to call SetProcessDpiAwareness so that Windows API calls will
	// tell us the scale factor for our screen so that our screenshot works
	// on hi-res displays.
	wapi.SetProcessDpiAwareness.Call(uintptr(2)) // PROCESS_PER_MONITOR_DPI_AWARE

	go func() {
		for {
			time.Sleep(time.Second * 5)

			windows, _, err := list()
			if err != nil {
				notify.Error("Failed to list windows (%v)", err)
			}

			Sources = windows
		}
	}()
}

// Capture captures the desired area from a Window and returns an image.
func Capture() (*image.RGBA, error) {
	handle, err := find(config.Current.Window)
	if err != nil {
		notify.Error("Failed to find %s (%v)", config.Current.Window, err)
		if config.Current.LostWindow == "" {
			config.Current.LostWindow = config.Current.Window
		}
		config.Current.Window = config.MainDisplay
		return monitor.Capture()
	}

	// Determine the full width and height of the window.
	rect, err := windowRect(handle)
	if err != nil {
		notify.Error("Failed to find window dimensions \"%s\" (%v)", config.Current.Window, err)
		if config.Current.LostWindow == "" {
			config.Current.LostWindow = config.Current.Window
		}
		config.Current.Window = config.MainDisplay
		return monitor.Capture()
	}

	img, err := CaptureRect(rect)
	if err != nil {
		notify.Error("Failed to capture \"%s\" window (%v)", config.Current.Window, err)
		if config.Current.LostWindow == "" {
			config.Current.LostWindow = config.Current.Window
		}
		config.Current.Window = config.MainDisplay
		return monitor.Capture()
	}

	return img, err
}

func CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
	handle, err := find(config.Current.Window)
	if err != nil {
		notify.Error("%v", err)
		return monitor.CaptureRect(rect)
	}

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
		return nil, fmt.Errorf("Failed to create bitmap for \"%s\" window", config.Current.Window)
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
		notify.Error("Failed to capture \"%s\" window", config.Current.Window)
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

	img := &image.RGBA{
		Pix:    imageBytes,
		Stride: 4 * width,
		Rect: image.Rect(
			0,
			0,
			width,
			height,
		),
	}

	return img, nil
}

func This(title string) (syscall.Handle, error) {
	var handle syscall.Handle

	// First look for the normal window
	ret, _, _ := wapi.FindWindow.Call(0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title))))
	if ret == 0 {
		return handle, fmt.Errorf("Failed to find \"%s\" window", title)
	}

	return syscall.Handle(ret), nil
}

func IsOpen() bool {
	return !Lost()
}

func Open() error {
	windows, _, err := list()
	if err != nil {
		return err
	}

	Sources = windows

	for _, win := range windows {
		if win == config.Current.Window {
			if !monitor.IsDisplay() {
				config.Current.LostWindow = ""
			}
			return nil
		}
	}

	if monitor.IsDisplay() {
		return nil
	}

	notify.Error("\"%s\" could not be found", config.Current.Window)

	config.Current.LostWindow = config.Current.Window
	config.Current.Window = config.MainDisplay

	return nil
}

func Lost() bool {
	return config.Current.LostWindow != ""
}

var attempts = 0

func Reattach() error {
	if !Lost() {
		return nil
	}

	max := 5
	windows, _, err := list()
	if err != nil {
		return err
	}

	for _, win := range windows {
		if win == config.Current.LostWindow {
			config.Current.Window = win

			notify.Announce("Found \"%s\" window", config.Current.Window)
			config.Current.LostWindow = ""
			attempts = 0

			return nil
		}
	}

	attempts++
	if attempts == max {
		config.Current.Window = config.MainDisplay
		config.Current.LostWindow = ""
		attempts = 0
	}

	return nil
}

func enumWindows(enumFunc uintptr, lparam uintptr) (err error) {
	r1, _, e1 := syscall.Syscall(wapi.EnumWindows.Addr(), 2, uintptr(enumFunc), uintptr(lparam), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func list() ([]string, []syscall.Handle, error) {
	lock.Lock()
	defer lock.Unlock()

	Sources = []string{}

	err := enumWindows(callback, 0)
	if err != nil {
		return nil, nil, err
	}

	return Sources, handles, nil
}

// find finds the handle to the window.
func find(name string) (syscall.Handle, error) {
	var handle syscall.Handle

	// First look for the normal window
	argv, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return handle, err
	}

	ret, _, _ := wapi.FindWindow.Call(0, uintptr(unsafe.Pointer(argv)))
	if ret == 0 {
		config.Current.LostWindow = config.Current.Window
		config.Current.Window = config.MainDisplay

		return handle, fmt.Errorf("Failed to find \"%s\"", name)
	}

	return syscall.Handle(ret), nil
}

func getWindowText(hwnd syscall.Handle) (name string, err error) {
	var str *uint16
	b := make([]uint16, 200)
	maxCount := uint32(200)
	str = &b[0]

	r0, _, e1 := syscall.Syscall(wapi.GetWindowTextW.Addr(), 3, uintptr(hwnd), uintptr(unsafe.Pointer(str)), uintptr(maxCount))
	len := int32(r0)
	if len == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}

	return syscall.UTF16ToString(b), err
}

func scaled(img *image.RGBA, scale float64) *image.RGBA {
	x := float64(img.Rect.Max.X) * scale
	return resize.Resize(uint(x), 0, img, resize.Lanczos3).(*image.RGBA)
}

// windowRect gets the dimensions for a Window handle.
func windowRect(hwnd syscall.Handle) (image.Rectangle, error) {
	var rect wapi.Rect
	ret, _, err := wapi.GetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return image.Rectangle{}, fmt.Errorf("Error getting window dimensions: %s", err)
	}

	return image.Rect(0, 0, int(rect.Right), int(rect.Bottom)), nil
}
