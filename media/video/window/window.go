package window

import (
	"fmt"
	"image"
	"strings"
	"sync"
	"time"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/system/wapi"
)

var (
	Sources              = []wapi.WindowInfoEx{}
	lastResolution       image.Rectangle
	lastResolutionString = "0x0"

	once = sync.OnceFunc(func() {
		for ; ; time.Sleep(time.Second * 5) {
			err := list()
			if err != nil {
				notify.Error("<ini:f:find> active windows (%v)", err)
			}
		}
	})
)

func Active() wapi.Window {
	for _, w := range Sources {
		if w.Title == config.Current.Video.Capture.Window.Name {
			return w.Window
		}
	}

	return wapi.Window(0)
}

// Capture captures the desired area from a Window and returns an image.
func Capture() (*image.RGBA, error) {
	w := Active()

	r, err := w.Dimensions()
	if err != nil {
		return nil, fmt.Errorf("dimensions: %v", err)
	}

	if !r.Eq(lastResolution) {
		lastResolution = r
		lastResolutionString = fmt.Sprintf("%dx%d", lastResolution.Dx(), lastResolution.Dy())
	}

	return w.Capture(r, config.Current.Scale)
}

func CaptureRect(r image.Rectangle) (*image.RGBA, error) {
	return Active().Capture(r, config.Current.Scale)
}

func IsActive() bool {
	for _, w := range Sources {
		if w.Title == config.Current.Video.Capture.Window.Name {
			return true
		}
	}
	return false
}

func Lost() bool {
	return config.Current.Video.Capture.Window.Lost != ""
}

func Open() error {
	go once()

	err := list()
	if err != nil {
		return err
	}

	if config.Current.Video.Capture.Window.Name == config.MainDisplay {
		return nil
	}

	for _, w := range Sources {
		if w.Title == config.Current.Video.Capture.Window.Name {
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
