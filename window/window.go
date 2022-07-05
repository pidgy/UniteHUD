//go:build windows
// +build windows

package window

import (
	"fmt"
	"image"
	"reflect"
	"sync"
	"syscall"
	"unsafe"

	"github.com/disintegration/gift"
	"github.com/rs/zerolog/log"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
)

var (
	Open = []string{config.MainDisplay}

	handles = []syscall.Handle{}
	lock    = &sync.Mutex{}
)

func init() {
	// We need to call SetProcessDpiAwareness so that Windows API calls will
	// tell us the scale factor for our monitor so that our screenshot works
	// on hi-res displays.
	procSetProcessDpiAwareness.Call(uintptr(2)) // PROCESS_PER_MONITOR_DPI_AWARE
}

func Capture() (*image.RGBA, error) {
	if config.Current.Window == config.MainDisplay {
		return captureScreen()
	}

	return captureWindow()
}

func CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
	if config.Current.Window == config.MainDisplay {
		return captureScreenRect(rect)
	}

	return captureWindowRect(rect)
}

func Load() error {
	windows, _, err := list()
	if err != nil {
		return err
	}

	Open = windows

	for _, win := range windows {
		if win == config.Current.Window {
			if config.Current.Window != config.MainDisplay {
				config.Current.LostWindow = ""
			}
			return nil
		}
	}

	notify.Feed(rgba.Red, "\"%s\" could not be found", config.Current.Window)

	config.Current.LostWindow = config.Current.Window
	config.Current.Window = config.MainDisplay

	return nil
}

func Reattach() error {
	windows, _, err := list()
	if err != nil {
		return err
	}

	log.Debug().Str("lost", config.Current.LostWindow).Strs("windows", windows).Msg("reattaching window")

	for _, win := range windows {
		if win == config.Current.LostWindow {
			config.Current.Window = win
			if config.Current.Window != config.MainDisplay {
				config.Current.LostWindow = ""
			}

			notify.Feed(rgba.Seafoam, "Found \"%s\" window", config.Current.Window)

			return nil
		}
	}

	return nil
}

func Resize16x9() error {
	h, err := findWindow(config.Current.Window)
	if err != nil {
		return err
	}
	ret, _, _ := procMoveWindow.Call(
		uintptr(h),
		uintptr(0),
		uintptr(0),
		uintptr(1920),
		uintptr(1080),
		uintptr(1),
	)
	if ret == 0 {
		return fmt.Errorf("failed to resize \"%s\"", config.Current.Window)
	}
	return nil
}

var callback = syscall.NewCallback(func(h syscall.Handle, p uintptr) uintptr {
	found, _, _ := syscall.Syscall(procIsWindowVisible.Addr(), 1, uintptr(h), uintptr(0), uintptr(0))
	if found == 0 {
		return 1
	}

	b := make([]uint16, 200)
	_, err := getWindowText(h, &b[0], int32(len(b)))
	if err != nil {
		return 1 // continue enumeration
	}

	Open = append(Open, syscall.UTF16ToString(b))
	handles = append(handles, h)

	return 1 // continue enumeration
})

func list() ([]string, []syscall.Handle, error) {
	lock.Lock()
	defer lock.Unlock()

	Open = []string{}

	err := enumWindows(callback, 0)
	if err != nil {
		return nil, nil, err
	}

	return Open, handles, nil
}

// captureWindow captures the desired area from a Window and returns an image.
func captureWindow() (*image.RGBA, error) {
	handle, err := findWindow(config.Current.Window)
	if err != nil {
		return captureScreen()
	}

	// Determine the full width and height of the window.
	rect, err := windowRect(handle)
	if err != nil {
		config.Current.Window = config.MainDisplay
		notify.Feed(rgba.Red, "%v", err)
		return captureScreen()
	}

	img, err := captureWindowRect(rect)
	if err != nil {
		config.Current.Window = config.MainDisplay
		notify.Feed(rgba.Red, "%v", err)
		return captureScreen()
	}

	return img, err
}

