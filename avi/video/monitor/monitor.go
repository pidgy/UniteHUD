package monitor

import (
	"fmt"
	"image"
	"sync"

	"github.com/kbinani/screenshot"
	"golang.org/x/image/draw"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/system/wapi"
)

var (
	DefaultResolution = image.Rect(0, 0, 1920, 1080)

	Sources  = []string{}
	displays = new(sync.Map)
)

func Active(name string) bool {
	return IsDisplay() && name == config.Current.Video.Capture.Window.Name
}

func Capture() (*image.RGBA, error) {
	return captureFullscreen()
}

func CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
	img, err := captureFullscreen()
	if err != nil {
		return nil, err
	}

	return img.SubImage(rect).(*image.RGBA), nil
}

// func CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
// mutex.RLock()
// b := bounds[config.Current.Video.Capture.Window.Name]
// mutex.RUnlock()

// rect.Min.X = b.Min.X + rect.Min.X
// rect.Max.X = b.Min.X + rect.Max.X

// rect.Min.Y = b.Min.Y + rect.Min.Y
// rect.Max.Y = b.Min.Y + rect.Max.Y

// src, err := wapi.Window(0).Device()
// if err != nil {
// 	return nil, errors.Wrap(err, "device")
// }
// defer src.Release()

// dst, err := src.Compatible()
// if err != nil {
// 	return nil, fmt.Errorf("could not create compatible DC (%d)", lastError())
// }
// defer dst.Delete()

// size := rect.Size()

// info := wapi.BitmapInfo{
// 	BmiHeader: wapi.BitmapInfoHeader{
// 		BiSize:        wapi.BitmapInfoHeaderSize,
// 		BiWidth:       int32(size.X),
// 		BiHeight:      -int32(size.Y),
// 		BiPlanes:      1,
// 		BiBitCount:    32,
// 		BiCompression: wapi.BitmapInfoHeaderCompression.RGB,
// 	},
// }

// bitmap, raw, err := info.CreateRGBSection(&dst)
// if err != nil {
// 	return nil, errors.Wrap(err, "section")
// }
// defer bitmap.Delete()

// obj, err := dst.Select(bitmap)
// if err != nil {
// 	return nil, errors.Wrap(err, "bitmap select")
// }
// defer obj.Delete()

// err = dst.Copy(src, size, rect, config.Current.Scale)
// if err != nil {
// 	return nil, errors.Wrap(err, "bitmap copy")
// }

// data := raw.Slice(size)
// pix := make([]byte, len(data))

// for i := 0; i < len(pix); i += 4 {
// 	pix[i], pix[i+2], pix[i+1], pix[i+3] = byte(data[i+2]), byte(data[i]), byte(data[i+1]), byte(data[i+3])
// }

// return &image.RGBA{
// 	Pix:    pix,
// 	Stride: 4 * size.X,
// 	Rect:   image.Rect(0, 0, size.X, size.Y),
// }, nil
//}

func TaskbarHeight() int {
	r, err := wapi.WorkArea()
	if err != nil {
		notify.Error("[Video] Failed to find monitor info: %v", err)
		return 0
	}

	return Resolution().Max.Y - int(r.Bottom)
}

func IsDisplay() bool {
	_, ok := displays.Load(config.Current.Video.Capture.Window.Name)
	return ok
}

func Resolution() image.Rectangle {
	if IsDisplay() {
		n, ok := displays.Load(config.Current.Video.Capture.Window.Name)
		if !ok {
			return DefaultResolution
		}

		return screenshot.GetDisplayBounds(n.(int))
	}
	return screenshot.GetDisplayBounds(0)
}

func Open() {
	sourcesTmp := []string{}
	displaysTmp := new(sync.Map)

	leftDisplays := 0
	rightDisplays := 0
	topDisplays := 0
	bottomDisplays := 0

	m := DefaultResolution

	for i := 0; i < screenshot.NumActiveDisplays(); i++ {
		name := ""
		r := screenshot.GetDisplayBounds(i)

		switch {
		case r.Eq(m):
			name = config.MainDisplay
		case i == 0 && r.Dx() > m.Dx() && r.Dy() > m.Dy():
			notify.Warn("[Video] Rescaling display #%d from %s to %s", i, r, m)
			name = config.MainDisplay
		case r.Min.X < m.Min.X:
			leftDisplays++
			name = display("Left", leftDisplays)
		case r.Min.X > m.Min.X:
			rightDisplays++
			name = display("Right", rightDisplays)
		case r.Min.Y < m.Min.Y:
			topDisplays++
			name = display("Top", topDisplays)
		case r.Min.Y > m.Min.Y:
			bottomDisplays++
			name = display("Bottom", bottomDisplays)
		default:
			notify.Error("[Video] Failed to locate display #%d [%s] relative to %s [%s]", i, r, config.MainDisplay, m)
			continue
		}

		displaysTmp.Store(name, i)
		sourcesTmp = append(sourcesTmp, name)
	}

	Sources = sourcesTmp
	displays = displaysTmp
}

func captureFullscreen() (*image.RGBA, error) {
	n, ok := displays.Load(config.Current.Video.Capture.Window.Name)
	if !ok {
		return nil, fmt.Errorf("%s: failed to find display", config.Current.Video.Capture.Window.Name)
	}

	img, err := screenshot.CaptureDisplay(n.(int))
	if err != nil {
		return nil, err
	}

	if img.Rect.Max.X > DefaultResolution.Max.X && img.Rect.Max.Y > DefaultResolution.Max.Y {
		scaled := image.NewRGBA(DefaultResolution)
		draw.NearestNeighbor.Scale(scaled, scaled.Rect, img, img.Bounds(), draw.Over, &draw.Options{})
		return scaled, nil
	}

	return img, nil

}

func display(name string, count int) string {
	if count <= 1 {
		return name
	}
	return fmt.Sprintf("%s #%d", name, count)
}
