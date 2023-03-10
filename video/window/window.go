//go:build windows
// +build windows

package window

import (
	"fmt"
	"image"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/disintegration/gift"
	"github.com/nfnt/resize"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/video/proc"
	"github.com/pidgy/unitehud/video/screen"
	"github.com/pidgy/unitehud/video/window/electron"
)

var (
	Sources = []string{}

	handles = []syscall.Handle{}
	lock    = &sync.Mutex{}

	callback = syscall.NewCallback(func(h syscall.Handle, p uintptr) uintptr {
		found, _, _ := syscall.Syscall(proc.IsWindowVisible.Addr(), 1, uintptr(h), uintptr(0), uintptr(0))
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
	proc.SetProcessDpiAwareness.Call(uintptr(2)) // PROCESS_PER_MONITOR_DPI_AWARE

	go func() {
		for range time.NewTicker(time.Second * 5).C {
			windows, _, err := list()
			if err != nil {

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
		return screen.Capture()
	}

	// Determine the full width and height of the window.
	rect, err := windowRect(handle)
	if err != nil {
		notify.Error("Failed to find window dimensions \"%s\" (%v)", config.Current.Window, err)
		if config.Current.LostWindow == "" {
			config.Current.LostWindow = config.Current.Window
		}
		config.Current.Window = config.MainDisplay
		return screen.Capture()
	}

	img, err := CaptureRect(rect)
	if err != nil {
		notify.Error("Failed to capture \"%s\" window (%v)", config.Current.Window, err)
		if config.Current.LostWindow == "" {
			config.Current.LostWindow = config.Current.Window
		}
		config.Current.Window = config.MainDisplay
		return screen.Capture()
	}

	return img, err
}

func CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
	handle, err := find(config.Current.Window)
	if err != nil {
		notify.Error("%v", err)
		return screen.CaptureRect(rect)
	}

	// Get the device context for screenshotting.
	src, _, err := proc.GetDC.Call(uintptr(handle))
	if src == 0 {
		return nil, fmt.Errorf("failed to prepare screen capture: %s", err)
	}
	defer proc.ReleaseDC.Call(0, src)

	// Grab a compatible DC for drawing.
	dst, _, err := proc.CreateCompatibleDC.Call(src)
	if dst == 0 {
		return nil, fmt.Errorf("failed to create DC for drawing: %s", err)
	}
	defer proc.DeleteDC.Call(dst)

	// Determine the width/height of our capture
	width := rect.Dx()
	height := rect.Dy()

	// Get the bitmap we're going to draw onto.
	var bitmapInfo proc.WindowsBitmapInfo
	bitmapInfo.BmiHeader = proc.WindowsBitmapInfoHeader{
		BiSize:        uint32(reflect.TypeOf(bitmapInfo.BmiHeader).Size()),
		BiWidth:       int32(width),
		BiHeight:      int32(height),
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: proc.BIRGBCompression,
	}

	bitmapData := unsafe.Pointer(uintptr(0))
	bitmap, _, err := proc.CreateDIBSection.Call(
		dst,
		uintptr(unsafe.Pointer(&bitmapInfo)),
		0,
		uintptr(unsafe.Pointer(&bitmapData)),
		0, 0,
	)
	if bitmap == 0 {
		return nil, fmt.Errorf("Failed to create bitmap for \"%s\" window", config.Current.Window)
	}

	defer proc.DeleteObject.Call(bitmap)

	// Select the object and paint it.
	proc.SelectObject.Call(dst, bitmap)

	ret, _, err := proc.BitBlt.Call(dst, 0, 0,
		uintptr(width), uintptr(height),
		src,
		uintptr(rect.Min.X), uintptr(rect.Min.Y),
		uintptr(proc.CaptureBLT|proc.SrcCopy),
	)
	if ret == 0 {
		notify.Error("Failed to capture \"%s\" window", config.Current.Window)
		return nil, err
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

	img2 := image.NewRGBA(img.Bounds())

	if config.Current.Window == config.BrowserWindow {
		gift.New(
			gift.FlipVertical(),
			/*
				gift.ResizeToFill(1920, 1080, gift.CubicResampling, gift.TopLeftAnchor),
				gift.CropToSize(1920, 1080, gift.TopLeftAnchor),
			*/
		).Draw(img2, img)

		return img2, nil
		//return resize.Resize(uint(float64(img.Rect.Max.X)*1.3), uint(float64(img.Rect.Max.Y)*1.3), img2, resize.Lanczos3).(*image.RGBA), nil
	} else if config.Current.Scale == 1 {
		gift.New(
			gift.FlipVertical(),
		).Draw(img2, img)

		return img2, nil
	}

	/*
		r := img.Rect
		r.Min = r.Min.Add(image.Pt(config.Current.Shift.W, config.Current.Shift.S))
		r.Max = r.Max.Add(image.Pt(0, config.Current.Shift.S))
		r.Max = r.Max.Sub(image.Pt(config.Current.Shift.E, 0))
	*/

	// img.Rect.Max.X += config.Current.Shift.E

	scaledW := int(float64(width) * config.Current.Scale)
	scaledH := int(float64(height) * config.Current.Scale)

	gift.New(
		gift.FlipVertical(),
		gift.ResizeToFill(scaledW, scaledH, gift.LanczosResampling, gift.CenterAnchor),
		//gift.Resize(scaledW, scaledH, gift.LanczosResampling),
	).Draw(img2, img)

	img3 := image.NewRGBA(img2.Bounds())
	gift.New(
		gift.CropToSize(width, height, gift.CenterAnchor),
	).Draw(img3, img2)

	return img3, nil
}

func IsWindow() bool {
	return !Lost()
}

func Open() error {
	if config.Current.Window == config.BrowserWindow {
		err := electron.Open()
		if err != nil {
			notify.Error("%v", err)
		}
	}

	windows, _, err := list()
	if err != nil {
		return err
	}

	Sources = windows

	for _, win := range windows {
		if win == config.Current.Window {
			if !screen.IsDisplay() {
				config.Current.LostWindow = ""
			}
			return nil
		}
	}

	if screen.IsDisplay() {
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

const max = 5

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

func Resize16x9() error {
	h, err := find(config.Current.Window)
	if err != nil {
		return err
	}
	ret, _, _ := proc.MoveWindow.Call(
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

func StartingWith(name string) error {
	windows, _, err := list()
	if err != nil {
		return err
	}

	for _, w := range windows {
		if strings.HasPrefix(w, name) {
			return nil
		}
	}

	return fmt.Errorf("Failed to find window starting with \"%s\"", name)
}

func enumWindows(enumFunc uintptr, lparam uintptr) (err error) {
	r1, _, e1 := syscall.Syscall(proc.EnumWindows.Addr(), 2, uintptr(enumFunc), uintptr(lparam), 0)
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
	// ret, _, _ := proc.FindWindow.Call(0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("AppName"))))
	ret, _, _ := proc.FindWindow.Call(0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(name))))
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

	r0, _, e1 := syscall.Syscall(proc.GetWindowTextW.Addr(), 3, uintptr(hwnd), uintptr(unsafe.Pointer(str)), uintptr(maxCount))
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
	var rect proc.WindowsRect
	ret, _, err := proc.GetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return image.Rectangle{}, fmt.Errorf("Error getting window dimensions: %s", err)
	}

	return image.Rect(0, 0, int(rect.Right), int(rect.Bottom)), nil
}
