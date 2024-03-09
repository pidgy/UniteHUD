package window

import (
	"image"
	"sync"
	"time"
	"unsafe"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/system/wapi"
	"github.com/pkg/errors"
)

var (
	Sources = []string{}

	handles = []uintptr{}
	lock    = &sync.Mutex{}
)

func init() {
	go func() {
		for {
			time.Sleep(time.Second * 5)

			windows, _, err := list()
			if err != nil {
				notify.Warn("Window: Failed to list windows (%v)", err)
				continue
			}

			Sources = windows
		}
	}()
}

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
	src, err := w.Device()
	if err != nil {
		return nil, errors.Wrap(err, "device")
	}
	defer src.Release()

	dst, err := src.Compatible()
	if dst == 0 {
		return nil, errors.Wrap(err, "context")
	}
	defer dst.Delete()

	size := rect.Size()

	info := wapi.BitmapInfo{
		BmiHeader: wapi.BitmapInfoHeader{
			BiSize:        wapi.BitmapInfoHeaderSize,
			BiWidth:       int32(size.X),
			BiHeight:      -int32(size.Y),
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: wapi.BitmapInfoHeaderCompression.RGB,
		},
	}

	bitmap, data, err := info.CreateSection(dst)
	if err != nil {
		return nil, errors.Wrap(err, "section")
	}
	defer bitmap.Delete()

	// Select the object and paint it.
	err = w.Select(bitmap)
	if err != nil {
		return nil, errors.Wrap(err, "bitmap select")
	}

	err = dst.Copy(src, size, rect, config.Current.Scale)
	if err != nil {
		return nil, errors.Wrap(err, "bitmap copy")
	}

	slice := unsafe.Slice(&data, size.X*size.Y*4)

	pix := make([]byte, len(slice))
	for i := 0; i < len(pix); i += 4 {
		pix[i] = byte(slice[i+2])
		pix[i+2] = byte(slice[i])
		pix[i+1] = byte(slice[i+1])
		pix[i+3] = byte(slice[i+3])
	}

	return &image.RGBA{
		Pix:    pix,
		Stride: 4 * size.X,
		Rect:   image.Rectangle{Max: size},
	}, nil
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
		if win == config.Current.Video.Capture.Window.Name {
			config.Current.Video.Capture.Window.Lost = ""
			return nil
		}
	}

	notify.Error("Window: Failed to find \"%s\"", config.Current.Video.Capture.Window.Name)

	config.Current.Video.Capture.Window.Lost = config.Current.Video.Capture.Window.Name
	config.Current.Video.Capture.Window.Name = config.MainDisplay

	return nil
}

func Lost() bool {
	return config.Current.Video.Capture.Window.Lost != ""
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
		if win == config.Current.Video.Capture.Window.Lost {
			config.Current.Video.Capture.Window.Name = win

			notify.Announce("Window: Found \"%s\" window", config.Current.Video.Capture.Window.Name)
			config.Current.Video.Capture.Window.Lost = ""
			attempts = 0

			return nil
		}
	}

	attempts++
	if attempts == max {
		config.Current.Video.Capture.Window.Name = config.MainDisplay
		config.Current.Video.Capture.Window.Lost = ""
		attempts = 0
	}

	return nil
}

func list() ([]string, []uintptr, error) {
	lock.Lock()
	defer lock.Unlock()

	Sources = []string{}

	err := wapi.EnumerateWindows(func(h uintptr, p uintptr) uintptr {
		w := wapi.Window(h)

		if w.Visible() {
			return 1
		}

		name, err := w.Title()
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
	if err != nil {
		return nil, nil, err
	}

	return Sources, handles, nil
}
