package window

import (
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/system/wapi"
)

var (
	Sources              = []string{}
	lock                 = &sync.Mutex{}
	lastResolution       image.Rectangle
	lastResolutionString = "0x0"
)

// Capture captures the desired area from a Window and returns an image.
func Capture() (*image.RGBA, error) {
	w, err := wapi.NewWindow(config.Current.Video.Capture.Window.Name)
	if err != nil {
		return nil, err
	}

	rect, err := w.Dimensions()
	if err != nil {
		return nil, fmt.Errorf("dimensions: %v", err)
	}

	if !rect.Eq(lastResolution) {
		lastResolution = rect
		lastResolutionString = fmt.Sprintf("%dx%d", lastResolution.Dx(), lastResolution.Dy())
	}

	return CaptureRect(w, rect)
}

func CaptureRect(w wapi.Window, rect image.Rectangle) (*image.RGBA, error) {
	src, err := w.Device()
	if err != nil {
		return nil, fmt.Errorf("invalid window handle: %v", err)
	}
	defer src.Release()

	dst, err := src.CreateCompatible()
	if err != nil {
		return nil, fmt.Errorf("%s: create compatible device: %v", w, err)
	}
	defer dst.Delete()

	bitmap, bitvals, err := dst.CreateDIBSection(rect.Size())
	if err != nil {
		return nil, fmt.Errorf("%s: create bitmap section: %v", w, err)
	}
	defer bitmap.Delete()

	_, err = dst.Select(bitmap)
	if err != nil {
		return nil, fmt.Errorf("%s: bitmap select: %v", w, err)
	}

	if config.Current.Scale == 1 {
		err = dst.BitBlt(src, rect.Size(), rect.Min)
	} else {
		err = dst.StretchBlt(src, rect.Size(), rect.Min, config.Current.Scale)
	}
	if err != nil {
		return nil, fmt.Errorf("bit-block transfer (scale=%.2fx): %v", config.Current.Scale, err)
	}

	return bitvals.Image(rect.Size()), nil
}

func IsOpen() bool {
	for _, s := range Sources {
		if config.Current.Video.Capture.Window.Name == s {
			return true
		}
	}
	return false
}

func Lost() bool {
	return config.Current.Video.Capture.Window.Lost != ""
}

func Open() error {
	go sync.OnceFunc(func() {
		for ; ; time.Sleep(time.Second * 5) {
			err := list()
			if err != nil {
				notify.Error("<ini:f:find> active windows (%v)", err)
			}
		}
	})

	err := list()
	if err != nil {
		return err
	}

	if config.Current.Video.Capture.Window.Name == config.MainDisplay {
		return nil
	}

	for _, win := range Sources {
		if win == config.Current.Video.Capture.Window.Name {
			config.Current.Video.Capture.Window.Lost = ""
			return nil
		}
	}

	notify.Error("[Window] <ini:f:find> \"%s\"", config.Current.Video.Capture.Window.Name)

	config.Current.Video.Capture.Window.Lost = config.Current.Video.Capture.Window.Name
	config.Current.Video.Capture.Window.Name = config.MainDisplay

	return nil
}

func Resolution() string {
	return lastResolutionString
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

func list() error {
	lock.Lock()
	defer lock.Unlock()

	windows, err := wapi.GetAllWindows()
	if err != nil {
		return err
	}

	Sources = []string{}
	for _, w := range windows.Infos {
		if w.WindowInfo.HasVisibleStyle() {
			Sources = append(Sources, w.Title)
		}
	}

	return nil
}