func captureWindowRect(rect image.Rectangle) (*image.RGBA, error) {
	handle, err := findWindow(config.Current.Window)
	if err != nil {
		return captureScreenRect(rect)
	}

	// Get the device context for screenshotting
	dcSrc, _, err := procGetDC.Call(uintptr(handle))
	if dcSrc == 0 {
		return nil, fmt.Errorf("failed to prepare screen capture: %s", err)
	}
	defer procReleaseDC.Call(0, dcSrc)

	// Grab a compatible DC for drawing
	dcDst, _, err := procCreateCompatibleDC.Call(dcSrc)
	if dcDst == 0 {
		return nil, fmt.Errorf("failed to create DC for drawing: %s", err)
	}
	defer procDeleteDC.Call(dcDst)

	// Determine the width/height of our capture
	width := rect.Dx()
	height := rect.Dy()

	// Get the bitmap we're going to draw onto
	var bitmapInfo windowsBitmapInfo
	bitmapInfo.BmiHeader = windowsBitmapInfoHeader{
		BiSize:        uint32(reflect.TypeOf(bitmapInfo.BmiHeader).Size()),
		BiWidth:       int32(width),
		BiHeight:      int32(height),
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: 0, // BI_RGB
	}

	bitmapData := unsafe.Pointer(uintptr(0))
	bitmap, _, err := procCreateDIBSection.Call(
		dcDst,
		uintptr(unsafe.Pointer(&bitmapInfo)),
		0,
		uintptr(unsafe.Pointer(&bitmapData)), 0, 0)
	if bitmap == 0 {
		return nil, fmt.Errorf("error creating bitmap for screen capture: %s", err)
	}
	defer procDeleteObject.Call(bitmap)

	// Select the object and paint it
	procSelectObject.Call(dcDst, bitmap)
	ret, _, err := procBitBlt.Call(
		dcDst, 0, 0, uintptr(width), uintptr(height),
		dcSrc, uintptr(rect.Min.X), uintptr(rect.Min.Y), bitBlt_SRCCOPY)
	if ret == 0 {
		return nil, fmt.Errorf("error capturing screen: %s", err)
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
		imageBytes[i], imageBytes[i+2], imageBytes[i+1], imageBytes[i+3] = slice[i+2], slice[i], slice[i+1], slice[i+3]
	}

	// The window gets screenshotted upside down and I don't know why.
	// Flip it.
	img := &image.RGBA{imageBytes, 4 * width, image.Rect(0, 0, width, height)}
	dst := image.NewRGBA(img.Bounds())
	gift.New(gift.FlipVertical()).Draw(dst, img)

	return dst, nil
}

func enumWindows(enumFunc uintptr, lparam uintptr) (err error) {
	r1, _, e1 := syscall.Syscall(procEnumWindows.Addr(), 2, uintptr(enumFunc), uintptr(lparam), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

// findWindow finds the handle to the window.
func findWindow(name string) (syscall.Handle, error) {
	var handle syscall.Handle

	// First look for the normal window
	// ret, _, _ := procFindWindow.Call(0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("AppName"))))
	ret, _, _ := procFindWindow.Call(0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(name))))
	if ret == 0 {
		err := fmt.Errorf("Failed to find \"%s\"", config.Current.Window)

		config.Current.LostWindow = config.Current.Window
		config.Current.Window = config.MainDisplay

		notify.Feed(rgba.PaleRed, err.Error())

		return handle, err
	}

	return syscall.Handle(ret), nil
}

func getWindowText(hwnd syscall.Handle, str *uint16, maxCount int32) (len int32, err error) {
	r0, _, e1 := syscall.Syscall(procGetWindowTextW.Addr(), 3, uintptr(hwnd), uintptr(unsafe.Pointer(str)), uintptr(maxCount))
	len = int32(r0)
	if len == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

// windowRect gets the dimensions for a Window handle.
func windowRect(hwnd syscall.Handle) (image.Rectangle, error) {
	var rect windowsRect
	ret, _, err := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return image.Rectangle{}, fmt.Errorf("Error getting window dimensions: %s", err)
	}

	return image.Rect(0, 0, int(rect.Right), int(rect.Bottom)), nil
}
