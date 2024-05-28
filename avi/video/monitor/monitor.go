package monitor

import (
	"fmt"
	"image"
	"sync"

	"github.com/kbinani/screenshot"
	"github.com/pkg/errors"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/system/wapi"
)

var (
	MainResolution = image.Rect(0, 0, 1920, 1080)
	Sources        = []string{}

	displays = map[string]int{}
	bounds   = map[string]image.Rectangle{}

	mutex = &sync.RWMutex{}
)

func Active(name string) bool {
	return IsDisplay() && name == config.Current.Video.Capture.Window.Name
}

func Bounds() image.Rectangle {
	mutex.RLock()
	defer mutex.RUnlock()
	b := bounds[config.Current.Video.Capture.Window.Name]
	return b
}

func BoundsOf(d string) image.Rectangle {
	mutex.RLock()
	defer mutex.RUnlock()
	b := bounds[d]
	return b
}

func Capture() (*image.RGBA, error) {
	return CaptureRect(dims())
}

func CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
	b := dims()

	rect.Min.X = b.Min.X + rect.Min.X
	rect.Max.X = b.Min.X + rect.Max.X

	rect.Min.Y = b.Min.Y + rect.Min.Y
	rect.Max.Y = b.Min.Y + rect.Max.Y

	src, err := wapi.Window(0).Device()
	if err != nil {
		return nil, errors.Wrap(err, "device")
	}
	defer src.Release()

	dst, err := src.Compatible()
	if err != nil {
		return nil, fmt.Errorf("could not create compatible DC (%d)", getLastError())
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

	bitmap, raw, err := info.CreateRGBSection(&dst)
	if err != nil {
		return nil, errors.Wrap(err, "section")
	}
	defer bitmap.Delete()

	obj, err := dst.Select(bitmap)
	if err != nil {
		return nil, errors.Wrap(err, "bitmap select")
	}
	defer obj.Delete()

	err = dst.Copy(src, size, rect, config.Current.Scale)
	if err != nil {
		return nil, errors.Wrap(err, "bitmap copy")
	}

	data := raw.Slice(size)

	pix := make([]byte, len(data))

	for i := 0; i < len(pix); i += 4 {
		pix[i], pix[i+2], pix[i+1], pix[i+3] = byte(data[i+2]), byte(data[i]), byte(data[i+1]), byte(data[i+3])
	}

	return &image.RGBA{
		Pix:    pix,
		Stride: 4 * size.X,
		Rect:   image.Rect(0, 0, size.X, size.Y),
	}, nil
}

func IsDisplay() bool {
	mutex.RLock()
	defer mutex.RUnlock()

	_, ok := displays[config.Current.Video.Capture.Window.Name]
	return ok
}

func Open() {
	sourcesTmp := []string{}
	displaysTmp := map[string]int{}
	boundsTmp := map[string]image.Rectangle{}

	leftDisplays := 0
	rightDisplays := 0
	topDisplays := 0
	bottomDisplays := 0

	m := MainResolution

	for i := 0; i < screenshot.NumActiveDisplays(); i++ {
		name := ""

		r := screenshot.GetDisplayBounds(i)
		switch {
		case r.Eq(m):
			name = config.MainDisplay
		case r.Min.X < m.Min.X:
			leftDisplays++
			name = display("Left Display", leftDisplays)
		case r.Min.X > m.Min.X:
			rightDisplays++
			name = display("Right Display", rightDisplays)
		case r.Min.Y < m.Min.Y:
			topDisplays++
			name = display("Top Display", topDisplays)
		case r.Min.Y > m.Min.Y:
			bottomDisplays++
			name = display("Bottom Display", bottomDisplays)
		default:
			notify.Error("[Video] Failed to locate display #%d [%s] relative to %s [%s]", i, r, config.MainDisplay, m)
			continue
		}

		displaysTmp[name] = i
		boundsTmp[name] = r
		sourcesTmp = append(sourcesTmp, name)
	}

	set(sourcesTmp, displaysTmp, boundsTmp)
}

func createCompatibleDC(hdc uintptr) uintptr {
	ret, _, _ := wapi.CreateCompatibleDC.Call(uintptr(hdc))
	return ret
}

func deleteObject(hObject uintptr) bool {
	ret, _, _ := wapi.DeleteObject.Call(hObject)
	return ret != 0
}

func dims() image.Rectangle {
	mutex.RLock()
	defer mutex.RUnlock()

	b := bounds[config.Current.Video.Capture.Window.Name]
	return b
}

func display(name string, count int) string {
	if count <= 1 {
		return name
	}
	return fmt.Sprintf("%s %d", name, count)
}

func getDC(hwnd uintptr) uintptr {
	ret, _, _ := wapi.GetDC.Call(uintptr(hwnd))
	return ret
}

func getLastError() uint32 {
	ret, _, _ := wapi.GetLastError.Call()
	return uint32(ret)
}

func releaseDC(hwnd uintptr, hdc uintptr) bool {
	ret, _, _ := wapi.ReleaseDC.Call(uintptr(hwnd), uintptr(hdc))
	return ret != 0
}

func selectObject(hdc, hgdiobj uintptr) uintptr {
	ret, _, _ := wapi.SelectObject.Call(uintptr(hdc), uintptr(hgdiobj))
	return ret
}

func set(s []string, d map[string]int, b map[string]image.Rectangle) {
	mutex.Lock()
	defer mutex.Unlock()

	Sources = s
	displays = d
	bounds = b
}
